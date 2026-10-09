package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/async"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/deploy"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/doctor"
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
	ID         string                 `json:"id"`
	State      async.State            `json:"state"`
	Secrets    []infisical.SecretName `json:"secrets,omitempty"`
	Deployment *deploy.Result         `json:"deployment,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

func newServer(client *infisical.Client) *mcp.Server {
	return newServerWithRunner(infisical.NewService(client), async.NewRunner(context.Background(), 4, 64))
}

func newServerWithRunner(service *infisical.Service, runner *async.Runner) *mcp.Server {
	return newServerWithDeploy(service, runner, deployConfig{ProfileDir: os.Getenv("KUCHDESK_DEPLOY_PROFILE_DIR"), Enabled: os.Getenv("KUCHDESK_DEPLOY_ENABLED") == "true", Runner: deploy.ExecRunner{}})
}

type deployConfig struct {
	ProfileDir string
	Enabled    bool
	Runner     deploy.Runner
}

type deployInput struct {
	Profile string `json:"profile" jsonschema:"required,profile filename within configured directory"`
}

type deployStartInput struct {
	Profile string `json:"profile" jsonschema:"required,profile filename within configured directory"`
	Confirm string `json:"confirm" jsonschema:"required,exact release/namespace confirmation"`
}

type deployPlanOutput struct {
	Release     string   `json:"release"`
	Namespace   string   `json:"namespace"`
	KubeContext string   `json:"kubeContext"`
	Image       string   `json:"image"`
	Steps       []string `json:"steps"`
}

type doctorInput struct {
	Domain       string          `json:"domain" jsonschema:"required,DNS name to resolve on this Mac"`
	KubeContext  string          `json:"kubeContext" jsonschema:"required,explicit Kubernetes context"`
	RegistryURL  string          `json:"registryUrl" jsonschema:"required,loopback registry HTTP origin"`
	DiskPath     string          `json:"diskPath" jsonschema:"required,absolute build-volume path"`
	MinFreeBytes uint64          `json:"minFreeBytes,omitempty"`
	Services     []doctor.Target `json:"services,omitempty"`
	AI           bool            `json:"ai,omitempty"`
}

var profileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}\.json$`)

func (cfg deployConfig) load(profile string) (deploy.Spec, error) {
	if cfg.ProfileDir == "" || !filepath.IsAbs(cfg.ProfileDir) {
		return deploy.Spec{}, errors.New("deployment_profiles_not_configured")
	}
	if !profileNamePattern.MatchString(profile) {
		return deploy.Spec{}, errors.New("invalid_deployment_profile_name")
	}
	base, err := filepath.EvalSymlinks(cfg.ProfileDir)
	if err != nil {
		return deploy.Spec{}, errors.New("deployment_profile_directory_unavailable")
	}
	file, err := filepath.EvalSymlinks(filepath.Join(base, profile))
	if err != nil {
		return deploy.Spec{}, errors.New("deployment_profile_unavailable")
	}
	rel, err := filepath.Rel(base, file)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return deploy.Spec{}, errors.New("deployment_profile_outside_directory")
	}
	return deploy.LoadSpec(file)
}

func newServerWithDeploy(service *infisical.Service, runner *async.Runner, cfg deployConfig) *mcp.Server {
	return newServerWithServices(service, runner, cfg, doctor.Service{})
}

