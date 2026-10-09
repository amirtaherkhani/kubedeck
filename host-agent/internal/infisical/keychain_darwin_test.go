//go:build darwin

package infisical

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExplicitKeychainBackendUsesOnlySelectedItem(t *testing.T) {
	backend, err := newKeychainBackend("client-id")
	if err != nil {
		t.Fatal(err)
	}
	selected := backend.(keychainBackend)
	selected.lookup = func(_ context.Context, service, account string) ([]byte, error) {
		if service != keychainService || account != "client-id" {
			t.Fatalf("wrong Keychain selector: %q %q", service, account)
		}
		return []byte("test-secret\n"), nil
	}
	credentials, err := selected.Load(context.Background())
	if err != nil || credentials.ClientID != "client-id" || credentials.ClientSecret != "test-secret" {
		t.Fatalf("selected item not loaded: %v", err)
	}
	selected.lookup = func(context.Context, string, string) ([]byte, error) { return nil, errors.New("test-secret") }
	if _, err := selected.Load(context.Background()); err == nil || strings.Contains(err.Error(), "test-secret") {
		t.Fatal("Keychain error leaked or fell back")
	}
}
