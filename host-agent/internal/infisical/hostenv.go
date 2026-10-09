package infisical

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const maxHostEnvBytes = 4096

// CredentialBackend keeps storage selection separate from Infisical's client.
// Neither backend writes credentials or falls back to another backend.
type CredentialBackend interface {
	Load(context.Context) (Credentials, error)
}

type Credentials struct {
	ClientID     string
	ClientSecret string
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
		root := getenv("KUCHDESK_PROJECT_ROOT")
		if root == "" {
			var err error
			root, err = findProjectRoot()
			if err != nil {
				return nil, err
			}
		}
		if !filepath.IsAbs(root) {
			return nil, errors.New("KUCHDESK_PROJECT_ROOT must be absolute")
		}
		return envBackend{projectRoot: root}, nil
	case "keychain":
		return newKeychainBackend(getenv("KUCHDESK_INFISICAL_KEYCHAIN_ACCOUNT"))
	default:
		return nil, errors.New("unsupported Infisical credential backend")
	}
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.New("project working directory unavailable")
	}
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "host-agent", "go.mod")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("KuchDesk project root not found; set KUCHDESK_PROJECT_ROOT")
		}
		dir = parent
	}
}

func (backend envBackend) Load(context.Context) (Credentials, error) {
	root, err := filepath.EvalSymlinks(backend.projectRoot)
	if err != nil {
		return Credentials{}, errors.New("KuchDesk project root unavailable")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o022 != 0 || !ownedByCurrentUser(info) {
		return Credentials{}, errors.New("KuchDesk project root must be user-owned and not group/world writable")
	}
	return readProjectEnv(filepath.Join(root, ".env"))
}

func readProjectEnv(path string) (Credentials, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return Credentials{}, nil
	}
	if err != nil {
		return Credentials{}, errors.New("project .env unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) || info.Size() > maxHostEnvBytes {
		return Credentials{}, errors.New("project .env must be user-owned regular mode 0600 and at most 4 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxHostEnvBytes+1))
	if err != nil || len(data) > maxHostEnvBytes {
		return Credentials{}, errors.New("project .env unavailable")
	}
	var result Credentials
	seen := make(map[string]bool, 2)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || strings.ContainsAny(key, " \t\r\x00") || strings.ContainsAny(value, "\r\x00") {
			return Credentials{}, errors.New("project .env contains invalid entries")
		}
		if key != "INFISICAL_CLIENT_ID" && key != "INFISICAL_CLIENT_SECRET" {
			continue
		}
		if seen[key] || strings.TrimSpace(value) != value {
			return Credentials{}, errors.New("project .env contains invalid or duplicate Infisical keys")
		}
		seen[key] = true
		if key == "INFISICAL_CLIENT_ID" {
			result.ClientID = value
		} else {
			result.ClientSecret = value
		}
	}
	if (result.ClientID == "") != (result.ClientSecret == "") {
		return Credentials{}, errors.New("project .env requires both Infisical credential keys")
	}
	return result, nil
}

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}
