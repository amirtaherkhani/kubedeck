package management

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/httpapi"
	"github.com/gorilla/websocket"
)

const forwardPath = "/v1/manage/pods/apps/worker/port-forward?port=8080"

func forwardHeaders() http.Header {
	header := make(http.Header)
	header.Set("Authorization", "Bearer test-token")
	header.Set("X-KuchDesk-Confirm", "pods/apps/worker/port-forward/8080")
	header.Set("If-Match-UID", "pod-uid")
	header.Set("If-Match", "7")
	return header
}

func forwardURL(server *httptest.Server, path string) string {
	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	u.Path, u.RawQuery, _ = strings.Cut(path, "?")
	return u.String()
}

func startForwardTestServer(t *testing.T, manager *Manager) *httptest.Server {
	t.Helper()
	api := httpapi.New(nil, nil, nil, "test-token", time.Second, nil)
	api.SetManagementHandler(manager.Handler())
	server := httptest.NewServer(api.Handler())
	t.Cleanup(server.Close)
	return server
}

func TestPortForwardOffByDefaultAndRejectsInvalidRequests(t *testing.T) {
	m := readyExecManager(t)
	m.ExecEnabled = false
	m.PortForwardEnabled = false
	var calls atomic.Int32
	m.ForwardRun = func(context.Context, string, string, uint16) (net.Conn, error) {
		calls.Add(1)
		return nil, nil
	}
	server := startForwardTestServer(t, m)
	if _, response, err := websocket.DefaultDialer.Dial(forwardURL(server, forwardPath), forwardHeaders()); err == nil || response.StatusCode != http.StatusNotFound {
		t.Fatalf("disabled forward response: %v %v", response, err)
	}
	m.PortForwardEnabled = true
	server = startForwardTestServer(t, m)
	tests := []struct {
		name, path string
		header     http.Header
		want       int
	}{
		{"missing-bearer", forwardPath, http.Header{}, http.StatusUnauthorized},
		{"wrong-bearer", forwardPath, http.Header{"Authorization": {"Bearer wrong"}}, http.StatusUnauthorized},
		{"invalid-port", "/v1/manage/pods/apps/worker/port-forward?port=0", forwardHeaders(), http.StatusBadRequest},
		{"missing-confirmation", forwardPath, http.Header{"Authorization": {"Bearer test-token"}}, http.StatusConflict},
		{"wrong-uid", forwardPath, func() http.Header { h := forwardHeaders(); h.Set("If-Match-UID", "wrong"); return h }(), http.StatusConflict},
		{"wrong-version", forwardPath, func() http.Header { h := forwardHeaders(); h.Set("If-Match", "6"); return h }(), http.StatusConflict},
		{"browser-origin", forwardPath, func() http.Header { h := forwardHeaders(); h.Set("Origin", "https://evil.example"); return h }(), http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connection, response, err := websocket.DefaultDialer.Dial(forwardURL(server, tt.path), tt.header)
			if connection != nil {
				connection.Close()
			}
			if err == nil || response == nil || response.StatusCode != tt.want {
				t.Fatalf("status=%v error=%v, want %d", response, err, tt.want)
			}
		})
	}
	request, _ := http.NewRequest(http.MethodGet, server.URL+forwardPath, nil)
	for key, values := range forwardHeaders() {
		request.Header[key] = values
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil || response.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-upgrade request: %v %v", response, err)
	}
	response.Body.Close()
	if calls.Load() != 0 {
		t.Fatalf("invalid request opened %d tunnels", calls.Load())
	}
}

func TestPortForwardCapabilityIsAdvertisedOnlyWhenEnabled(t *testing.T) {
	m := testManager(t)
	if response := request(m, http.MethodGet, "/v1/manage/capabilities", nil, nil); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"podPortForward":false`) {
		t.Fatalf("disabled capability response: %d", response.Code)
	}
	m.PortForwardEnabled = true
	if response := request(m, http.MethodGet, "/v1/manage/capabilities", nil, nil); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"podPortForward":true`) {
		t.Fatalf("enabled capability response: %d", response.Code)
	}
}