func newServerWithServices(service *infisical.Service, runner *async.Runner, cfg deployConfig, diagnostics doctor.Service) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "kuchdesk-infisical", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "infisical_capabilities", Description: "Show implemented Infisical tools and local credential configuration without accessing secrets.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(context.Context, *mcp.CallToolRequest, capabilityInput) (*mcp.CallToolResult, capabilityOutput, error) {
			return nil, capabilityOutput{Configured: service.Configured(), Capabilities: []capability{
				{Name: "list_secret_names", Status: "implemented; live permissions unverified"},
				{Name: "async_name_listing", Status: "implemented in memory; bounded to four concurrent operations and 64 records"},
				{Name: "secret_values", Status: "not exposed to MCP"},
				{Name: "secret_writes", Status: "planned; not implemented"},
				{Name: "project_admin", Status: "planned; not implemented"},
				{Name: "helm_deploy", Status: "implemented as opt-in local profile workflow; live rollout unverified"},
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
	mcp.AddTool(server, &mcp.Tool{Name: "kuchdesk_deploy_plan", Description: "Read a configured local profile and return its deployment plan without running commands.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(_ context.Context, _ *mcp.CallToolRequest, input deployInput) (*mcp.CallToolResult, deployPlanOutput, error) {
			spec, err := cfg.load(input.Profile)
			if err != nil {
				return nil, deployPlanOutput{}, err
			}
			steps, err := spec.Plan()
			if err != nil {
				return nil, deployPlanOutput{}, err
			}
			return nil, deployPlanOutput{Release: spec.Release, Namespace: spec.Namespace, KubeContext: spec.KubeContext, Image: spec.ImageRepository + ":" + spec.ImageTag, Steps: steps}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "kuchdesk_deploy_start", Description: "Start an opt-in local build, registry push, Helm upgrade, and rollout check from a named profile. Requires exact release/namespace confirmation."},
		func(_ context.Context, _ *mcp.CallToolRequest, input deployStartInput) (*mcp.CallToolResult, jobStarted, error) {
			if !cfg.Enabled {
				return nil, jobStarted{}, errors.New("deployment_not_enabled")
			}
			spec, err := cfg.load(input.Profile)
			if err != nil {
				return nil, jobStarted{}, err
			}
			if input.Confirm != spec.Release+"/"+spec.Namespace {
				return nil, jobStarted{}, errors.New("exact_release_namespace_confirmation_required")
			}
			if cfg.Runner == nil {
				return nil, jobStarted{}, errors.New("deployment_runner_unavailable")
			}
			key := spec.KubeContext + "/" + spec.Namespace + "/" + spec.Release
			id, err := runner.Submit(key, 20*time.Minute, func(ctx context.Context) (any, error) {
				result, err := (deploy.Workflow{Runner: cfg.Runner}).Apply(ctx, spec)
				if err != nil {
					return nil, err
				}
				return result, nil
			})
			if err != nil {
				return nil, jobStarted{}, publicError(err)
			}
			return nil, jobStarted{ID: id}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "kuchdesk_job_status", Description: "Read a bounded in-process KuchDesk operation status.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(_ context.Context, _ *mcp.CallToolRequest, input jobInput) (*mcp.CallToolResult, jobOutput, error) {
			status, err := runner.Status(input.ID)
			if err != nil {
				return nil, jobOutput{}, publicError(err)
			}
			return nil, outputFor(status), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "kuchdesk_cancel_job", Description: "Request cancellation of a queued or running KuchDesk operation."},
		func(_ context.Context, _ *mcp.CallToolRequest, input jobInput) (*mcp.CallToolResult, jobOutput, error) {
			status, err := runner.Cancel(input.ID)
			if err != nil {
				return nil, jobOutput{}, publicError(err)
			}
			return nil, outputFor(status), nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "kuchdesk_doctor", Description: "Run bounded read-only Mac, DNS, Docker, KIND, registry, resource, and Deployment checks. Optional AI is unavailable until a provider is explicitly configured.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, input doctorInput) (*mcp.CallToolResult, doctor.Report, error) {
			config := doctor.Config{Domain: input.Domain, KubeContext: input.KubeContext, RegistryURL: input.RegistryURL, DiskPath: input.DiskPath, MinFreeBytes: input.MinFreeBytes, Services: input.Services}
			report, err := diagnostics.Run(ctx, config, input.AI)
			if err != nil {
				return nil, doctor.Report{}, err
			}
			return nil, report, nil
		})
	return server
}

func outputFor(status async.Snapshot) jobOutput {
	output := jobOutput{ID: status.ID, State: status.State, Error: status.Error}
	if names, ok := status.Result.([]infisical.SecretName); ok {
		output.Secrets = names
	}
	if deployment, ok := status.Result.(deploy.Result); ok {
		output.Deployment = &deployment
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
