package infisical

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestListSecretNamesRenewsAndRedacts(t *testing.T) {
	var logins atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/universal-auth/login":
			logins.Add(1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"accessToken":"token-%d","expiresIn":100,"tokenType":"Bearer"}`, logins.Load())
		case "/api/v4/secrets":
			if r.URL.Query().Get("projectId") != "project-1" || r.URL.Query().Get("environment") != "dev" || r.URL.Query().Get("secretPath") != "/app" || r.URL.Query().Get("viewSecretValue") != "false" || r.URL.Query().Get("recursive") != "false" {
				t.Errorf("unexpected scope or value-view query: %s", r.URL.RawQuery)
			}
			if r.Header.Get("Authorization") != fmt.Sprintf("Bearer token-%d", logins.Load()) {
				t.Errorf("unexpected authorization header")
			}
			fmt.Fprint(w, `{"secrets":[{"secretKey":"DB_USER","secretPath":"/app","secretValue":"must-never-return"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "id", "secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1000, 0)
	client.now = func() time.Time { return now }
	for _, advance := range []time.Duration{0, 81 * time.Second} {
		now = now.Add(advance)
		got, err := client.ListSecretNames(context.Background(), "project-1", "dev", "/app")
		if err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(got)
		if len(got) != 1 || got[0].Name != "DB_USER" || strings.Contains(string(encoded), "must-never-return") {
			t.Fatalf("unexpected name-only response: %s", encoded)
		}
	}
	if got := logins.Load(); got != 2 {
		t.Fatalf("want re-login after the renewal threshold, got %d logins", got)
	}
}

func TestUnauthorizedReadRetriesOnce(t *testing.T) {
	var logins, reads atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/universal-auth/login":
			fmt.Fprintf(w, `{"accessToken":"token-%d","expiresIn":100,"tokenType":"Bearer"}`, logins.Add(1))
		case "/api/v4/secrets":
			if reads.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"secretValue":"do-not-log"}`)
				return
			}
			fmt.Fprint(w, `{"secrets":[]}`)
		}
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "id", "secret", server.Client())
	if _, err := client.ListSecretNames(context.Background(), "project", "dev", "/"); err != nil {
		t.Fatal(err)
	}
	if logins.Load() != 2 || reads.Load() != 2 {
		t.Fatalf("expected one safe 401 replay, got %d logins and %d reads", logins.Load(), reads.Load())
	}
}

func TestRejectedCredentialsDoNotCauseLoginStorm(t *testing.T) {
	var logins atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"clientSecret":"do-not-log"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "id", "secret", server.Client())
	for i := 0; i < 3; i++ {
		_, err := client.ListSecretNames(context.Background(), "project", "dev", "/")
		if !errors.Is(err, ErrAuthRejected) || strings.Contains(err.Error(), "do-not-log") {
			t.Fatalf("expected redacted auth rejection, got %v", err)
		}
	}
	if logins.Load() != 1 {
		t.Fatalf("expected one login, got %d", logins.Load())
	}
}

func TestForbiddenReadDoesNotRelogin(t *testing.T) {
	var logins, reads atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/universal-auth/login" {
			logins.Add(1)
			fmt.Fprint(w, `{"accessToken":"token","expiresIn":100,"tokenType":"Bearer"}`)
			return
		}
		reads.Add(1)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"secretValue":"do-not-log"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "id", "secret", server.Client())
	_, err := client.ListSecretNames(context.Background(), "project", "dev", "/")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden || strings.Contains(err.Error(), "do-not-log") {
		t.Fatalf("expected redacted 403, got %v", err)
	}
	if logins.Load() != 1 || reads.Load() != 1 {
		t.Fatalf("403 caused retry: %d logins and %d reads", logins.Load(), reads.Load())
	}
}

func TestTransientRenewalUsesUnexpiredToken(t *testing.T) {
	var logins atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/universal-auth/login" {
			if logins.Add(1) == 1 {
				fmt.Fprint(w, `{"accessToken":"token","expiresIn":100,"tokenType":"Bearer"}`)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Error("unexpired token was not retained")
		}
		fmt.Fprint(w, `{"secrets":[]}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "id", "secret", server.Client())
	now := time.Unix(1000, 0)
	client.now = func() time.Time { return now }
	if _, err := client.ListSecretNames(context.Background(), "project", "dev", "/"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(81 * time.Second)
	if _, err := client.ListSecretNames(context.Background(), "project", "dev", "/"); err != nil {
		t.Fatalf("transient renewal failure interrupted valid token: %v", err)
	}
	if logins.Load() != 2 {
		t.Fatalf("expected one failed renewal, got %d login calls", logins.Load())
	}
}

func TestConcurrentReadsShareLogin(t *testing.T) {
	var logins atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/universal-auth/login" {
			logins.Add(1)
			time.Sleep(20 * time.Millisecond)
			fmt.Fprint(w, `{"accessToken":"token","expiresIn":100,"tokenType":"Bearer"}`)
			return
		}
		fmt.Fprint(w, `{"secrets":[]}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, "id", "secret", server.Client())
	var group sync.WaitGroup
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := client.ListSecretNames(context.Background(), "project", "dev", "/"); err != nil {
				t.Errorf("read: %v", err)
			}
		}()
	}
	group.Wait()
	if logins.Load() != 1 {
		t.Fatalf("concurrent readers caused %d logins", logins.Load())
	}
}

func TestInvalidConfigAndScope(t *testing.T) {
	for _, endpoint := range []string{"", "http://infisical.example", "https://user:password@infisical.example", "https://infisical.example/api"} {
		if _, err := NewClient(endpoint, "id", "secret", nil); err == nil {
			t.Errorf("accepted unsafe endpoint %q", endpoint)
		}
	}
	if _, err := NewClient("https://infisical.example", "id", "", nil); err == nil {
		t.Fatal("accepted incomplete credential pair")
	}
	client, err := NewClient("https://infisical.example", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListSecretNames(context.Background(), "project", "dev", "relative"); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("expected invalid scope, got %v", err)
	}
	if _, err := client.ListSecretNames(context.Background(), "project", "dev", "/"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected missing credentials, got %v", err)
	}
}

func TestLoginRedirectDoesNotForwardCredential(t *testing.T) {
	var forwarded atomic.Bool
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		forwarded.Store(true)
	}))
	defer target.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client, err := NewClient(source.URL, "id", "secret", source.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListSecretNames(context.Background(), "project", "dev", "/"); err == nil {
		t.Fatal("expected login redirect to fail")
	}
	if forwarded.Load() {
		t.Fatal("login credentials were forwarded to redirect target")
	}
}
