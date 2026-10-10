package enrollment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionImportScopeAndPermissions(t *testing.T) {
	dir := t.TempDir()
	_ = os.Chmod(dir, 0700)
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	source := filepath.Join(dir, "handoff.json")
	p := testPolicy()
	d := SessionData{Origin: "https://example.invalid", OrganizationID: "org", Version: p.Version, AccessToken: testJWT(time.Now().Add(time.Hour)), RefreshToken: "test-refresh"}
	b, _ := json.Marshal(d)
	_ = os.WriteFile(source, b, 0600)
	if e = ImportSession(source, d.Origin, p, s); e != nil {
		t.Fatal(e)
	}
	if e = ImportSession(source, "https://other.invalid", p, s); e == nil {
		t.Fatal("cross-origin import")
	}
	_ = os.Chmod(source, 0644)
	if e = ImportSession(source, d.Origin, p, s); e == nil {
		t.Fatal("unsafe handoff file")
	}
}
func TestDoctorEnrollmentStatusIsValueFree(t *testing.T) {
	for _, state := range []string{"ready", "dry_run", "session_expired", "enrollment_required", "denied", ""} {
		c := DiagnosticCheck(Report{Status: state})
		want := "warn"
		if state == "ready" || state == "dry_run" {
			want = "ok"
		}
		if c.Status != want {
			t.Fatal("diagnostic classification")
		}
		b, _ := json.Marshal(c)
		if strings.Contains(string(b), "accessToken") || strings.Contains(string(b), "refreshToken") {
			t.Fatal("credential field in diagnostics")
		}
	}
}
func TestStoreRejectsSymlinkAndCorruptState(t *testing.T) {
	dir := t.TempDir()
	_ = os.Chmod(dir, 0700)
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	target := filepath.Join(t.TempDir(), "target")
	_ = os.WriteFile(target, []byte(`{}`), 0600)
	_ = os.Symlink(target, filepath.Join(dir, "session.json"))
	var d SessionData
	if e = s.Load("session.json", &d); e == nil {
		t.Fatal("symlink followed")
	}
	_ = os.WriteFile(filepath.Join(dir, "ledger.json"), []byte(`{broken`), 0600)
	if _, e = NewController(&fakeAPI{}, s, &Access{}, "", testPolicy()); e == nil {
		t.Fatal("corrupt journal ignored")
	}
}
