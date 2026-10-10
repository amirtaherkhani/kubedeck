package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Admin login is an explicit interactive operation, never part of DNS startup.
func runAdminLogin(args []string, stdout, stderr io.Writer, launch func([]string, io.Writer, io.Writer) error) error {
	flags := flag.NewFlagSet("kuchdesk-host-agent admin-login", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("project-root", "", "absolute project root; defaults to KUCHDESK_PROJECT_ROOT")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: kuchdesk-host-agent admin-login [--project-root /absolute/project]\nUser-only official Infisical browser login with hidden paste recovery. Never runs during ordinary DNS startup.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return errors.New("invalid_admin_login_arguments")
	}
	if flags.NArg() != 0 {
		return errors.New("invalid_admin_login_arguments")
	}
	if *root == "" {
		*root = os.Getenv("KUCHDESK_PROJECT_ROOT")
	}
	if !filepath.IsAbs(*root) {
		return errors.New("absolute_project_root_required: use --project-root or KUCHDESK_PROJECT_ROOT")
	}
	return launch([]string{"-login-easy", "-project-root", *root}, stdout, stderr)
}
