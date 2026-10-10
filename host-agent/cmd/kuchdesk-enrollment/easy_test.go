package main

import (
	"bytes"
	"context"
	"errors"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/enrollment"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEasyLoginResolvesSpacedProjectAndMissingPrerequisites(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project with spaces")
	if _, e := loadProjectLogin(root); e == nil {
		t.Fatal("missing project accepted")
	}
	os.MkdirAll(filepath.Join(root, "host-agent"), 0700)
	os.MkdirAll(filepath.Join(root, "lab"), 0700)
	os.WriteFile(filepath.Join(root, "host-agent", "go.mod"), []byte("module test"), 0600)
	if _, e := loadProjectLogin(root); e == nil {
		t.Fatal("missing site accepted")
	}
	site := filepath.Join(root, "lab", "site.json")
	os.WriteFile(site, []byte(`{"domain":"example.test","hosts":{"infisical":"vault"}}`), 0600)
	got, e := loadProjectLogin(root)
	if e != nil || got.origin != "https://vault.example.test" || !filepath.IsAbs(got.policy) || !strings.Contains(got.state, "project with spaces") {
		t.Fatalf("bad project resolution: %+v %v", got, e)
	}
	os.WriteFile(site, []byte(`{"domain":"example.test/steal","hosts":{"infisical":"vault"}}`), 0600)
	if _, e := loadProjectLogin(root); e == nil {
		t.Fatal("unsafe origin accepted")
	}
}
func TestEasyLoginOpenerFailureKeepsExactFreshURL(t *testing.T) {
	var out bytes.Buffer
	u := "https://vault.example.test/login?callback_port=54321"
	if e := announceEasyLogin(context.Background(), &out, u, "mock-org", func(context.Context, string) error { return errors.New("unavailable") }); e != nil {
		t.Fatal(e)
	}
	for _, want := range []string{u, "mock-org", "30 seconds", "hidden prompt", "Browser opening failed"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing instruction: %s", want)
		}
	}
}

func TestEasyLoginDoesNotOpenBrowserWhileWriterIsActive(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "host-agent"), 0700)
	os.MkdirAll(filepath.Join(root, "lab"), 0700)
	os.WriteFile(filepath.Join(root, "host-agent", "go.mod"), []byte("module test"), 0600)
	os.WriteFile(filepath.Join(root, "lab", "site.json"), []byte(`{"domain":"example.test","hosts":{"infisical":"vault"}}`), 0600)
	state := filepath.Join(root, ".kuchdesk", "infisical-control")
	os.MkdirAll(state, 0700)
	os.WriteFile(filepath.Join(state, "policy.json"), []byte(`{"version":"v0.151.0","organizationId":"org","hostIdentityId":"host","k8IdentityId":"k8","intervalSeconds":30,"maxProjects":20}`), 0600)
	store, e := enrollment.OpenStore(state)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	var out bytes.Buffer
	e = runWithBrowser([]string{"-login-easy", "-project-root", root}, &out, &out, func(context.Context, string) error { t.Fatal("duplicate login opened browser"); return nil })
	if e == nil || e.Error() != "controller_already_running" {
		t.Fatalf("expected writer lock rejection: %v", e)
	}
}
