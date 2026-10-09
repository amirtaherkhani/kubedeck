//go:build !darwin

package infisical

import "errors"

func newKeychainBackend(string) (CredentialBackend, error) {
	return nil, errors.New("Keychain credential backend is available only on macOS")
}
