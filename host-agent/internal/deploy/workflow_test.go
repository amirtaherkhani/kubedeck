package deploy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type call struct {
	name string
	args []string
}

type fakeRunner struct {
	calls   []call
	failAt  string
	digests string
	nodes   string
}

func (f *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	f.calls = append(f.calls, call{name: name, args: args})
	if name+" "+args[0] == f.failAt {
		return errors.New("command failed")
	}
	return nil
}

func (f *fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, call{name: name, args: args})
	switch name + " " + args[0] {
	case "git rev-parse":
		return []byte("123456789abc\n"), nil
	case "git status":
		return []byte{}, nil
	case "docker image":
		return []byte(f.digests), nil
	case "crane digest":
		return []byte(f.digests), nil
	case "kubectl --context":
		return []byte(f.nodes), nil
	default:
		return nil, errors.New("unexpected output call")
	}
}

func TestHostRegistryPushAndKindPrepull(t *testing.T) {
	spec := fixture(t)
	spec.PushMode = "host-crane"
	spec.HostPushRepo = "127.0.0.1:5001/kuchdesk-agent"
	spec.KindPrepull = true
	digest := "sha256:" + strings.Repeat("b", 64)
	runner := &fakeRunner{digests: digest + "\n", nodes: `{"items":[{"metadata":{"name":"desktop-control-plane"}}]}`}
	result, err := (Workflow{Runner: runner}).Apply(context.Background(), spec)
	if err != nil || result.Digest != digest {
		t.Fatalf("host push failed: %+v %v", result, err)
	}
	var save, push, prepull, upgrade bool
	for _, call := range runner.calls {
		joined := call.name + " " + strings.Join(call.args, " ")
		if strings.HasPrefix(joined, "docker save ") {
			save = true
		}
		if strings.HasPrefix(joined, "crane push ") {
			push = true
		}
		if strings.Contains(joined, "docker push ") {
			t.Fatal("used Docker daemon to push through host-only port-forward")
		}
		if strings.HasPrefix(joined, "docker exec desktop-control-plane crictl pull ") && strings.Contains(joined, spec.ImageRepository+":"+spec.ImageTag+"@"+digest) {
			prepull = true
		}
		if strings.Contains(joined, " ctr ") {
			t.Fatal("containerd image alias does not ensure kubelet CRI visibility")
		}
		if strings.HasPrefix(joined, "helm upgrade ") {
			upgrade = true
			if !strings.Contains(joined, "image.pullPolicy=Never") {
				t.Fatal("pre-pulled image may be fetched again by kubelet")
			}
		}
	}
	if !save || !push || !prepull || !upgrade {
		t.Fatalf("missing workflow phase: save=%t push=%t prepull=%t upgrade=%t", save, push, prepull, upgrade)
	}
}

func TestPrepullFailureStopsBeforeHelm(t *testing.T) {
	spec := fixture(t)
	spec.PushMode = "host-crane"
	spec.HostPushRepo = "127.0.0.1:5001/kuchdesk-agent"
	spec.KindPrepull = true
	runner := &fakeRunner{digests: "sha256:" + strings.Repeat("b", 64), nodes: `{"items":[{"metadata":{"name":"desktop-control-plane"}}]}`, failAt: "docker exec"}
	if _, err := (Workflow{Runner: runner}).Apply(context.Background(), spec); err == nil {
		t.Fatal("accepted failed node pre-pull")
	}
	for _, call := range runner.calls {
		if call.name == "helm" && call.args[0] == "upgrade" {
			t.Fatal("deployed without node image")
		}
	}
}

func fixture(t *testing.T) Spec {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "source")
	chart := filepath.Join(source, "chart")
	if err := os.MkdirAll(chart, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{filepath.Join(source, "Dockerfile"), filepath.Join(chart, "Chart.yaml"), filepath.Join(chart, "values.yaml")} {
		if err := os.WriteFile(file, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return Spec{SourceDir: source, Dockerfile: filepath.Join(source, "Dockerfile"), ChartDir: chart, ValuesFiles: []string{filepath.Join(chart, "values.yaml")}, ImageRepository: "localhost:5001/kuchdesk-agent", ImageTag: "git-123456789abc", KubeContext: "docker-desktop", Namespace: "development-tools", Release: "kuchdesk-agent", Deployment: "kuchdesk-agent", Test: "go"}
}

func TestApplyPinsDigestAndValidatesRollout(t *testing.T) {
	spec := fixture(t)
	digest := "sha256:" + strings.Repeat("a", 64)
	runner := &fakeRunner{digests: `["localhost:5001/kuchdesk-agent@` + digest + `"]`}
	result, err := (Workflow{Runner: runner}).Apply(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if result.Digest != digest || result.Image != "localhost:5001/kuchdesk-agent:git-123456789abc" {
		t.Fatalf("wrong resolved image: %+v", result)
	}
	var steps []string
	for _, call := range runner.calls {
		steps = append(steps, call.name+" "+call.args[0])
		if call.name == "helm" && (call.args[0] == "template" || call.args[0] == "upgrade") {
			joined := strings.Join(call.args, " ")
			if !strings.Contains(joined, "image.digest="+digest) || !strings.Contains(joined, "--kube-context docker-desktop") {
				t.Fatalf("Helm call lost digest or explicit context: %s", joined)
			}
		}
	}
	want := []string{"git rev-parse", "git status", "helm lint", "go test", "docker build", "docker push", "docker image", "helm template", "helm upgrade", "kubectl --context"}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("unexpected step order: got %v want %v", steps, want)
	}
}

func TestPushFailureNeverCallsHelmUpgrade(t *testing.T) {
	spec := fixture(t)
	runner := &fakeRunner{failAt: "docker push"}
	if _, err := (Workflow{Runner: runner}).Apply(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "docker push failed") {
		t.Fatalf("expected push failure, got %v", err)
	}
	for _, call := range runner.calls {
		if call.name == "helm" && call.args[0] == "upgrade" {
			t.Fatal("Helm upgraded after push failure")
		}
	}
}

func TestMissingDigestNeverCallsHelmUpgrade(t *testing.T) {
	spec := fixture(t)
	runner := &fakeRunner{digests: `[]`}
	if _, err := (Workflow{Runner: runner}).Apply(context.Background(), spec); err == nil || !strings.Contains(err.Error(), "digest unavailable") {
		t.Fatalf("expected digest failure, got %v", err)
	}
	for _, call := range runner.calls {
		if call.name == "helm" && (call.args[0] == "template" || call.args[0] == "upgrade") {
			t.Fatal("Helm ran without a resolved image digest")
		}
	}
}

func TestSpecRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	spec := fixture(t)
	spec.ImageTag = "latest"
	if _, err := spec.Plan(); err == nil {
		t.Fatal("accepted mutable image tag")
	}
	spec = fixture(t)
	spec.Dockerfile = filepath.Join(t.TempDir(), "Dockerfile")
	if err := os.WriteFile(spec.Dockerfile, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := spec.Plan(); err == nil {
		t.Fatal("accepted Dockerfile outside source revision")
	}
	spec = fixture(t)
	spec.KubeContext = ""
	if _, err := spec.Plan(); err == nil {
		t.Fatal("accepted implicit Kubernetes context")
	}
}
