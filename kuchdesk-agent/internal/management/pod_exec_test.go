package management

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/httpapi"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
	utilexec "k8s.io/client-go/util/exec"
)

const execPath = "/v1/manage/pods/apps/worker/exec"

func execHeaders() map[string]string {
	return map[string]string{
		"X-KuchDesk-Confirm": "pods/apps/worker/exec",
		"If-Match-UID":       "pod-uid",
		"If-Match":           "7",
	}
}

func readyExecManager(t *testing.T) *Manager {
	t.Helper()
	m := testManager(t)
	m.ExecEnabled = true
	m.Kube = kubefake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "apps", UID: "pod-uid", ResourceVersion: "7"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "main", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}},
	})
	return m
}

func TestPodExecIsOffByDefaultAndRequiresExactIdentity(t *testing.T) {
	m := testManager(t)
	if response := request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["true"]}`), execHeaders()); response.Code != http.StatusNotFound {
		t.Fatalf("disabled exec returned %d", response.Code)
	}
	m = readyExecManager(t)
	var called atomic.Int32
	m.ExecRun = func(context.Context, string, string, *corev1.PodExecOptions, io.Writer, io.Writer) error {
		called.Add(1)
		return nil
	}
	for _, headers := range []map[string]string{nil, {"X-KuchDesk-Confirm": "pods/apps/worker/exec", "If-Match-UID": "wrong", "If-Match": "7"}, {"X-KuchDesk-Confirm": "pods/apps/worker/exec", "If-Match-UID": "pod-uid", "If-Match": "6"}} {
		if response := request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["true"]}`), headers); response.Code != http.StatusConflict {
			t.Fatalf("invalid identity returned %d", response.Code)
		}
	}
	for _, body := range []string{`{"container":"main","command":[]}`, `{"container":"main","command":["true"],"stdin":true}`, `{"container":"main","command":["true"]} {}`} {
		if response := request(m, http.MethodPost, execPath, []byte(body), execHeaders()); response.Code != http.StatusBadRequest {
			t.Fatalf("invalid request returned %d", response.Code)
		}
	}
	if response := request(m, http.MethodPost, execPath, []byte(`{"container":"other","command":["true"]}`), execHeaders()); response.Code != http.StatusConflict {
		t.Fatalf("non-running container returned %d", response.Code)
	}
	if called.Load() != 0 {
		t.Fatal("invalid request reached the exec transport")
	}
}

func TestPodExecCapabilityIsAdvertisedOnlyWhenEnabled(t *testing.T) {
	m := testManager(t)
	if response := request(m, http.MethodGet, "/v1/manage/capabilities", nil, nil); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"podExec":false`)) {
		t.Fatalf("disabled capability result: %d", response.Code)
	}
	m.ExecEnabled = true
	if response := request(m, http.MethodGet, "/v1/manage/capabilities", nil, nil); response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"podExec":true`)) {
		t.Fatalf("enabled capability result: %d", response.Code)
	}
}