func TestPortForwardRelaysBinaryDataAndReleasesConnection(t *testing.T) {
	m := readyExecManager(t)
	m.ExecEnabled = false
	m.PortForwardEnabled = true
	var calls atomic.Int32
	m.ForwardRun = func(_ context.Context, namespace, name string, port uint16) (net.Conn, error) {
		if namespace != "apps" || name != "worker" || port != 8080 {
			t.Errorf("wrong target: %s/%s:%d", namespace, name, port)
		}
		calls.Add(1)
		client, pod := net.Pipe()
		go func() {
			defer pod.Close()
			_, _ = io.Copy(pod, pod)
		}()
		return client, nil
	}
	server := startForwardTestServer(t, m)
	connection, response, err := websocket.DefaultDialer.Dial(forwardURL(server, forwardPath), forwardHeaders())
	if err != nil || response.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("handshake: %v %v", response, err)
	}
	defer connection.Close()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	if err := connection.WriteMessage(websocket.BinaryMessage, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	kind, payload, err := connection.ReadMessage()
	if err != nil || kind != websocket.BinaryMessage || string(payload) != "ping" || calls.Load() != 1 {
		t.Fatalf("relay kind=%d payload=%q calls=%d err=%v", kind, payload, calls.Load(), err)
	}
}

func TestPortForwardLifetimeAndByteLimit(t *testing.T) {
	for _, test := range []struct {
		name     string
		limit    int64
		lifetime time.Duration
		payload  string
	}{
		{"byte-limit", 4, time.Second, "hello"},
		{"frame-limit", defaultForwardByteLimit, time.Second, strings.Repeat("x", maxForwardFrame+1)},
		{"lifetime", 64, 20 * time.Millisecond, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := readyExecManager(t)
			m.ExecEnabled = false
			m.PortForwardEnabled = true
			m.forwardByteLimit = test.limit
			m.forwardLifetime = test.lifetime
			m.ForwardRun = func(context.Context, string, string, uint16) (net.Conn, error) {
				client, pod := net.Pipe()
				t.Cleanup(func() { pod.Close() })
				return client, nil
			}
			server := startForwardTestServer(t, m)
			connection, _, err := websocket.DefaultDialer.Dial(forwardURL(server, forwardPath), forwardHeaders())
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			_ = connection.SetReadDeadline(time.Now().Add(time.Second))
			if test.payload != "" {
				if err := connection.WriteMessage(websocket.BinaryMessage, []byte(test.payload)); err != nil {
					t.Fatal(err)
				}
			}
			if _, _, err := connection.ReadMessage(); err == nil {
				t.Fatal("bounded forward remained open")
			}
		})
	}
}

func TestPortForwardCapacityReleasesOnDisconnect(t *testing.T) {
	m := readyExecManager(t)
	m.ExecEnabled = false
	m.PortForwardEnabled = true
	m.forwardOnce.Do(func() { m.forwardSlots = make(chan struct{}, 1) })
	var calls atomic.Int32
	m.ForwardRun = func(ctx context.Context, _, _ string, _ uint16) (net.Conn, error) {
		calls.Add(1)
		client, pod := net.Pipe()
		go func() {
			<-ctx.Done()
			_ = pod.Close()
		}()
		return client, nil
	}
	server := startForwardTestServer(t, m)
	first, _, err := websocket.DefaultDialer.Dial(forwardURL(server, forwardPath), forwardHeaders())
	if err != nil {
		t.Fatal(err)
	}
	_, response, err := websocket.DefaultDialer.Dial(forwardURL(server, forwardPath), forwardHeaders())
	if err == nil || response == nil || response.StatusCode != http.StatusTooManyRequests || calls.Load() != 1 {
		t.Fatalf("capacity status=%v calls=%d err=%v", response, calls.Load(), err)
	}
	first.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		third, response, err := websocket.DefaultDialer.Dial(forwardURL(server, forwardPath), forwardHeaders())
		if err == nil {
			third.Close()
			if calls.Load() != 2 {
				t.Fatalf("slot reused with %d runner calls", calls.Load())
			}
			return
		}
		if response == nil || response.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("unexpected retry status=%v err=%v", response, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("port-forward slot did not release")
}
