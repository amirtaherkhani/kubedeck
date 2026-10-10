package infisical

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureProject(t *testing.T, content string) string {
	t.Helper()
	root := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func projectEnv(root string) func(string) string {
	return func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	}
}

func TestDefaultProjectEnvAndNoEnvironmentFallback(t *testing.T) {
	root := fixtureProject(t, "# shared project settings\nOTHER=value\nINFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=sensitive-value\n")
	id, secret, err := HostCredentials(projectEnv(root))
	if err != nil || id != "id" || secret != "sensitive-value" {
		t.Fatalf("project credentials unavailable: %v", err)
	}
	getenv := func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		if key == "INFISICAL_CLIENT_ID" {
			return "unselected-id"
		}
		if key == "INFISICAL_CLIENT_SECRET" {
			return "unselected-secret"
		}
		return ""
	}
	id, secret, err = HostCredentials(getenv)
	if err != nil || id != "id" || secret != "sensitive-value" {
		t.Fatal("unselected process values overrode project backend")
	}
	if _, err := os.Stat(filepath.Join(root, ".env")); err != nil {
		t.Fatal("existing .env changed")
	}
}

func TestMissingProjectEnvIsUnconfiguredWithoutFallback(t *testing.T) {
	root := fixtureProject(t, "")
	id, secret, err := HostCredentials(projectEnv(root))
	if err != nil || id != "" || secret != "" {
		t.Fatalf("missing .env should be unconfigured: %v", err)
	}
}

func TestHostBridgeTokenUsesPrivateProjectFile(t *testing.T) {
	root := fixtureProject(t, "INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=secret\nKUCHDESK_HOST_BRIDGE_TOKEN=0123456789abcdef0123456789abcdef\n")
	token, err := HostBridgeToken(projectEnv(root))
	if err != nil || len(token) != 32 {
		t.Fatalf("bridge token unavailable: %v", err)
	}
	if err := os.Chmod(filepath.Join(root, ".env"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := HostBridgeToken(projectEnv(root)); err == nil {
		t.Fatal("bridge token read from unsafe file")
	}
}

func TestProjectEnvRejectsUnsafeMetadataAndPartialValues(t *testing.T) {
	root := fixtureProject(t, "INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=sensitive-value\n")
	path := filepath.Join(root, ".env")
	assertRejected := func() {
		t.Helper()
		_, _, err := HostCredentials(projectEnv(root))
		if err == nil || strings.Contains(err.Error(), "sensitive-value") {
			t.Fatalf("unsafe file accepted or value leaked: %v", err)
		}
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	assertRejected()
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{
		"INFISICAL_CLIENT_ID=id\n",
		"INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_ID=second\nINFISICAL_CLIENT_SECRET=sensitive-value\n",
		"INFISICAL_CLIENT_ID=id\nINFISICAL_CLIENT_SECRET=sensitive-value\ninvalid line\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
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
}

func TestBackendSelectionIsExplicitAndStrict(t *testing.T) {
	root := fixtureProject(t, "")
	if _, err := SelectCredentialBackend(func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		if key == "KUCHDESK_INFISICAL_CREDENTIAL_BACKEND" {
			return "unknown"
		}
		return ""
	}); err == nil {
		t.Fatal("unknown backend accepted")
	}
	if _, err := SelectCredentialBackend(func(key string) string {
		if key == "KUCHDESK_INFISICAL_CREDENTIAL_BACKEND" {
			return "keychain"
		}
		return ""
	}); err == nil {
		t.Fatal("Keychain selected without explicit account")
	}
	if _, err := SelectCredentialBackend(func(key string) string {
		if key == "KUCHDESK_PROJECT_ROOT" {
			return "relative"
		}
		return ""
	}); err == nil {
		t.Fatal("relative project root accepted")
	}
}
