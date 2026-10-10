package platform

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileNamesDoNotSelectProjects(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"", "Example Platform", "پلتفرم ابزار", "  Base Tools  "} {
		p, e := Load(func(k string) string {
			if k == "KUCHDESK_PROJECT_ROOT" {
				return root
			}
			if k == "KUCHDESK_BASE_PROJECT_NAME" {
				return name
			}
			return ""
		})
		if e != nil || p.ProjectID != "" || p.Slug != "" {
			t.Fatal("display name selected a project")
		}
		if name == "" && p.Name != DefaultName {
			t.Fatal("default mismatch")
		}
	}
	for _, name := range []string{"  ", "bad\nname"} {
		if _, e := Load(func(string) string { return name }); e == nil {
			t.Fatal("invalid name accepted")
		}
	}
}

func TestProjectMetadataFilePrecedenceAndNoSideEffects(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("INFISICAL_CLIENT_ID=fixture-id\nINFISICAL_CLIENT_SECRET=fixture-sensitive\nKUCHDESK_BASE_PROJECT_NAME=پلتفرم ابزار\nKUCHDESK_BASE_PROJECT_ID=file-id\nKUCHDESK_BASE_PROJECT_SLUG=file-slug\nUNRELATED=$(touch should-never-exist)\n")
	if e := os.WriteFile(path, original, 0600); e != nil {
		t.Fatal(e)
	}
	env := map[string]string{"KUCHDESK_PROJECT_ROOT": root}
	get := func(k string) string { return env[k] }
	p, e := Load(get)
	if e != nil || p.Name != "پلتفرم ابزار" || p.ProjectID != "file-id" || p.Slug != "file-slug" {
		t.Fatal("project metadata unavailable", e)
	}
	env["KUCHDESK_BASE_PROJECT_NAME"] = "Process Name"
	env["KUCHDESK_BASE_PROJECT_ID"] = "process-id"
	p, e = Load(get)
	if e != nil || p.Name != "Process Name" || p.ProjectID != "process-id" || p.Slug != "file-slug" {
		t.Fatal("process precedence failed", e)
	}
	env["KUCHDESK_BASE_PROJECT_NAME"] = ""
	p, e = Load(get)
	if e != nil || p.Name != "پلتفرم ابزار" {
		t.Fatal("empty process value prevented file fallback", e)
	}
	after, e := os.ReadFile(path)
	if e != nil || !bytes.Equal(after, original) {
		t.Fatal("loader changed the private file")
	}
	if os.Getenv("UNRELATED") == "$(touch should-never-exist)" {
		t.Fatal("loader changed process environment")
	}
	if e := os.WriteFile(path, []byte("KUCHDESK_BASE_PROJECT_NAME=Changed Locally\n"), 0600); e != nil {
		t.Fatal(e)
	}
	p, e = Load(get)
	if e != nil || p.Name != "Changed Locally" || p.ProjectID != "process-id" {
		t.Fatal("metadata reload lost environment precedence", e)
	}
}

func TestProjectMetadataRejectsUnsafeFilesAndDuplicateKeys(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	get := func(k string) string {
		if k == "KUCHDESK_PROJECT_ROOT" {
			return root
		}
		return ""
	}
	for _, content := range []string{
		"KUCHDESK_BASE_PROJECT_NAME=one\nKUCHDESK_BASE_PROJECT_NAME=fixture-sensitive\n",
		"KUCHDESK_BASE_PROJECT_NAME=fixture-sensitive\ninvalid line\n",
		strings.Repeat("x", 4097),
	} {
		if e := os.WriteFile(path, []byte(content), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := Load(get); e == nil || strings.Contains(e.Error(), "fixture-sensitive") {
			t.Fatal("unsafe file accepted or contents exposed")
		}
	}
	if e := os.WriteFile(path, []byte("KUCHDESK_BASE_PROJECT_NAME=Safe\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(get); e == nil {
		t.Fatal("public .env accepted")
	}
	if e := os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(filepath.Join(t.TempDir(), "absent"), path); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(get); e == nil {
		t.Fatal("symlink accepted")
	}
}
