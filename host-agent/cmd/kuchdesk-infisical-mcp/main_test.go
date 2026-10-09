package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
	if len(tools.Tools) != 2 || tools.Tools[0].InputSchema == nil || tools.Tools[1].InputSchema == nil {
		t.Fatalf("expected two typed MCP tools, got %+v", tools.Tools)
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
