package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/enrollment"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(os.Args[1:], os.Stdout, os.Stderr); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("kuchdesk-enrollment", flag.ContinueOnError)
	flags.SetOutput(stderr)
	origin := flags.String("url", "", "Infisical HTTPS origin")
	policy := flags.String("policy", "", "private JSON policy file")
	dir := flags.String("state-dir", "", "existing private state directory")
	once := flags.Bool("once", false, "run one bounded cycle")
	login := flags.Bool("login", false, "user-run official browser login; save a validated private human session")
	status := flags.Bool("status", false, "read-only server validation of saved human enrollment authority; never refresh")
	source := flags.String("import-session", "", "explicit private session handoff file; import only")
	if e := flags.Parse(args); e != nil {
		return errors.New("invalid_arguments")
	}
	if flags.NArg() != 0 || *policy == "" || *dir == "" || *origin == "" {
		return errors.New("url_policy_and_state_required")
	}
	modes := 0
	for _, enabled := range []bool{*login, *status, *once, *source != ""} {
		if enabled {
			modes++
		}
	}
	if modes > 1 {
		return errors.New("choose_one_enrollment_mode")
	}
	h, e := enrollment.NewTransport(*origin, nil)
	if e != nil {
		return e
	}
	if *login || *status {
		p, e := enrollment.LoadPolicy(*policy)
		if e != nil {
			return e
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if *status {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			report, e := enrollment.HumanStatusAt(ctx, h, p, *dir)
			if err := json.NewEncoder(stdout).Encode(report); err != nil {
				return errors.New("status_output_failed")
			}
			if e != nil {
				return e
			}
			if !report.AccessAllProjects {
				return enrollment.ErrEnrollment
			}
			return nil
		}
		store, e := enrollment.OpenStore(*dir)
		if e != nil {
			return e
		}
		defer store.Close()
		report, e := enrollment.BrowserLogin(ctx, h, p, store, func(loginURL string) error {
			_, err := fmt.Fprintf(stderr, "Open this official login URL in your browser:\n%s\nComplete login and MFA, then select organization %s. Keep this terminal open.\nEnter credentials only in the official page; tokens are saved privately on this Mac.\n", loginURL, p.OrganizationID)
			return err
		})
		if err := json.NewEncoder(stdout).Encode(report); err != nil {
			return errors.New("status_output_failed")
		}
		return e
	}
	if *source != "" {
		p, e := enrollment.LoadPolicy(*policy)
		if e != nil {
			return e
		}
		s, e := enrollment.OpenStore(*dir)
		if e != nil {
			return e
		}
		defer s.Close()
		if e = enrollment.ImportSession(*source, *origin, p, s); e != nil {
			return e
		}
		fmt.Fprintln(stdout, "session_imported")
		return nil
	}
	id, secret, e := infisical.HostCredentials(os.Getenv)
	if e != nil {
		return errors.New("machine_bootstrap_unavailable")
	}
	machine, e := infisical.NewClient(*origin, id, secret, nil)
	if e != nil {
		return e
	}
	c, s, e := enrollment.OpenRuntime(*origin, *policy, *dir, machine)
	if e != nil {
		return e
	}
	defer s.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *once {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		enrollment.EmitReport(c.Cycle(ctx))
		return nil
	}
	c.Run(ctx, enrollment.EmitReport)
	return nil
}
