//go:build darwin

package infisical

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

const keychainService = "kuchdesk.infisical.machine-identity"

type keychainBackend struct {
	account string
	lookup  func(context.Context, string, string) ([]byte, error)
}

func newKeychainBackend(account string) (CredentialBackend, error) {
	if account == "" || strings.TrimSpace(account) != account || strings.ContainsAny(account, "\r\n\x00") {
		return nil, errors.New("KUCHDESK_INFISICAL_KEYCHAIN_ACCOUNT is required for Keychain backend")
	}
	return keychainBackend{account: account, lookup: lookupKeychain}, nil
}

func (backend keychainBackend) Load(parent context.Context) (Credentials, error) {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	output, err := backend.lookup(ctx, keychainService, backend.account)
	if err != nil {
		return Credentials{}, errors.New("selected Keychain credential unavailable")
	}
	secret := strings.TrimSuffix(string(output), "\n")
	if secret == "" || strings.ContainsAny(secret, "\r\n\x00") {
		return Credentials{}, errors.New("selected Keychain credential invalid")
	}
	return Credentials{ClientID: backend.account, ClientSecret: secret}, nil
}

func lookupKeychain(ctx context.Context, service, account string) ([]byte, error) {
	// security writes only the password to stdout. Never include its output or
	// stderr in a public error or log.
	return exec.CommandContext(ctx, "/usr/bin/security", "find-generic-password", "-s", service, "-a", account, "-w").Output()
}
