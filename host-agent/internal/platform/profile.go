// Package platform labels the base tools project without selecting managed projects.
package platform

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const DefaultName = "Kubedesk Platform"

type Profile struct {
	Name      string `json:"name"`
	ProjectID string `json:"projectId,omitempty"`
	Slug      string `json:"slug,omitempty"`
}

func Load(getenv func(string) string) (Profile, error) {
	name := getenv("KUCHDESK_BASE_PROJECT_NAME")
	if name == "" {
		name = DefaultName
	}
	name = strings.TrimSpace(name)
	if !utf8.ValidString(name) || name == "" || utf8.RuneCountInString(name) > 128 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return Profile{}, errors.New("invalid KUCHDESK_BASE_PROJECT_NAME")
	}
	p := Profile{Name: name, ProjectID: getenv("KUCHDESK_BASE_PROJECT_ID"), Slug: getenv("KUCHDESK_BASE_PROJECT_SLUG")}
	for _, v := range []string{p.ProjectID, p.Slug} {
		if strings.ContainsAny(v, "/\\?# \t\n\r") || len(v) > 128 {
			return Profile{}, errors.New("invalid base project selector")
		}
	}
	return p, nil
}
