package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupWritesPrivateFileWithoutEchoingValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "infisical", "host.env")
	var output bytes.Buffer
	answers := []string{"client-id-123", "sensitive-value"}
	var prompts []string
	ask := func(label string) (string, error) {
		prompts = append(prompts, label)
		answer := answers[len(prompts)-1]
		return answer, nil
	}
	if err := run(path, ask, &output); err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 2 || strings.Contains(output.String(), "sensitive-value") || strings.Contains(output.String(), "client-id-123") {
		t.Fatal("setup did not prompt safely")
	}
	if err := run(path, func(string) (string, error) { t.Fatal("prompted before existing-file check"); return "", nil }, &output); err == nil {
		t.Fatal("overwrote existing file")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("unsafe file mode: %v %v", info, err)
	}
}
