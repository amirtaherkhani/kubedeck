package hostbridge

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/doctor"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
)

type Commander interface {
	Execute(context.Context, infisical.Command) (infisical.CommandResult, error)
}

type Diagnostician interface {
	Run(context.Context, doctor.Config) (doctor.Report, error)
}

// Server handles one authenticated host command surface. It is independent of
// listener setup, so the network service can remain disabled by default.
type Server struct {
	Commands Commander
	Doctor   Diagnostician
	Token    string
	slots    chan struct{}
}

func New(commands Commander, diagnostics Diagnostician, token string) (*Server, error) {
	if len(token) < 32 || commands == nil || diagnostics == nil {
		return nil, errors.New("host bridge requires commands, Doctor, and a private bearer token")
	}
	return &Server{Commands: commands, Doctor: diagnostics, Token: token, slots: make(chan struct{}, 8)}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/infisical/commands", s.command)
	mux.HandleFunc("POST /v1/doctor", s.doctor)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path != "/healthz" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			want := sha256.Sum256([]byte(s.Token))
			got := sha256.Sum256([]byte(provided))
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		default:
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "capacity_reached"})
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) command(w http.ResponseWriter, r *http.Request) {
	var command infisical.Command
	if !decode(w, r, &command) {
		return
	}
	if !infisical.ReadOperation(command.Operation) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "read_only_bridge"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	result, err := s.Commands.Execute(ctx, command)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) doctor(w http.ResponseWriter, r *http.Request) {
	var cfg doctor.Config
	if !decode(w, r, &cfg) {
		return
	}
	if cfg.Validate() != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_doctor_config"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	report, err := s.Doctor.Run(ctx, cfg)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "doctor_unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func decode(w http.ResponseWriter, r *http.Request, output any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	code := "infisical_unavailable"
	var api *infisical.APIError
	switch {
	case errors.Is(err, infisical.ErrInvalidCommand), errors.Is(err, infisical.ErrInvalidScope):
		status, code = http.StatusBadRequest, "invalid_command"
	case errors.Is(err, infisical.ErrProjectDenied):
		status, code = http.StatusForbidden, "project_denied"
	case errors.Is(err, infisical.ErrConfirmation):
		status, code = http.StatusConflict, "confirmation_required"
	case errors.As(err, &api) && api.StatusCode == http.StatusForbidden:
		status, code = http.StatusForbidden, "upstream_permission_denied"
	case errors.As(err, &api) && api.StatusCode == http.StatusNotFound:
		status, code = http.StatusNotFound, "not_found"
	case errors.As(err, &api) && api.StatusCode == http.StatusConflict:
		status, code = http.StatusConflict, "upstream_conflict"
	case errors.As(err, &api) && api.StatusCode == http.StatusTooManyRequests:
		status, code = http.StatusTooManyRequests, "upstream_rate_limited"
	}
	writeJSON(w, status, map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
