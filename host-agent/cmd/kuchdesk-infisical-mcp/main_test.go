package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/async"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/deploy"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPDiscoveryAndRedactedErrors(t *testing.T) {
	client, err := infisical.NewClient("https://infisical.example", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := newServer(client).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-harness"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 9 {
		t.Fatalf("expected nine MCP tools, got %+v", tools.Tools)
	}
	for _, tool := range tools.Tools {
		if tool.InputSchema == nil {
			t.Fatalf("tool %s has no input schema", tool.Name)
		}
	}
	capabilities, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "infisical_capabilities", Arguments: map[string]any{}})
	if err != nil || capabilities.IsError {
		t.Fatalf("capability discovery failed: %v %+v", err, capabilities)
	}
	encoded, _ := json.Marshal(capabilities.StructuredContent)
	if !strings.Contains(string(encoded), `"configured":false`) {
		t.Fatalf("unexpected capability output: %s", encoded)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "infisical_list_secret_names", Arguments: map[string]any{
		"projectId": "project", "environment": "dev", "secretPath": "/",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("missing credentials should return a tool error")
	}
	encoded, _ = json.Marshal(result)
	if !strings.Contains(string(encoded), "credentials_not_configured") || strings.Contains(string(encoded), "clientSecret") {
		t.Fatalf("unexpected error output: %s", encoded)
	}
}

func TestMCPDeployProfileBoundary(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	chart := filepath.Join(source, "chart")
	profiles := filepath.Join(root, "profiles")
	for _, dir := range []string{chart, profiles} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{filepath.Join(source, "Dockerfile"), filepath.Join(chart, "Chart.yaml")} {
		if err := os.WriteFile(file, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	spec := deploy.Spec{SourceDir: source, Dockerfile: filepath.Join(source, "Dockerfile"), ChartDir: chart, ImageRepository: "localhost:5001/kuchdesk-agent", ImageTag: "git-123456789abc", KubeContext: "docker-desktop", Namespace: "development-tools", Release: "kuchdesk-agent", Deployment: "kuchdesk-agent"}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profiles, "agent.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside.json"), filepath.Join(profiles, "escape.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "outside.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	runner := async.NewRunner(ctx, 2, 4)
	defer runner.Close()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := newServerWithDeploy(infisical.NewService(fakeNameLister{}), runner, deployConfig{ProfileDir: profiles}).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-harness"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	plan, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "kuchdesk_deploy_plan", Arguments: map[string]any{"profile": "agent.json"}})
	if err != nil || plan.IsError {
		t.Fatalf("plan failed: %v %+v", err, plan)
	}
	encoded, _ := json.Marshal(plan.StructuredContent)
	if !strings.Contains(string(encoded), "kuchdesk-agent") {
		t.Fatalf("unexpected plan: %s", encoded)
	}
	for _, profile := range []string{"../outside.json", "escape.json"} {
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "kuchdesk_deploy_plan", Arguments: map[string]any{"profile": profile}})
		if err != nil || !result.IsError {
			t.Fatalf("unsafe profile accepted: %s %v %+v", profile, err, result)
		}
	}
	start, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "kuchdesk_deploy_start", Arguments: map[string]any{"profile": "agent.json", "confirm": "kuchdesk-agent/development-tools"}})
	if err != nil || !start.IsError {
		t.Fatalf("disabled deploy accepted: %v %+v", err, start)
	}
	pushClient, pushServer := mcp.NewInMemoryTransports()
	pushSession, err := newServerWithDeploy(infisical.NewService(fakeNameLister{}), runner, deployConfig{ProfileDir: profiles, Enabled: true, Runner: deployCommandRunner{}}).Connect(ctx, pushServer, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer pushSession.Close()
	client, err := mcp.NewClient(&mcp.Implementation{Name: "deploy-test"}, nil).Connect(ctx, pushClient, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	bad, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "kuchdesk_deploy_start", Arguments: map[string]any{"profile": "agent.json", "confirm": "wrong/namespace"}})
	if err != nil || !bad.IsError {
		t.Fatalf("missing exact confirmation accepted: %v %+v", err, bad)
	}
	started, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "kuchdesk_deploy_start", Arguments: map[string]any{"profile": "agent.json", "confirm": "kuchdesk-agent/development-tools"}})
	if err != nil || started.IsError {
		t.Fatalf("confirmed deployment failed: %v %+v", err, started)
	}
	encoded, _ = json.Marshal(started.StructuredContent)
	var job jobStarted
	if err := json.Unmarshal(encoded, &job); err != nil || job.ID == "" {
		t.Fatalf("missing deployment job ID: %s %v", encoded, err)
	}
	deadline := time.After(time.Second)
	for {
		status, err := client.CallTool(ctx, &mcp.CallToolParams{Name: "kuchdesk_job_status", Arguments: map[string]any{"id": job.ID}})
		if err != nil || status.IsError {
			t.Fatalf("deployment status failed: %v %+v", err, status)
		}
		encoded, _ = json.Marshal(status.StructuredContent)
		var output jobOutput
		if err := json.Unmarshal(encoded, &output); err != nil {
			t.Fatal(err)
		}
		if output.State == async.Succeeded {
			if output.Deployment == nil || output.Deployment.Digest == "" {
				t.Fatalf("deployment result missing digest: %+v", output)
			}
			break
		}
		if output.State == async.Failed {
			t.Fatalf("deployment failed: %+v", output)
		}
		select {
		case <-deadline:
			t.Fatalf("deployment timed out: %+v", output)
		case <-time.After(time.Millisecond):
		}
	}
}

