package infisical

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testHostEnv(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "infisical", "host.env")
}

func TestHostCredentialsPrivateFileAndEnvironmentOverride(t *testing.T) {
	path := testHostEnv(t)
	if err := SaveHostCredentials(path, "client-id", "client-secret"); err != nil {
		t.Fatal(err)
	}
	for target, wanted := range map[string]os.FileMode{filepath.Dir(path): 0o700, path: 0o600} {
		info, err := os.Stat(target)
		if err != nil || info.Mode().Perm() != wanted {
			t.Fatalf("incorrect private permissions on %s: %v %v", target, info, err)
		}
	}
	getenv := func(key string) string {
		if key == "KUCHDESK_INFISICAL_ENV_FILE" {
			return path
		}
		return ""
	}
	id, secret, err := HostCredentials(getenv)
	if err != nil || id != "client-id" || secret != "client-secret" {
		t.Fatal("private credentials did not round trip")
	}
	getenv = func(key string) string {
		switch key {
		case "KUCHDESK_INFISICAL_ENV_FILE":
			return path
		case "INFISICAL_CLIENT_ID":
			return "override-id"
		case "INFISICAL_CLIENT_SECRET":
			return "override-secret"
		default:
			return ""
		}
	}
	id, secret, err = HostCredentials(getenv)
	if err != nil || id != "override-id" || secret != "override-secret" {
		t.Fatal("complete process pair did not override file")
	}
}

func TestHostEnvRejectsUnsafeMetadataAndContentWithoutLeak(t *testing.T) {
	path := testHostEnv(t)
	if err := SaveHostCredentials(path, "id", "sensitive-value"); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "KUCHDESK_INFISICAL_ENV_FILE" {
			return path
		}
		return ""
	}
	assertRejected := func() {
		t.Helper()
		_, _, err := HostCredentials(getenv)
		if err == nil || strings.Contains(err.Error(), "sensitive-value") {
			t.Fatalf("unsafe file accepted or leaked value: %v", err)
		}
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	assertRejected()
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{
		"INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=sensitive-value\nEXTRA=bad\n",
		"INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_ID=duplicate\nINFISICAL_CLIENT_SECRET=sensitive-value\n",
		"INFISICAL_CLIENT_ID=id\n",
	} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		assertRejected()
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), path); err != nil {
		t.Fatal(err)
	}
	assertRejected()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=sensitive-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	assertRejected()
}

func TestSaveHostCredentialsNeverOverwritesOrLeavesTempFile(t *testing.T) {
	path := testHostEnv(t)
	var group sync.WaitGroup
	var success int
	var mu sync.Mutex
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if SaveHostCredentials(path, "id", "sensitive-value") == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}()
	}
	group.Wait()
	if success != 1 {
		t.Fatalf("expected one atomic setup, got %d", success)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != "host.env" {
		t.Fatalf("temporary credential file remains: %v %v", entries, err)
	}
	if err := SaveHostCredentials(path, "new-id", "new-secret"); err == nil {
		t.Fatal("existing credentials overwritten")
	}
}

func TestHostCredentialsMissingAndPartialEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	id, secret, err := HostCredentials(func(string) string { return "" })
	if err != nil || id != "" || secret != "" {
		t.Fatalf("missing default should remain unconfigured: %v", err)
	}
	_, _, err = HostCredentials(func(key string) string {
		if key == "INFISICAL_CLIENT_ID" {
			return "id"
		}
		return ""
	})
	if err == nil {
		t.Fatal("partial process credentials accepted")
	}
	_, _, err = HostCredentials(func(key string) string {
		if key == "KUCHDESK_INFISICAL_ENV_FILE" {
			return "/missing/host.env"
		}
		return ""
	})
	if err == nil {
		t.Fatal("explicit missing file accepted")
	}
}
