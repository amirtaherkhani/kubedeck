package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLINameOnlyAndMissingCredential(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/universal-auth/login":
			fmt.Fprint(w, `{"accessToken":"token","expiresIn":60,"tokenType":"Bearer"}`)
		case "/api/v4/secrets":
			fmt.Fprint(w, `{"secrets":[{"secretKey":"DB_USER","secretPath":"/app","secretValue":"never-print-me"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	// The production CLI uses default TLS verification. Its rejection of this
	// test server proves it will not silently bypass certificate validation.
	var out, errOut bytes.Buffer
	args := []string{"-url", server.URL, "list-secret-names", "-project", "p", "-environment", "dev", "-path", "/app"}
	code := run(context.Background(), args, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 1 || out.Len() != 0 || strings.Contains(errOut.String(), "secret") {
		t.Fatalf("untrusted TLS endpoint should fail without leaking data: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	emptyRoot := t.TempDir()
	code = run(context.Background(), []string{"-url", server.URL, "capabilities"}, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return emptyRoot
		}
		return ""
	})
	if code != 0 || !strings.Contains(out.String(), `"configured":false`) {
		t.Fatalf("offline discovery failed: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), args, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return emptyRoot
		}
		return ""
	})
	if code != 1 || strings.TrimSpace(errOut.String()) != "credentials_not_configured" {
		t.Fatalf("missing credential should be explicit: code=%d err=%q", code, errOut.String())
	}
}

func TestCLIManageReadsStdinAndNeverEchoesSecretValue(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/universal-auth/login" {
			fmt.Fprint(w, `{"accessToken":"token","expiresIn":60,"tokenType":"Bearer"}`)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/v4/secrets/API_KEY" {
			t.Errorf("unexpected management request: %s %s", r.Method, r.URL.Path)
		}
		fmt.Fprint(w, `{"secret":{"secretValue":"upstream-secret"}}`)
	}))
	defer server.Close()
	input := `{"operation":"secret.create","projectId":"project-1","environment":"dev","path":"/app","name":"API_KEY","value":"input-secret","confirm":"secret.create:project-1/dev/app/API_KEY"}`
	var out, errOut bytes.Buffer
	code := runWithInput(context.Background(), []string{"-url", server.URL, "manage", "-project-ids", "project-1"}, strings.NewReader(input), &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 0 || !strings.Contains(out.String(), `"applied":true`) || strings.Contains(out.String()+errOut.String(), "input-secret") || strings.Contains(out.String()+errOut.String(), "upstream-secret") {
		t.Fatalf("management output was not redacted: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestCLIRejectsIncompleteScope(t *testing.T) {
	root := t.TempDir()
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"-url", "https://infisical.example", "list-secret-names", "-project", "p", "-environment", "dev", "-path", "relative"}, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 2 || strings.TrimSpace(errOut.String()) != "invalid_scope" {
		t.Fatalf("invalid scope should fail before a request: code=%d err=%q", code, errOut.String())
	}
}

func TestCLIReadsProjectLocalCredentialFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=sensitive-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"-url", "https://infisical.example", "capabilities"}, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 0 || !strings.Contains(out.String(), `"configured":true`) || strings.Contains(out.String()+errOut.String(), "sensitive-value") {
		t.Fatalf("private file not consumed safely: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestCLINameOnlySuccess(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/universal-auth/login" {
			fmt.Fprint(w, `{"accessToken":"token","expiresIn":60,"tokenType":"Bearer"}`)
			return
		}
		if r.URL.Query().Get("viewSecretValue") != "false" {
			t.Error("value view was not disabled")
		}
		fmt.Fprint(w, `{"secrets":[{"secretKey":"DB_USER","secretPath":"/app","secretValue":"never-print-me"}]}`)
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"-url", server.URL, "list-secret-names", "-project", "p", "-environment", "dev", "-path", "/app"}, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 0 || !strings.Contains(out.String(), "DB_USER") || strings.Contains(out.String(), "never-print-me") || errOut.Len() != 0 {
		t.Fatalf("unexpected CLI result: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}

func TestCLIProjectAccessDoesNotEquateEmptyListWithMissingProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=test-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/universal-auth/login":
			fmt.Fprint(w, `{"accessToken":"token","expiresIn":60,"tokenType":"Bearer"}`)
		case "/api/v1/projects":
			fmt.Fprint(w, `{"projects":[]}`)
		case "/api/v1/projects/slug/home-lab":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"secret":"never-print-me"}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	code := run(context.Background(), []string{"-url", server.URL, "check-project-access", "-slug", "home-lab"}, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 0 || errOut.Len() != 0 || !strings.Contains(out.String(), `"listed":false`) || !strings.Contains(out.String(), `"detailStatus":"forbidden"`) || strings.Contains(out.String(), "never-print-me") {
		t.Fatalf("unexpected project access result: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code = run(context.Background(), []string{"-url", server.URL, "check-project-access", "-slug", "../home-lab"}, &out, &errOut, func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	})
	if code != 2 || out.Len() != 0 || strings.TrimSpace(errOut.String()) != "invalid_project_slug" {
		t.Fatalf("unsafe slug should be rejected locally: code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}
