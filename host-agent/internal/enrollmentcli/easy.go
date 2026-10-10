package enrollmentcli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

type loginSettings struct{ origin, policy, state string }

func loadProjectLogin(root string) (loginSettings, error) {
	if root == "" {
		root = os.Getenv("KUCHDESK_PROJECT_ROOT")
	}
	if !filepath.IsAbs(root) {
		return loginSettings{}, errors.New("absolute_project_root_required")
	}
	root, e := filepath.EvalSymlinks(root)
	if e != nil {
		return loginSettings{}, errors.New("project_unavailable")
	}
	if _, e = os.Stat(filepath.Join(root, "host-agent", "go.mod")); e != nil {
		return loginSettings{}, errors.New("project_unavailable")
	}
	f, e := os.Open(filepath.Join(root, "lab", "site.json"))
	if e != nil {
		return loginSettings{}, errors.New("site_configuration_missing")
	}
	defer f.Close()
	var site struct {
		Domain string `json:"domain"`
		Hosts  struct {
			Infisical string `json:"infisical"`
		} `json:"hosts"`
	}
	d := json.NewDecoder(io.LimitReader(f, 64<<10))
	if d.Decode(&site) != nil || d.Decode(new(any)) != io.EOF {
		return loginSettings{}, errors.New("invalid_site_configuration")
	}
	label := regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	domain := regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
	if !label.MatchString(site.Hosts.Infisical) || !domain.MatchString(site.Domain) {
		return loginSettings{}, errors.New("invalid_site_configuration")
	}
	state := filepath.Join(root, ".kuchdesk", "infisical-control")
	return loginSettings{"https://" + site.Hosts.Infisical + "." + site.Domain, filepath.Join(state, "policy.json"), state}, nil
}
func announceEasyLogin(ctx context.Context, out io.Writer, url, org string, open browserOpener) error {
	if _, e := fmt.Fprintf(out, "Official login URL: %s\nSelect organization %s and complete login/MFA.\nIf Copy to clipboard appears, copy immediately (the page copy lasts 30 seconds), return here, paste into the hidden prompt and press Enter. Never paste into chat.\n", url, org); e != nil {
		return e
	}
	if e := open(ctx, url); e != nil {
		_, e = fmt.Fprintln(out, "Browser opening failed. Open the exact URL above manually; this attempt remains active.")
		return e
	}
	return nil
}
