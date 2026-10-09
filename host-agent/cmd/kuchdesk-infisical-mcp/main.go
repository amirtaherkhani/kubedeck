package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/async"
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

type jobInput struct {
	ID string `json:"id" jsonschema:"required,operation ID"`
}

type jobStarted struct {
	ID string `json:"id"`
}

type jobOutput struct {
	ID      string                 `json:"id"`
	State   async.State            `json:"state"`
	Secrets []infisical.SecretName `json:"secrets,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

func newServer(client *infisical.Client) *mcp.Server {
	return newServerWithRunner(infisical.NewService(client), async.NewRunner(context.Background(), 4, 64))
}

func newServerWithRunner(service *infisical.Service, runner *async.Runner) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "kuchdesk-infisical", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_capabilities", Description: "Show implemented Infisical tools and local credential configuration without accessing secrets.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(context.Context, *mcp.CallToolRequest, capabilityInput) (*mcp.CallToolResult, capabilityOutput, error) {
			return nil, capabilityOutput{Configured: service.Configured(), Capabilities: []capability{
				{Name: "list_secret_names", Status: "implemented; live permissions unverified"},
				{Name: "async_name_listing", Status: "implemented in memory; bounded to four concurrent operations and 64 records"},
				{Name: "secret_values", Status: "not exposed to MCP"},
				{Name: "secret_writes", Status: "planned; not implemented"},
				{Name: "project_admin", Status: "planned; not implemented"},
				{Name: "helm_deploy", Status: "planned; not implemented"},
			}}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_list_secret_names", Description: "List only secret names and paths in one explicit project, environment, and path. Values are never returned.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, listOutput, error) {
			secrets, err := service.ListSecretNames(ctx, infisical.Scope{ProjectID: input.ProjectID, Environment: input.Environment, SecretPath: input.SecretPath})
			if err != nil {
				return nil, listOutput{}, publicError(err)
			}
			return nil, listOutput{Secrets: secrets}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_start_list_secret_names", Description: "Start a bounded in-process name-only secret listing operation.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(_ context.Context, _ *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, jobStarted, error) {
			scope := infisical.Scope{ProjectID: input.ProjectID, Environment: input.Environment, SecretPath: input.SecretPath}
			if err := infisical.ValidateScope(scope); err != nil {
				return nil, jobStarted{}, publicError(err)
			}
			id, err := runner.Submit("", 30*time.Second, func(ctx context.Context) (any, error) {
				secrets, err := service.ListSecretNames(ctx, scope)
				if err != nil {
					return nil, publicError(err)
				}
				return secrets, nil
			})
			if err != nil {
				return nil, jobStarted{}, publicError(err)
			}
			return nil, jobStarted{ID: id}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_job_status", Description: "Read the status of an in-process operation. Completed results are retained only in bounded memory.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(_ context.Context, _ *mcp.CallToolRequest, input jobInput) (*mcp.CallToolResult, jobOutput, error) {
			status, err := runner.Status(input.ID)
			if err != nil {
				return nil, jobOutput{}, publicError(err)
			}
			return nil, outputFor(status), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_cancel_job", Description: "Request cancellation of a queued or running in-process operation."},
		func(_ context.Context, _ *mcp.CallToolRequest, input jobInput) (*mcp.CallToolResult, jobOutput, error) {
			status, err := runner.Cancel(input.ID)
			if err != nil {
				return nil, jobOutput{}, publicError(err)
			}
			return nil, outputFor(status), nil
		})
	return server
}

func outputFor(status async.Snapshot) jobOutput {
	output := jobOutput{ID: status.ID, State: status.State, Error: status.Error}
	if names, ok := status.Result.([]infisical.SecretName); ok {
		output.Secrets = names
	}
	return output
}

func publicError(err error) error {
	switch {
	case errors.Is(err, async.ErrCapacity):
		return errors.New("operation_capacity_reached")
	case errors.Is(err, async.ErrNotFound):
		return errors.New("operation_not_found")
	default:
		return infisical.PublicError(err)
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
	runner := async.NewRunner(context.Background(), 4, 64)
	defer runner.Close()
	if err := newServerWithRunner(infisical.NewService(client), runner).Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "Infisical MCP server stopped")
		os.Exit(1)
	}
}
