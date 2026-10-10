package enrollmentcli

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

// Run executes the existing enrollment CLI, including explicit user-only login.
func Run(args []string, stdout, stderr io.Writer) error {
	return run(args, stdout, stderr)
}
func run(args []string, stdout, stderr io.Writer) error {
	return runWithBrowser(args, stdout, stderr, openBrowser)
}

func runWithBrowser(args []string, stdout, stderr io.Writer, opener browserOpener) error {
	flags := flag.NewFlagSet("kuchdesk-enrollment", flag.ContinueOnError)
	flags.SetOutput(stderr)
	easy := flags.Bool("login-easy", false, "open official login with automatic callback and hidden paste recovery")
	root := flags.String("project-root", "", "absolute project root for easy login")
	origin := flags.String("url", "", "Infisical HTTPS origin")
	policy := flags.String("policy", "", "private JSON policy file")
	dir := flags.String("state-dir", "", "existing private state directory")
	once := flags.Bool("once", false, "run one bounded cycle")
	manual := flags.Bool("login-token", false, "user-only hidden terminal input of the official browser fallback; never reads clipboard")
	login := flags.Bool("login", false, "user-run official browser login; save a validated private human session")
	status := flags.Bool("status", false, "read-only server validation of saved human enrollment authority; never refresh")
	source := flags.String("import-session", "", "explicit private session handoff file; import only")
	if e := flags.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return errors.New("invalid_arguments")
	}
	if *easy {
		settings, e := loadProjectLogin(*root)
		if e != nil {
			return e
		}
		*origin, *policy, *dir = settings.origin, settings.policy, settings.state
	}
	if flags.NArg() != 0 || *policy == "" || *dir == "" || *origin == "" {
		return errors.New("url_policy_and_state_required")
	}
	modes := 0
	for _, enabled := range []bool{*easy, *login, *manual, *status, *once, *source != ""} {
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
	if *easy || *login || *manual || *status {
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
		if *manual {
			encoded, e := readHiddenBrowserToken(ctx, os.Stdin, stderr)
			if e != nil {
				return e
			}
			report, e := enrollment.LoginFromBrowserToken(ctx, h, p, store, encoded)
			if err := json.NewEncoder(stdout).Encode(report); err != nil {
				return errors.New("status_output_failed")
			}
			return e
		}
		if *easy {
			if _, e := terminalState(int(os.Stdin.Fd())); e != nil {
				return errors.New("interactive_terminal_required")
			}
			report, e := enrollment.BrowserLoginWithFallback(ctx, h, p, store, func(u string) error {
				return announceEasyLogin(ctx, stderr, u, p.OrganizationID, opener)
			}, func(ctx context.Context) (string, error) { return readHiddenBrowserToken(ctx, os.Stdin, stderr) }, func(stage string) { fmt.Fprintln(stderr, stage) })
			if err := json.NewEncoder(stdout).Encode(report); err != nil {
				return errors.New("status_output_failed")
			}
			return e
		}
		report, e := enrollment.BrowserLogin(ctx, h, p, store, func(loginURL string) error {
			return announceBrowserLogin(ctx, stderr, loginURL, p.OrganizationID, opener)
		}, func(stage string) { _, _ = fmt.Fprintln(stderr, stage) })
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
