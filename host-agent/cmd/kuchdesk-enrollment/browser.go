package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"
)

type browserOpener func(context.Context, string) error
type commandRunner func(context.Context, string, ...string) error

func openBrowser(ctx context.Context, url string) error {
	return openBrowserForOS(ctx, runtime.GOOS, url, func(ctx context.Context, name string, args ...string) error {
		return exec.CommandContext(ctx, name, args...).Run()
	})
}
func openBrowserForOS(ctx context.Context, platform, url string, run commandRunner) error {
	if platform != "darwin" {
		return errors.New("automatic_browser_open_unavailable")
	}
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return run(bounded, "/usr/bin/open", url)
}
func announceBrowserLogin(ctx context.Context, out io.Writer, url, org string, open browserOpener) error {
	if _, e := fmt.Fprintf(out, "Official login URL: %s\nComplete login/MFA and select organization %s. Keep this terminal open.\nEnter credentials only on the official page. Do not paste browser fallback tokens into this terminal or chat.\n", url, org); e != nil {
		return e
	}
	if e := open(ctx, url); e != nil {
		_, e = fmt.Fprintln(out, "Could not open your browser automatically. Open the URL above manually; this login is still waiting.")
		return e
	}
	_, e := fmt.Fprintln(out, "Opened your browser. Waiting for the official callback (up to 10 minutes).")
	return e
}
