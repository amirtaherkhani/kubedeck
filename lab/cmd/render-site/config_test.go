package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func writeTestSite(t *testing.T, path string, profile siteProfile) {
	t.Helper()
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func configOutput(t *testing.T, args ...string) map[string]any {
	t.Helper()
	var buffer bytes.Buffer
	if err := runConfig(args, &buffer); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSiteConfigPlanApplyRollbackAndDryRun(t *testing.T) {
	root := t.TempDir()
	profilePath := filepath.Join(root, "site.json")
	candidatePath := filepath.Join(root, "candidate.json")
	original := loadProfile(t)
	writeTestSite(t, profilePath, original)
	candidate := original
	candidate.Domain = "example.internal"
	writeTestSite(t, candidatePath, candidate)
	before, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := configOutput(t, "snapshot", "-profile", profilePath)
	if snapshot["revision"] != siteRevision(before) {
		t.Fatal("snapshot revision mismatch")
	}
	validated := configOutput(t, "validate", "-profile", profilePath, "-candidate", candidatePath)
	if validated["valid"] != true {
		t.Fatal("candidate was not validated")
	}
	plan := configOutput(t, "plan", "-profile", profilePath, "-candidate", candidatePath)
	dryRun := configOutput(t, "dry-run", "-profile", profilePath, "-candidate", candidatePath)
	if plan["planId"] == "" || plan["planId"] != dryRun["planId"] || plan["baseRevision"] != snapshot["revision"] {
		t.Fatalf("plan and dry-run diverged: %v %v", plan, dryRun)
	}
	changed := plan["changedFields"].([]any)
	if len(changed) != 1 || changed[0] != "domain" {
		t.Fatalf("unexpected diff: %v", changed)
	}
	changes := plan["changes"].([]any)
	if len(changes) != 1 || changes[0].(map[string]any)["before"] != "local.dev" || changes[0].(map[string]any)["after"] != "example.internal" {
		t.Fatalf("diff values missing: %v", changes)
	}
	if current, _ := os.ReadFile(profilePath); !bytes.Equal(before, current) {
		t.Fatal("read-only operations changed site profile")
	}
	if err := runConfig([]string{"apply", "-profile", profilePath, "-candidate", candidatePath, "-confirm", "wrong"}, &bytes.Buffer{}); err == nil {
		t.Fatal("wrong confirmation accepted")
	}
	changedCandidate := candidate
	changedCandidate.Hosts.Grafana = "dashboard"
	writeTestSite(t, candidatePath, changedCandidate)
	if err := runConfig([]string{"apply", "-profile", profilePath, "-candidate", candidatePath, "-confirm", plan["planId"].(string)}, &bytes.Buffer{}); err == nil {
		t.Fatal("candidate changed after planning")
	}
	writeTestSite(t, candidatePath, candidate)
	applied := configOutput(t, "apply", "-profile", profilePath, "-candidate", candidatePath, "-confirm", plan["planId"].(string))
	if applied["applied"] != true || applied["rollbackRevision"] != snapshot["revision"] {
		t.Fatalf("apply result invalid: %v", applied)
	}
	if _, err := os.Stat(siteBackupPath(profilePath, snapshot["revision"].(string))); err != nil {
		t.Fatal("rollback revision missing")
	}
	if info, err := os.Stat(siteBackupPath(profilePath, snapshot["revision"].(string))); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("rollback revision not private: %v %v", info, err)
	}
	if err := runConfig([]string{"apply", "-profile", profilePath, "-candidate", candidatePath, "-confirm", plan["planId"].(string)}, &bytes.Buffer{}); err == nil {
		t.Fatal("stale plan accepted")
	}
	rolledBack := configOutput(t, "rollback", "-profile", profilePath, "-revision", snapshot["revision"].(string), "-confirm", applied["revision"].(string))
	if rolledBack["rolledBack"] != true || rolledBack["revision"] != snapshot["revision"] {
		t.Fatalf("rollback result invalid: %v", rolledBack)
	}
	current, err := os.ReadFile(profilePath)
	if err != nil || !bytes.Equal(current, before) {
		t.Fatal("rollback did not restore exact original bytes")
	}
}

func TestSiteConfigRejectsUnknownFieldsAndInvalidDomain(t *testing.T) {
	root := t.TempDir()
	profilePath := filepath.Join(root, "site.json")
	candidatePath := filepath.Join(root, "candidate.json")
	writeTestSite(t, profilePath, loadProfile(t))
	if err := os.WriteFile(candidatePath, []byte(`{"domain":"example.internal","secret":"do-not-expose"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runConfig([]string{"validate", "-profile", profilePath, "-candidate", candidatePath}, &bytes.Buffer{}); err == nil || strings.Contains(err.Error(), "do-not-expose") {
		t.Fatalf("unknown field accepted or leaked: %v", err)
	}
	candidate := loadProfile(t)
	candidate.Domain = "INVALID_DOMAIN"
	writeTestSite(t, candidatePath, candidate)
	if err := runConfig([]string{"plan", "-profile", profilePath, "-candidate", candidatePath}, &bytes.Buffer{}); err == nil {
		t.Fatal("invalid domain accepted")
	}
	if err := os.Remove(candidatePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(profilePath, candidatePath); err != nil {
		t.Fatal(err)
	}
	if err := runConfig([]string{"validate", "-profile", profilePath, "-candidate", candidatePath}, &bytes.Buffer{}); err == nil {
		t.Fatal("candidate symlink accepted")
	}
}

func TestSiteConfigConcurrentApplyUsesOneRevision(t *testing.T) {
	root := t.TempDir()
	profilePath := filepath.Join(root, "site.json")
	candidatePath := filepath.Join(root, "candidate.json")
	original := loadProfile(t)
	writeTestSite(t, profilePath, original)
	candidate := original
	candidate.Domain = "example.internal"
	writeTestSite(t, candidatePath, candidate)
	plan := configOutput(t, "plan", "-profile", profilePath, "-candidate", candidatePath)
	var group sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			results <- runConfig([]string{"apply", "-profile", profilePath, "-candidate", candidatePath, "-confirm", plan["planId"].(string)}, &bytes.Buffer{})
		}()
	}
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected one successful apply, got %d", successes)
	}
}
