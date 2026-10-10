package platform

import "testing"

func TestProfileNamesDoNotSelectProjects(t *testing.T) {
	for _, name := range []string{"", "Example Platform", "پلتفرم ابزار", "  Base Tools  "} {
		p, e := Load(func(k string) string {
			if k == "KUCHDESK_BASE_PROJECT_NAME" {
				return name
			}
			return ""
		})
		if e != nil || p.ProjectID != "" || p.Slug != "" {
			t.Fatal("display name selected a project")
		}
		if name == "" && p.Name != DefaultName {
			t.Fatal("default mismatch")
		}
	}
	for _, name := range []string{"  ", "bad\nname"} {
		if _, e := Load(func(string) string { return name }); e == nil {
			t.Fatal("invalid name accepted")
		}
	}
}
