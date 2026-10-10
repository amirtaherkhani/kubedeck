// Package platform labels the base tools project without selecting managed projects.
package platform

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/projectenv"
)

const DefaultName = "Kubedesk Platform"

type Profile struct {
	Name      string `json:"name"`
	ProjectID string `json:"projectId,omitempty"`
	Slug      string `json:"slug,omitempty"`
}

func Load(getenv func(string) string) (Profile, error) {
	keys := []string{"KUCHDESK_BASE_PROJECT_NAME", "KUCHDESK_BASE_PROJECT_ID", "KUCHDESK_BASE_PROJECT_SLUG"}
	values := map[string]string{}
	root, err := projectenv.ResolveRoot(getenv)
	if err != nil && !errors.Is(err, projectenv.ErrRootNotFound) {
		return Profile{}, err
	}
	if err == nil {
		values, err = projectenv.Read(root, keys...)
		if err != nil {
			return Profile{}, err
		}
	}
	// Non-empty process values override literal file values; defaults apply last.
	value := func(key string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return values[key]
	}
	name := value("KUCHDESK_BASE_PROJECT_NAME")
	if name == "" {
		name = DefaultName
	}
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || name == "" || utf8.RuneCountInString(name) > 128 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return Profile{}, errors.New("invalid KUCHDESK_BASE_PROJECT_NAME")
	}
	p := Profile{Name: name, ProjectID: value("KUCHDESK_BASE_PROJECT_ID"), Slug: value("KUCHDESK_BASE_PROJECT_SLUG")}
	for _, v := range []string{p.ProjectID, p.Slug} {
		if strings.ContainsAny(v, "/\\?# \t\n\r") || len(v) > 128 {
			return Profile{}, errors.New("invalid base project selector")
		}
	}
	return p, nil
}
