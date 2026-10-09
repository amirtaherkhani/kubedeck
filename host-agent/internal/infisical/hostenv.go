package infisical

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const maxHostEnvBytes = 4096

func HostEnvPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || !filepath.IsAbs(home) {
		return "", errors.New("user home directory unavailable")
	}
	return filepath.Join(home, ".config", "kuchdesk", "infisical", "host.env"), nil
}

// HostCredentials prefers a complete process environment pair. Otherwise it
// reads the user's private host.env file. No shell evaluation is performed.
func HostCredentials(getenv func(string) string) (string, string, error) {
	id, secret := getenv("INFISICAL_CLIENT_ID"), getenv("INFISICAL_CLIENT_SECRET")
	if (id == "") != (secret == "") {
		return "", "", errors.New("both Infisical host credential variables are required")
	}
	if id != "" {
		return id, secret, nil
	}
	path := getenv("KUCHDESK_INFISICAL_ENV_FILE")
	explicit := path != ""
	if !explicit {
		var err error
		path, err = HostEnvPath()
		if err != nil {
			return "", "", err
		}
	}
	return readHostEnv(path, explicit)
}

func readHostEnv(path string, explicit bool) (string, string, error) {
	if !filepath.IsAbs(path) {
		return "", "", errors.New("Infisical env file path must be absolute")
	}
	parent := filepath.Dir(path)
	parentInfo, err := os.Lstat(parent)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return "", "", nil
	}
	if err != nil || !parentInfo.IsDir() || parentInfo.Mode().Perm() != 0o700 || !ownedByCurrentUser(parentInfo) {
		return "", "", errors.New("Infisical env directory must be user-owned mode 0700")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) && !explicit {
		return "", "", nil
	}
	if err != nil {
		return "", "", errors.New("Infisical env file unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) || info.Size() > maxHostEnvBytes {
		return "", "", errors.New("Infisical env file must be user-owned regular mode 0600 and at most 4 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxHostEnvBytes+1))
	if err != nil || len(data) > maxHostEnvBytes {
		return "", "", errors.New("Infisical env file unavailable")
	}
	values := make(map[string]string, 2)
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || (key != "INFISICAL_CLIENT_ID" && key != "INFISICAL_CLIENT_SECRET") || value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\x00") || values[key] != "" {
			return "", "", errors.New("Infisical env file contains invalid or duplicate keys")
		}
		values[key] = value
	}
	if len(values) != 2 {
		return "", "", errors.New("Infisical env file requires exactly two credentials")
	}
	return values["INFISICAL_CLIENT_ID"], values["INFISICAL_CLIENT_SECRET"], nil
}

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}

// SaveHostCredentials creates the initial private file atomically and refuses
// to replace an existing identity. Values are supplied by a local no-echo TTY.
func SaveHostCredentials(path, id, secret string) error {
	if !filepath.IsAbs(path) || id == "" || secret == "" || strings.ContainsAny(id, "\r\n\x00") || strings.ContainsAny(secret, "\r\n\x00") || strings.TrimSpace(id) != id || strings.TrimSpace(secret) != secret || len(id)+len(secret) > maxHostEnvBytes-64 {
		return errors.New("invalid Infisical host credentials or path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return errors.New("cannot create Infisical env directory")
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || !ownedByCurrentUser(info) {
		return errors.New("Infisical env directory must be user-owned mode 0700")
	}
	if _, err := os.Lstat(path); err == nil {
		return errors.New("Infisical env file already exists; refusing to overwrite")
	} else if !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot inspect Infisical env file")
	}
	file, err := os.CreateTemp(dir, ".host-env-*")
	if err != nil {
		return errors.New("cannot create Infisical env file")
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(0o600); err != nil {
		return errors.New("cannot protect Infisical env file")
	}
	if _, err := fmt.Fprintf(file, "INFISICAL_CLIENT_ID=%s\nINFISICAL_CLIENT_SECRET=%s\n", id, secret); err != nil || file.Sync() != nil {
		return errors.New("cannot write Infisical env file")
	}
	// Link is atomic and cannot replace an existing file, unlike Rename.
	if err := os.Link(file.Name(), path); err != nil {
		return errors.New("cannot install Infisical env file without replacing existing data")
	}
	return nil
}
