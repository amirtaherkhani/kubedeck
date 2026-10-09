package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
)

func run(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	flags := flag.NewFlagSet("kuchdesk-infisical", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseURL := flags.String("url", "", "Infisical HTTPS origin")
	timeout := flags.Duration("timeout", 10*time.Second, "request timeout")
	if err := flags.Parse(args); err != nil || *baseURL == "" || *timeout <= 0 {
		fmt.Fprintln(stderr, "usage: kuchdesk-infisical -url HTTPS_ORIGIN [-timeout 10s] capabilities|list-secret-names")
		return 2
	}
	client, err := infisical.NewClient(*baseURL, getenv("INFISICAL_CLIENT_ID"), getenv("INFISICAL_CLIENT_SECRET"), nil)
	if err != nil {
		fmt.Fprintln(stderr, "invalid Infisical configuration:", err)
		return 2
	}
	service := infisical.NewService(client)
	remaining := flags.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(stderr, "provide capabilities or list-secret-names")
		return 2
	}
	encoder := json.NewEncoder(stdout)
	switch remaining[0] {
	case "capabilities":
		if len(remaining) != 1 {
			fmt.Fprintln(stderr, "capabilities accepts no arguments")
			return 2
		}
		if err := encoder.Encode(map[string]any{"configured": service.Configured(), "commands": []string{"capabilities", "list-secret-names"}}); err != nil {
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