type fakeNameLister struct{}

type deployCommandRunner struct{}

func (deployCommandRunner) Run(context.Context, string, string, ...string) error { return nil }
func (deployCommandRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	if name == "git" && args[0] == "rev-parse" {
		return []byte("123456789abc\n"), nil
	}
	if name == "git" && args[0] == "status" {
		return nil, nil
	}
	if name == "docker" && args[0] == "image" {
		return []byte(`["localhost:5001/kuchdesk-agent@sha256:` + strings.Repeat("a", 64) + `"]`), nil
	}
	return nil, os.ErrInvalid
}

func (fakeNameLister) Configured() bool { return true }
func (fakeNameLister) ListSecretNames(_ context.Context, _, _, _ string) ([]infisical.SecretName, error) {
	return []infisical.SecretName{{Name: "DB_USER", Path: "/app"}}, nil
}

func TestMCPAsyncNameListCompletes(t *testing.T) {
	ctx := context.Background()
	runner := async.NewRunner(ctx, 2, 4)
	defer runner.Close()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := newServerWithRunner(infisical.NewService(fakeNameLister{}), runner).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-harness"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	started, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "infisical_start_list_secret_names", Arguments: map[string]any{
		"projectId": "project", "environment": "dev", "secretPath": "/app",
	}})
	if err != nil || started.IsError {
		t.Fatalf("start failed: %v %+v", err, started)
	}
	var start jobStarted
	encoded, _ := json.Marshal(started.StructuredContent)
	if err := json.Unmarshal(encoded, &start); err != nil || start.ID == "" {
		t.Fatalf("invalid job ID: %v %s", err, encoded)
	}
	deadline := time.After(time.Second)
	for {
		status, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "infisical_job_status", Arguments: map[string]any{"id": start.ID}})
		if err != nil || status.IsError {
			t.Fatalf("status failed: %v %+v", err, status)
		}
		encoded, _ := json.Marshal(status.StructuredContent)
		var output jobOutput
		if err := json.Unmarshal(encoded, &output); err != nil {
			t.Fatal(err)
		}
		if output.State == async.Succeeded {
			if len(output.Secrets) != 1 || output.Secrets[0].Name != "DB_USER" {
				t.Fatalf("unexpected name-only result: %+v", output)
			}
			return
		}
		select {
		case <-deadline:
			t.Fatalf("job did not finish: %+v", output)
		case <-time.After(time.Millisecond):
		}
	}
}
