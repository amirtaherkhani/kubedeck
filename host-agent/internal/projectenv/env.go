// Package projectenv reads explicitly selected literal keys from the private project .env.
// It never executes shell syntax, changes process environment, or writes files.
package projectenv

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const maxBytes = 4096

var ErrRootNotFound = errors.New("KuchDesk project root not found; set KUCHDESK_PROJECT_ROOT")

func ResolveRoot(getenv func(string) string) (string, error) {
	root := getenv("KUCHDESK_PROJECT_ROOT")
	if root == "" {
		return findProjectRoot()
	}
	if !filepath.IsAbs(root) {
		return "", errors.New("KUCHDESK_PROJECT_ROOT must be absolute")
	}
	return root, nil
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
			return "", ErrRootNotFound
		}
		dir = parent
	}
}

func Read(root string, keys ...string) (map[string]string, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, errors.New("KuchDesk project root unavailable")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o022 != 0 || !ownedByCurrentUser(info) {
		return nil, errors.New("KuchDesk project root must be user-owned and not group/world writable")
	}
	selected := map[string]bool{}
	for _, key := range keys {
		selected[key] = true
	}
	return readFile(filepath.Join(resolved, ".env"), selected)
}

func readFile(path string, selected map[string]bool) (map[string]string, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("project .env unavailable")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || !ownedByCurrentUser(info) || info.Size() > maxBytes {
		return nil, errors.New("project .env must be user-owned regular mode 0600 and at most 4 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil || len(data) > maxBytes {
		return nil, errors.New("project .env unavailable")
	}
	result := map[string]string{}
	seen := make(map[string]bool, 3)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || strings.ContainsAny(key, " \t\r\x00") || strings.ContainsAny(value, "\r\x00") {
			return nil, errors.New("project .env contains invalid entries")
		}
		if !selected[key] {
			continue
		}
		if seen[key] || strings.TrimSpace(value) != value {
			return nil, errors.New("project .env contains invalid or duplicate selected keys")
		}
		seen[key] = true
		result[key] = value
	}
	return result, nil
}

func ownedByCurrentUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}