func TestPodExecReturnsBoundedOutputAndExitStatus(t *testing.T) {
	m := readyExecManager(t)
	m.ExecRun = func(_ context.Context, namespace, name string, options *corev1.PodExecOptions, stdout, stderr io.Writer) error {
		if namespace != "apps" || name != "worker" || options.Container != "main" || len(options.Command) != 2 || options.Command[0] != "probe" || options.Stdin || options.TTY {
			t.Fatalf("unsafe exec options: %#v", options)
		}
		_, _ = io.WriteString(stdout, "out")
		_, _ = io.WriteString(stderr, "err")
		return utilexec.CodeExitError{Err: errors.New("exit status 7"), Code: 7}
	}
	response := request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["probe","status"]}`), execHeaders())
	if response.Code != http.StatusOK {
		t.Fatalf("exec returned %d", response.Code)
	}
	var result struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exitCode"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Stdout != "out" || result.Stderr != "err" || result.ExitCode != 7 {
		t.Fatalf("unexpected exec response: %#v, %v", result, err)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("exec output was cacheable")
	}
	m.ExecRun = func(_ context.Context, _, _ string, _ *corev1.PodExecOptions, stdout, _ io.Writer) error {
		_, err := stdout.Write(bytes.Repeat([]byte("x"), maxExecOutputBytes+1))
		return err
	}
	response = request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["probe"]}`), execHeaders())
	if response.Code != http.StatusRequestEntityTooLarge || bytes.Contains(response.Body.Bytes(), []byte("xxxxx")) {
		t.Fatalf("output limit response: %d", response.Code)
	}
	m.ExecRun = func(context.Context, string, string, *corev1.PodExecOptions, io.Writer, io.Writer) error {
		return errors.New("transport detail")
	}
	response = request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["probe"]}`), execHeaders())
	if response.Code != http.StatusBadGateway || bytes.Contains(response.Body.Bytes(), []byte("transport detail")) {
		t.Fatalf("transport error leaked: %d", response.Code)
	}
}

func TestPodExecCancellationReleasesConcurrencySlot(t *testing.T) {
	m := readyExecManager(t)
	m.execOnce.Do(func() { m.execSlots = make(chan struct{}, 1) })
	entered := make(chan struct{})
	var calls atomic.Int32
	m.ExecRun = func(ctx context.Context, _, _ string, _ *corev1.PodExecOptions, _, _ io.Writer) error {
		if calls.Add(1) == 1 {
			close(entered)
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	first := httptest.NewRequest(http.MethodPost, execPath, bytes.NewBufferString(`{"container":"main","command":["probe"]}`)).WithContext(ctx)
	for key, value := range execHeaders() {
		first.Header.Set(key, value)
	}
	done := make(chan struct{})
	go func() {
		m.Handler().ServeHTTP(httptest.NewRecorder(), first)
		close(done)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first exec never started")
	}
	second := request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["probe"]}`), execHeaders())
	if second.Code != http.StatusTooManyRequests || calls.Load() != 1 {
		t.Fatalf("concurrent exec returned %d with %d calls", second.Code, calls.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("canceled exec did not finish")
	}
	third := request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["probe"]}`), execHeaders())
	if third.Code != http.StatusOK || calls.Load() != 2 {
		t.Fatalf("slot was not released: %d with %d calls", third.Code, calls.Load())
	}
}

func TestPodExecRequiresBearerAtHTTPBoundary(t *testing.T) {
	m := readyExecManager(t)
	var calls atomic.Int32
	m.ExecRun = func(context.Context, string, string, *corev1.PodExecOptions, io.Writer, io.Writer) error {
		calls.Add(1)
		return nil
	}
	api := httpapi.New(nil, nil, nil, "test-token", time.Second, nil)
	api.SetManagementHandler(m.Handler())
	for _, tc := range []struct {
		token string
		want  int
	}{
		{"", http.StatusUnauthorized},
		{"Bearer wrong", http.StatusUnauthorized},
		{"Bearer test-token", http.StatusOK},
	} {
		r := httptest.NewRequest(http.MethodPost, execPath, bytes.NewBufferString(`{"container":"main","command":["true"]}`))
		for key, value := range execHeaders() {
			r.Header.Set(key, value)
		}
		if tc.token != "" {
			r.Header.Set("Authorization", tc.token)
		}
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("auth %q returned %d, want %d", tc.token, w.Code, tc.want)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("unauthorized exec reached transport: %d calls", calls.Load())
	}
}

func TestPodExecDeadlineStopsRunner(t *testing.T) {
	m := readyExecManager(t)
	m.execTimeout = 10 * time.Millisecond
	m.ExecRun = func(ctx context.Context, _, _ string, _ *corev1.PodExecOptions, _, _ io.Writer) error {
		<-ctx.Done()
		return ctx.Err()
	}
	response := request(m, http.MethodPost, execPath, []byte(`{"container":"main","command":["wait"]}`), execHeaders())
	if response.Code != http.StatusGatewayTimeout || !bytes.Contains(response.Body.Bytes(), []byte("exec timed out")) {
		t.Fatalf("timed-out exec response: %d %q", response.Code, response.Body.String())
	}
}
