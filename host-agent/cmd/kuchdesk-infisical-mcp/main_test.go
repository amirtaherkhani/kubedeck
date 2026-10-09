package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/async"
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
	if len(tools.Tools) != 5 {
		t.Fatalf("expected five MCP tools, got %+v", tools.Tools)
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

type fakeNameLister struct{}

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
