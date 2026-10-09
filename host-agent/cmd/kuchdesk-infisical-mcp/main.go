package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type capabilityInput struct{}

type capability struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type capabilityOutput struct {
	Configured   bool         `json:"configured"`
	Capabilities []capability `json:"capabilities"`
}

type listInput struct {
	ProjectID   string `json:"projectId" jsonschema:"required,Infisical project ID"`
	Environment string `json:"environment" jsonschema:"required,Infisical environment slug"`
	SecretPath  string `json:"secretPath" jsonschema:"required,absolute secret path"`
}

type listOutput struct {
	Secrets []infisical.SecretName `json:"secrets"`
}

func newServer(client *infisical.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "kuchdesk-infisical", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_capabilities", Description: "Show implemented Infisical tools and local credential configuration without accessing secrets.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(context.Context, *mcp.CallToolRequest, capabilityInput) (*mcp.CallToolResult, capabilityOutput, error) {
			return nil, capabilityOutput{Configured: client.Configured(), Capabilities: []capability{
				{Name: "list_secret_names", Status: "implemented; live permissions unverified"},
				{Name: "secret_values", Status: "not exposed to MCP"},
				{Name: "secret_writes", Status: "planned; not implemented"},
				{Name: "project_admin", Status: "planned; not implemented"},
				{Name: "helm_deploy", Status: "planned; not implemented"},
			}}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_list_secret_names", Description: "List only secret names and paths in one explicit project, environment, and path. Values are never returned.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, listOutput, error) {
			secrets, err := client.ListSecretNames(ctx, input.ProjectID, input.Environment, input.SecretPath)
			if err != nil {
				return nil, listOutput{}, publicError(err)
			}
			return nil, listOutput{Secrets: secrets}, nil
		})
	return server
}

func publicError(err error) error {
	var apiErr *infisical.APIError
	switch {
	case errors.Is(err, infisical.ErrNotConfigured):
		return errors.New("credentials_not_configured")
	case errors.Is(err, infisical.ErrInvalidScope):
		return errors.New("invalid_scope")
	case errors.Is(err, infisical.ErrAuthRejected):
		return errors.New("authentication_rejected")
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden:
		return errors.New("permission_denied")
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
		return errors.New("not_found")
	default:
		return errors.New("infisical_unavailable")
	}
}

func main() {
	baseURL := flag.String("url", "", "Infisical HTTPS origin (required)")
	flag.Parse()
	if *baseURL == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "provide -url with the Infisical HTTPS origin")
		os.Exit(2)
	}
	client, err := infisical.NewClient(*baseURL, os.Getenv("INFISICAL_CLIENT_ID"), os.Getenv("INFISICAL_CLIENT_SECRET"), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid Infisical configuration:", err)
		os.Exit(2)
	}
	if err := newServer(client).Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "Infisical MCP server stopped")
		os.Exit(1)
	}
}
