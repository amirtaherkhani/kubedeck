package management

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	defaultForwardLifetime  = 5 * time.Minute
	defaultForwardIdle      = 30 * time.Second
	defaultForwardByteLimit = 64 << 20
	maxForwardFrame         = 64 << 10
)

// PortForwardRunner returns one connected stream to a Pod port. The HTTP
// boundary is tested with net.Pipe without opening a real Kubernetes tunnel.
type PortForwardRunner func(context.Context, string, string, uint16) (net.Conn, error)

func (m *Manager) portForwardPod(w http.ResponseWriter, r *http.Request) {
	namespace, name := r.PathValue("namespace"), r.PathValue("name")
	portNumber, err := strconv.ParseUint(r.URL.Query().Get("port"), 10, 16)
	if !segment.MatchString(namespace) || !segment.MatchString(name) || err != nil || portNumber == 0 {
		http.Error(w, "invalid Pod identity or port", http.StatusBadRequest)
		return
	}
	port := uint16(portNumber)
	defer m.audit(r, "pods", namespace, name, "port-forward")
	if r.Header.Get("X-KuchDesk-Confirm") != fmt.Sprintf("pods/%s/%s/port-forward/%d", namespace, name, port) || r.Header.Get("If-Match-UID") == "" || r.Header.Get("If-Match") == "" {
		http.Error(w, "exact confirmation, Pod UID and resourceVersion required", http.StatusConflict)
		return
	}
	if !websocket.IsWebSocketUpgrade(r) {
		http.Error(w, "WebSocket upgrade required", http.StatusBadRequest)
		return
	}
	if r.Header.Get("Origin") != "" {
		http.Error(w, "browser origins are not accepted", http.StatusForbidden)
		return
	}
	pod, err := m.Kube.CoreV1().Pods(namespace).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, err)
		return
	}
	if string(pod.UID) != r.Header.Get("If-Match-UID") || pod.ResourceVersion != r.Header.Get("If-Match") || pod.Status.Phase != corev1.PodRunning {
		http.Error(w, "Pod identity or state changed", http.StatusConflict)
		return
	}
	m.forwardOnce.Do(func() { m.forwardSlots = make(chan struct{}, 4) })
	select {
	case m.forwardSlots <- struct{}{}:
		defer func() { <-m.forwardSlots }()
	default:
		http.Error(w, "port-forward capacity reached", http.StatusTooManyRequests)
		return
	}
	lifetime, idle, byteLimit := m.forwardLifetime, m.forwardIdle, m.forwardByteLimit
	if lifetime <= 0 {
		lifetime = defaultForwardLifetime
	}
	if idle <= 0 {
		idle = defaultForwardIdle
	}
	if byteLimit <= 0 {
		byteLimit = defaultForwardByteLimit
	}
	ctx, cancel := context.WithTimeout(r.Context(), lifetime)
	defer cancel()
	run := m.ForwardRun
	if run == nil {
		run = m.runPodForward
	}
	backend, err := run(ctx, namespace, name, port)
	if err != nil || backend == nil {
		http.Error(w, "Pod port-forward failed", http.StatusBadGateway)
		return
	}
	defer backend.Close()
	upgrader := websocket.Upgrader{
		ReadBufferSize: maxForwardFrame, WriteBufferSize: maxForwardFrame,
		CheckOrigin: func(request *http.Request) bool { return request.Header.Get("Origin") == "" },
	}
	w.Header().Set("Cache-Control", "no-store")
	client, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer client.Close()
	client.SetReadLimit(maxForwardFrame)
	var workers sync.WaitGroup
	done := make(chan struct{}, 2)
	for _, pump := range []func(){
		func() { pumpClientToPod(client, backend, idle, byteLimit) },
		func() { pumpPodToClient(backend, client, idle, byteLimit) },
	} {
		workers.Add(1)
		go func() {
			defer workers.Done()
			pump()
			done <- struct{}{}
		}()
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
	_ = backend.Close()
	_ = client.Close()
	workers.Wait()
}

func pumpClientToPod(client *websocket.Conn, backend net.Conn, idle time.Duration, byteLimit int64) {
	var transferred int64
	for {
		_ = client.SetReadDeadline(time.Now().Add(idle))
		kind, payload, err := client.ReadMessage()
		if err != nil || kind != websocket.BinaryMessage || transferred+int64(len(payload)) > byteLimit {
			return
		}
		transferred += int64(len(payload))
		_ = backend.SetWriteDeadline(time.Now().Add(idle))
		if _, err := io.Copy(backend, bytes.NewReader(payload)); err != nil {
			return
		}
	}
}

func pumpPodToClient(backend net.Conn, client *websocket.Conn, idle time.Duration, byteLimit int64) {
	buffer := make([]byte, 32<<10)
	var transferred int64
	for {
		_ = backend.SetReadDeadline(time.Now().Add(idle))
		count, err := backend.Read(buffer)
		if count > 0 {
			if transferred+int64(count) > byteLimit {
				return
			}
			transferred += int64(count)
			_ = client.SetWriteDeadline(time.Now().Add(idle))
			if client.WriteMessage(websocket.BinaryMessage, buffer[:count]) != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (m *Manager) runPodForward(ctx context.Context, namespace, name string, port uint16) (net.Conn, error) {
	if m.RESTConfig == nil || m.Kube == nil {
		return nil, errors.New("Kubernetes REST configuration unavailable")
	}
	config := *m.RESTConfig
	if config.Timeout <= 0 || config.Timeout > 10*time.Second {
		config.Timeout = 10 * time.Second
	}
	request := m.Kube.CoreV1().RESTClient().Post().Resource("pods").Namespace(namespace).Name(name).SubResource("portforward").VersionedParams(&corev1.PodPortForwardOptions{Ports: []int32{int32(port)}}, scheme.ParameterCodec)
	websocketDialer, err := portforward.NewSPDYOverWebsocketDialer(request.URL(), &config)
	if err != nil {
		return nil, err
	}
	transport, upgrader, err := spdy.RoundTripperFor(&config)
	if err != nil {
		return nil, err
	}
	spdyDialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport, Timeout: config.Timeout}, http.MethodPost, request.URL())
	dialer := portforward.NewFallbackDialer(websocketDialer, spdyDialer, httpstream.IsUpgradeFailure)
	stop, ready, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	forwarder, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{fmt.Sprintf("0:%d", port)}, stop, ready, io.Discard, io.Discard)
	if err != nil {
		return nil, err
	}
	go func() { finished <- forwarder.ForwardPorts() }()
	select {
	case <-ctx.Done():
		close(stop)
		return nil, ctx.Err()
	case err := <-finished:
		close(stop)
		if err == nil {
			err = errors.New("port-forward ended before becoming ready")
		}
		return nil, err
	case <-ready:
	}
	ports, err := forwarder.GetPorts()
	if err != nil || len(ports) != 1 {
		close(stop)
		return nil, errors.New("local port-forward unavailable")
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(int(ports[0].Local)))
	connection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		close(stop)
		return nil, err
	}
	return &localForwardConnection{Conn: connection, stop: stop}, nil
}

type localForwardConnection struct {
	net.Conn
	stop chan struct{}
	once sync.Once
}

func (connection *localForwardConnection) Close() error {
	connection.once.Do(func() { close(connection.stop) })
	return connection.Conn.Close()
}
