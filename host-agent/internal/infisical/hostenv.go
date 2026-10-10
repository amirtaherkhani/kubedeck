package infisical

import (
	"context"
	"errors"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/projectenv"
)

// CredentialBackend keeps storage selection separate from Infisical's client.
// Neither backend writes credentials or falls back to another backend.
type CredentialBackend interface {
	Load(context.Context) (Credentials, error)
}

type Credentials struct {
	ClientID     string
	ClientSecret string
	BridgeToken  string
}

// HostBridgeToken reads the separate bridge bearer from the same private host
// file. It is not an Infisical credential and is never copied into Helm values.
func HostBridgeToken(getenv func(string) string) (string, error) {
	root, err := projectenv.ResolveRoot(getenv)
	if err != nil {
		return "", err
	}
	credentials, err := (envBackend{projectRoot: root}).Load(context.Background())
	if err != nil {
		return "", err
	}
	if len(credentials.BridgeToken) < 32 || len(credentials.BridgeToken) > 256 {
		return "", errors.New("private .env requires a 32-256 character KUCHDESK_HOST_BRIDGE_TOKEN")
	}
	return credentials.BridgeToken, nil
}

type envBackend struct{ projectRoot string }

// HostCredentials selects one backend. The project-local .env is the default;
// macOS Keychain is used only when explicitly selected.
func HostCredentials(getenv func(string) string) (string, string, error) {
	backend, err := SelectCredentialBackend(getenv)
	if err != nil {
		return "", "", err
	}
	credentials, err := backend.Load(context.Background())
	if err != nil {
		return "", "", err
	}
	return credentials.ClientID, credentials.ClientSecret, nil
}

func SelectCredentialBackend(getenv func(string) string) (CredentialBackend, error) {
	switch getenv("KUCHDESK_INFISICAL_CREDENTIAL_BACKEND") {
	case "", "env":
		root, err := projectenv.ResolveRoot(getenv)
		if err != nil {
			return nil, err
		}
		return envBackend{projectRoot: root}, nil
	case "keychain":
		return newKeychainBackend(getenv("KUCHDESK_INFISICAL_KEYCHAIN_ACCOUNT"))
	default:
		return nil, errors.New("unsupported Infisical credential backend")
	}
}

func (backend envBackend) Load(context.Context) (Credentials, error) {
	values, err := projectenv.Read(backend.projectRoot, "INFISICAL_CLIENT_ID", "INFISICAL_CLIENT_SECRET", "KUCHDESK_HOST_BRIDGE_TOKEN")
	if err != nil {
		return Credentials{}, err
	}
	result := Credentials{ClientID: values["INFISICAL_CLIENT_ID"], ClientSecret: values["INFISICAL_CLIENT_SECRET"], BridgeToken: values["KUCHDESK_HOST_BRIDGE_TOKEN"]}
	if (result.ClientID == "") != (result.ClientSecret == "") {
		return Credentials{}, errors.New("project .env requires both Infisical credential keys")
	}
	return result, nil
}
