package infisical

import (
	"context"
	"errors"
	"net/http"
)

// Scope is explicit for every request; no default project or environment is
// inherited from the host process or a previous command.
type Scope struct {
	ProjectID   string `json:"projectId"`
	Environment string `json:"environment"`
	SecretPath  string `json:"secretPath"`
}

type NameLister interface {
	Configured() bool
	ListSecretNames(context.Context, string, string, string) ([]SecretName, error)
}

// Service is the small command boundary shared by the CLI and MCP transport.
// It contains no transport-specific output or credential handling.
type Service struct{ client NameLister }

func NewService(client NameLister) *Service { return &Service{client: client} }

func (s *Service) Configured() bool { return s.client.Configured() }

func (s *Service) ListSecretNames(ctx context.Context, scope Scope) ([]SecretName, error) {
	return s.client.ListSecretNames(ctx, scope.ProjectID, scope.Environment, scope.SecretPath)
}

// PublicError removes transport errors and response bodies before a CLI or MCP
// caller can see them. It must be extended as new command types are added.
func PublicError(err error) error {
	var apiErr *APIError
	switch {
	case errors.Is(err, ErrNotConfigured):
		return errors.New("credentials_not_configured")
	case errors.Is(err, ErrInvalidScope):
		return errors.New("invalid_scope")
	case errors.Is(err, ErrInvalidProjectSlug):
		return errors.New("invalid_project_slug")
	case errors.Is(err, ErrInvalidCommand):
		return errors.New("invalid_command")
	case errors.Is(err, ErrProjectDenied):
		return errors.New("project_denied")
	case errors.Is(err, ErrConfirmation):
		return errors.New("confirmation_required")
	case errors.Is(err, ErrAuthRejected):
		return errors.New("authentication_rejected")
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden:
		return errors.New("permission_denied")
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
		return errors.New("not_found")
	default:
		return errors.New("infisical_unavailable")
	}
}
