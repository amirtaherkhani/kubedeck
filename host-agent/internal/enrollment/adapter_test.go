package enrollment

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDiscoveryPaginationAndOrgBoundary(t *testing.T) {
	pages := 0
	wrongOrg := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/universal-auth/login":
			fmt.Fprint(w, `{"accessToken":"machine-test","expiresIn":3600,"tokenType":"Bearer"}`)
		case "/api/v1/organization-admin/projects":
			pages++
			if r.Header.Get("Authorization") != "Bearer machine-test" {
				t.Error("discovery used non-machine auth")
			}
			org := "org"
			if wrongOrg {
				org = "other"
			}
			projects := []Project{}
			count := 101
			if r.URL.Query().Get("offset") == "0" {
				for i := 0; i < 100; i++ {
					projects = append(projects, Project{ID: fmt.Sprintf("p-%d", i), OrganizationID: org})
				}
			} else {
				projects = append(projects, Project{ID: "last", OrganizationID: org})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"projects": projects, "count": count})
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	machine, _ := infisical.NewClient(srv.URL, "test-id", "test-bootstrap", srv.Client())
	h, _ := NewTransport(srv.URL, srv.Client())
	h.MinInterval = 0
	a := API{HTTP: h, Machine: machine, Version: "v0.151.0", OrganizationID: "org"}
	projects, e := a.Projects(context.Background(), 150)
	if e != nil || len(projects) != 101 || pages != 2 {
		t.Fatal("pagination failed", e)
	}
	wrongOrg = true
	if _, e = a.Projects(context.Background(), 150); e == nil {
		t.Fatal("foreign organization accepted")
	}
}
func TestHumanEnrollmentAndRefreshContracts(t *testing.T) {
	for _, version := range []string{"v0.151.0", "v0.166.3"} {
		t.Run(version, func(t *testing.T) {
			calls := 0
			token := testJWT(time.Now().Add(time.Hour))
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				calls++
				if r.URL.Path != "/api/v1/auth/token" {
					t.Error("wrong refresh endpoint")
				}
				cookie, e := r.Cookie("jid")
				if e != nil || cookie.Value != "test-refresh" {
					t.Error("missing refresh cookie")
				}
				if strings.Contains(r.URL.String(), "test-refresh") {
					t.Error("token in URL")
				}
				result := RefreshResult{Token: token, OrganizationID: "org"}
				if version == "v0.166.3" {
					result.RefreshToken = "rotated-test"
				}
				_ = json.NewEncoder(w).Encode(result)
			}))
			defer srv.Close()
			h, _ := NewTransport(srv.URL, srv.Client())
			h.MinInterval = 0
			m := &memoryStore{}
			s, _ := NewSession(SessionData{Version: version, OrganizationID: "org", RefreshToken: "test-refresh"}, m, refreshSession(h))
			if _, e := s.Token(context.Background()); e != nil {
				t.Fatal(e)
			}
			want := "test-refresh"
			if version == "v0.166.3" {
				want = "rotated-test"
			}
			if calls != 1 || m.data.RefreshToken != want {
				t.Fatal("wrong version refresh behavior")
			}
		})
	}
}
func TestRedirectRateLimitAndSanitizedErrors(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++ }))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, target.URL, 302)
			return
		}
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(429)
		fmt.Fprint(w, "private-test-value")
	}))
	defer srv.Close()
	h, _ := NewTransport(srv.URL, nil)
	h.MinInterval = 0
	e := h.Call(context.Background(), "GET", "/redirect", nil, http.Header{"Authorization": {"Bearer test"}}, nil, nil)
	if e == nil || targetCalls != 0 {
		t.Fatal("followed credential redirect")
	}
	e = h.Call(context.Background(), "GET", "/rate", nil, nil, nil, nil)
	status, ok := e.(*StatusError)
	if !ok || status.RetryAfter != 120*time.Second || strings.Contains(e.Error(), "private-test") {
		t.Fatal("rate limit or redaction failed")
	}
}

func TestSelfEnrollmentUsesOnlyHumanSession(t *testing.T) {
	token := testJWT(time.Now().Add(time.Hour))
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/api/v1/organization-admin/projects/external-project/grant-admin-access" || r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("self-enrollment contract or actor mismatch")
		}
		fmt.Fprint(w, `{"membership":{}}`)
	}))
	defer srv.Close()
	h, _ := NewTransport(srv.URL, srv.Client())
	h.MinInterval = 0
	session, _ := NewSession(SessionData{Version: "v0.151.0", OrganizationID: "org", AccessToken: token}, &memoryStore{}, func(context.Context, string) (RefreshResult, error) {
		t.Fatal("unneeded refresh")
		return RefreshResult{}, nil
	})
	a := API{HTTP: h, Human: session, OrganizationID: "org", Version: "v0.151.0"}
	if e := a.EnrollHuman(context.Background(), "external-project"); e != nil {
		t.Fatal(e)
	}
	a.Human = nil
	if e := a.EnrollHuman(context.Background(), "external-project"); e == nil || calls != 1 {
		t.Fatal("machine fallback entered human enrollment")
	}
	a.Version = "v0.166.3"
	if _, e := a.Projects(context.Background(), 10); e != ErrUnsupported {
		t.Fatal("unverified version accepted")
	}
}
