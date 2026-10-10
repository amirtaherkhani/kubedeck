package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnrollmentCLIRejectsIncompleteOrUnsafeSetup(t *testing.T) {
	for _, args := range [][]string{nil, {"-url", "https://example.invalid"}, {"-url", "https://user:password@example.invalid", "-policy", "/unused", "-state-dir", "/unused"}, {"-unexpected"}} {
		var out, errOut bytes.Buffer
		if e := run(args, &out, &errOut); e == nil {
			t.Fatal("unsafe setup accepted")
		}
		if out.Len() != 0 {
			t.Fatal("output before validated setup")
		}
	}
}

func TestStatusWithoutSessionNeedsNoMachineCredentialsAndDoesNotCreateLock(t *testing.T) {
	dir := t.TempDir()
	os.Chmod(dir, 0700)
	path := filepath.Join(dir, "policy.json")
	os.WriteFile(path, []byte(`{"version":"v0.151.0","organizationId":"org","hostIdentityId":"host","k8IdentityId":"k8","intervalSeconds":30,"maxProjects":20}`), 0600)
	var out, errOut bytes.Buffer
	e := run([]string{"-status", "-url", "https://example.invalid", "-policy", path, "-state-dir", dir}, &out, &errOut)
	if e == nil || !strings.Contains(out.String(), `"status":"session_missing"`) {
		t.Fatalf("missing-session status: %v %s", e, out.String())
	}
	if _, e := os.Stat(filepath.Join(dir, "controller.lock")); !os.IsNotExist(e) {
		t.Fatal("status created lock")
	}
}
func TestLoginModesAreMutuallyExclusive(t *testing.T) {
	for _, flags := range [][]string{{"-login", "-status"}, {"-status", "-once"}, {"-login", "-import-session", "/unused"}} {
		var out, errOut bytes.Buffer
		args := append([]string{"-url", "https://example.invalid", "-policy", "/unused", "-state-dir", "/unused"}, flags...)
		if e := run(args, &out, &errOut); e == nil || e.Error() != "choose_one_enrollment_mode" {
			t.Fatalf("mode conflict: %v", e)
		}
	}
}
