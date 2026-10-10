package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBrowserAnnouncementPrintsURLBeforeOpening(t *testing.T) {
	var out bytes.Buffer
	url := "https://infisical.local.dev/login?callback_port=65081"
	calls := 0
	err := announceBrowserLogin(context.Background(), &out, url, "org", func(ctx context.Context, got string) error {
		calls++
		if got != url || !strings.Contains(out.String(), url) {
			t.Fatal("URL unavailable before launch")
		}
		return nil
	})
	if err != nil || calls != 1 || !strings.Contains(out.String(), "Opened your browser") {
		t.Fatalf("announcement: %v %s", err, out.String())
	}
}
func TestBrowserOpenFailureKeepsManualCallbackAvailable(t *testing.T) {
	var out bytes.Buffer
	err := announceBrowserLogin(context.Background(), &out, "https://infisical.local.dev/login?callback_port=65081", "org", func(context.Context, string) error { return errors.New("private internal error") })
	if err != nil || !strings.Contains(out.String(), "Could not open") || !strings.Contains(out.String(), "still waiting") || strings.Contains(out.String(), "private internal error") {
		t.Fatalf("unsafe fallback: %v %s", err, out.String())
	}
}
func TestMacBrowserOpenerUsesBoundedArgumentVector(t *testing.T) {
	url := "https://infisical.local.dev/login?callback_port=65081&literal=$(must-not-run)"
	calls := 0
	err := openBrowserForOS(context.Background(), "darwin", url, func(ctx context.Context, name string, args ...string) error {
		calls++
		if name != "/usr/bin/open" || len(args) != 1 || args[0] != url {
			t.Fatal("unsafe browser command")
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Fatal("unbounded opener")
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatal(err)
	}
	err = openBrowserForOS(context.Background(), "linux", url, func(context.Context, string, ...string) error { t.Fatal("unsupported platform command"); return nil })
	if err == nil {
		t.Fatal("missing fallback signal")
	}
}
