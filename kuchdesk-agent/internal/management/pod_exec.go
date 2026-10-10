package management

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
)

const maxExecOutputBytes = 256 << 10

var errExecOutputLimit = errors.New("exec output limit reached")

// PodExecRunner is injectable so the HTTP boundary can be tested without
// starting a command inside a real Pod.
type PodExecRunner func(context.Context, string, string, *corev1.PodExecOptions, io.Writer, io.Writer) error

type podExecRequest struct {
	Container string   `json:"container"`
	Command   []string `json:"command"`
}

func (m *Manager) execPod(w http.ResponseWriter, r *http.Request) {
	namespace, name := r.PathValue("namespace"), r.PathValue("name")
	if !segment.MatchString(namespace) || !segment.MatchString(name) {
		http.Error(w, "invalid Pod identity", http.StatusBadRequest)
		return
	}
	defer m.audit(r, "pods", namespace, name, "exec")
	if r.Header.Get("X-KuchDesk-Confirm") != "pods/"+namespace+"/"+name+"/exec" || r.Header.Get("If-Match-UID") == "" || r.Header.Get("If-Match") == "" {
		http.Error(w, "exact confirmation, Pod UID and resourceVersion required", http.StatusConflict)
		return
	}
	var input podExecRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		http.Error(w, "invalid exec request", http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) || !validExecRequest(input) {
		http.Error(w, "invalid exec request", http.StatusBadRequest)
		return
	}
	pod, err := m.Kube.CoreV1().Pods(namespace).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, err)
		return
	}
	if string(pod.UID) != r.Header.Get("If-Match-UID") || pod.ResourceVersion != r.Header.Get("If-Match") || pod.Status.Phase != corev1.PodRunning || !runningContainer(pod, input.Container) {
		http.Error(w, "Pod identity or running container changed", http.StatusConflict)
		return
	}
	m.execOnce.Do(func() { m.execSlots = make(chan struct{}, 4) })
	select {
	case m.execSlots <- struct{}{}:
		defer func() { <-m.execSlots }()
	default:
		http.Error(w, "exec capacity reached", http.StatusTooManyRequests)
		return
	}
	timeout := m.execTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	stdout := &boundedExecOutput{cancel: cancel}
	stderr := &boundedExecOutput{cancel: cancel}
	options := &corev1.PodExecOptions{Container: input.Container, Command: input.Command, Stdout: true, Stderr: true}
	run := m.ExecRun
	if run == nil {
		run = m.runPodExec
	}
	err = run(ctx, namespace, name, options, stdout, stderr)
	if stdout.exceeded || stderr.exceeded {
		http.Error(w, "exec output limit reached", http.StatusRequestEntityTooLarge)
		return
	}
	if r.Context().Err() != nil {
		return
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		http.Error(w, "exec timed out", http.StatusGatewayTimeout)
		return
	}
	exitCode := 0
	if err != nil {
		var exitErr utilexec.ExitError
		if !errors.As(err, &exitErr) {
			http.Error(w, "Pod exec failed", http.StatusBadGateway)
			return
		}
		exitCode = exitErr.ExitStatus()
	}
	writeJSON(w, http.StatusOK, map[string]any{"stdout": stdout.String(), "stderr": stderr.String(), "exitCode": exitCode})
}

func validExecRequest(input podExecRequest) bool {
	if len(validation.IsDNS1123Label(input.Container)) != 0 || len(input.Command) == 0 || len(input.Command) > 32 || input.Command[0] == "" {
		return false
	}
	length := 0
	for _, part := range input.Command {
		if len(part) > 256 || strings.ContainsRune(part, '\x00') {
			return false
		}
		length += len(part)
	}
	return length <= 4096
}

func runningContainer(pod *corev1.Pod, name string) bool {
	for _, container := range pod.Spec.Containers {
		if container.Name != name {
			continue
		}
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == name && status.State.Running != nil {
				return true
			}
		}
	}
	return false
}

func (m *Manager) runPodExec(ctx context.Context, namespace, name string, options *corev1.PodExecOptions, stdout, stderr io.Writer) error {
	if m.RESTConfig == nil {
		return errors.New("Kubernetes REST configuration unavailable")
	}
	request := m.Kube.CoreV1().RESTClient().Post().Resource("pods").Namespace(namespace).Name(name).SubResource("exec").VersionedParams(options, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(m.RESTConfig, http.MethodPost, request.URL())
	if err != nil {
		return err
	}
	return executor.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: stdout, Stderr: stderr})
}

type boundedExecOutput struct {
	bytes.Buffer
	cancel   context.CancelFunc
	exceeded bool
}

func (output *boundedExecOutput) Write(data []byte) (int, error) {
	remaining := maxExecOutputBytes - output.Len()
	if len(data) > remaining {
		output.exceeded = true
		output.cancel()
		return 0, errExecOutputLimit
	}
	return output.Buffer.Write(data)
}
