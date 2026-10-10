package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/platform"
)

func run(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	return runWithInput(ctx, args, os.Stdin, stdout, stderr, getenv)
}

func runWithInput(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("kuchdesk-infisical", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseURL := flags.String("url", "", "Infisical HTTPS origin")
	timeout := flags.Duration("timeout", 10*time.Second, "request timeout")
	if err := flags.Parse(args); err != nil || *baseURL == "" || *timeout <= 0 {
		fmt.Fprintln(stderr, "usage: kuchdesk-infisical -url HTTPS_ORIGIN [-timeout 10s] capabilities|check-project-access|list-secret-names|manage")
		return 2
	}
	profile, profileErr := platform.Load(getenv)
	if profileErr != nil {
		fmt.Fprintln(stderr, profileErr)
		return 2
	}
	clientID, clientSecret, err := infisical.HostCredentials(getenv)
	if err != nil {
		fmt.Fprintln(stderr, "Infisical credential backend unavailable:", err)
		return 2
	}
	client, err := infisical.NewClient(*baseURL, clientID, clientSecret, nil)
	if err != nil {
		fmt.Fprintln(stderr, "invalid Infisical configuration:", err)
		return 2
	}
	service := infisical.NewService(client)
	remaining := flags.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(stderr, "provide capabilities, check-project-access, list-secret-names, or manage")
		return 2
	}
	encoder := json.NewEncoder(stdout)
	switch remaining[0] {
	case "capabilities":
		if len(remaining) != 1 {
			fmt.Fprintln(stderr, "capabilities accepts no arguments")
			return 2
		}
		if err := encoder.Encode(map[string]any{"baseProject": profile, "configured": service.Configured(), "commands": []string{"capabilities", "check-project-access", "list-secret-names", "manage"}}); err != nil {
			fmt.Fprintln(stderr, "write output failed")
			return 1
		}
		return 0
	case "manage":
		commandFlags := flag.NewFlagSet("manage", flag.ContinueOnError)
		commandFlags.SetOutput(stderr)
		projectIDs := commandFlags.String("project-ids", "", "comma-separated project ID allowlist")
		allowProjectCreate := commandFlags.Bool("allow-project-create", false, "allow organization-level project creation")
		if err := commandFlags.Parse(remaining[1:]); err != nil || commandFlags.NArg() != 0 {
			fmt.Fprintln(stderr, "usage: manage -project-ids ID[,ID] [-allow-project-create] < command.json")
			return 2
		}
		allowed := make(map[string]bool)
		for _, raw := range strings.Split(*projectIDs, ",") {
			if id := strings.TrimSpace(raw); id != "" {
				allowed[id] = true
			}
		}
		if len(allowed) == 0 && !*allowProjectCreate {
			fmt.Fprintln(stderr, "project allowlist required")
			return 2
		}
		decoder := json.NewDecoder(io.LimitReader(stdin, 64<<10+1))
		decoder.DisallowUnknownFields()
		var input infisical.Command
		if err := decoder.Decode(&input); err != nil {
			fmt.Fprintln(stderr, "invalid command input")
			return 2
		}
		if err := decoder.Decode(&struct{}{}); err != io.EOF {
			fmt.Fprintln(stderr, "command input must contain one JSON object")
			return 2
		}
		requestCtx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()
		result, err := (infisical.CommandService{Client: client, AllowedProjects: allowed, AllowProjectCreate: *allowProjectCreate}).Execute(requestCtx, input)
		if err != nil {
			fmt.Fprintln(stderr, infisical.PublicError(err))
			return 1
		}
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintln(stderr, "write output failed")
			return 1
		}
		return 0
	case "check-project-access":
		command := flag.NewFlagSet("check-project-access", flag.ContinueOnError)
		command.SetOutput(stderr)
		slug := command.String("slug", "", "Infisical project slug")
		if err := command.Parse(remaining[1:]); err != nil || command.NArg() != 0 || *slug == "" {
			fmt.Fprintln(stderr, "usage: check-project-access -slug PROJECT_SLUG")
			return 2
		}
		requestCtx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()
		access, err := client.InspectProjectAccess(requestCtx, *slug)
		if err != nil {
			fmt.Fprintln(stderr, infisical.PublicError(err))
			if err == infisical.ErrInvalidProjectSlug {
				return 2
			}
			return 1
		}
		if err := encoder.Encode(access); err != nil {
			fmt.Fprintln(stderr, "write output failed")
			return 1
		}
		return 0
	case "list-secret-names":
		command := flag.NewFlagSet("list-secret-names", flag.ContinueOnError)
		command.SetOutput(stderr)
		project := command.String("project", "", "Infisical project ID")
		environment := command.String("environment", "", "Infisical environment slug")
		path := command.String("path", "", "absolute secret path")
		if err := command.Parse(remaining[1:]); err != nil || command.NArg() != 0 {
			fmt.Fprintln(stderr, "usage: list-secret-names -project ID -environment SLUG -path /PATH")
			return 2
		}
		scope := infisical.Scope{ProjectID: *project, Environment: *environment, SecretPath: *path}
		if err := infisical.ValidateScope(scope); err != nil {
			fmt.Fprintln(stderr, infisical.PublicError(err))
			return 2
		}
		requestCtx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()
		names, err := service.ListSecretNames(requestCtx, scope)
		if err != nil {
			fmt.Fprintln(stderr, infisical.PublicError(err))
			return 1
		}
		if err := encoder.Encode(map[string]any{"secrets": names}); err != nil {
			fmt.Fprintln(stderr, "write output failed")
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unknown Infisical command")
		return 2
	}
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, os.Getenv))
}
