package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/deploy"
)

func run(args []string, stdout, stderr io.Writer, runner deploy.Runner) int {
	flags := flag.NewFlagSet("kuchdesk-deploy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	profile := flags.String("profile", "", "absolute path to a deployment profile JSON file")
	apply := flags.Bool("apply", false, "run the planned commands")
	confirm := flags.String("confirm", "", "exact release/namespace confirmation for apply")
	timeout := flags.Duration("timeout", 20*time.Minute, "overall execution timeout")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || !filepath.IsAbs(*profile) || *timeout < time.Minute || *timeout > 2*time.Hour {
		fmt.Fprintln(stderr, "usage: kuchdesk-deploy -profile /absolute/profile.json [-apply -confirm release/namespace] [-timeout 20m]")
		return 2
	}
	spec, err := deploy.LoadSpec(*profile)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	steps, err := spec.Plan()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if !*apply {
		if err := json.NewEncoder(stdout).Encode(map[string]any{"release": spec.Release, "namespace": spec.Namespace, "kubeContext": spec.KubeContext, "sourceDir": spec.SourceDir, "chartDir": spec.ChartDir, "valuesFiles": spec.ValuesFiles, "deployment": spec.Deployment, "image": spec.ImageRepository + ":" + spec.ImageTag, "pushMode": spec.PushMode, "hostPushRepository": spec.HostPushRepo, "kindPrepull": spec.KindPrepull, "steps": steps}); err != nil {
			fmt.Fprintln(stderr, "write plan failed")
			return 1
		}
		return 0
	}
	if *confirm != spec.Release+"/"+spec.Namespace {
		fmt.Fprintln(stderr, "exact release/namespace confirmation required")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	result, err := (deploy.Workflow{Runner: runner, OnStep: func(step string) { fmt.Fprintln(stderr, step) }}).Apply(ctx, spec)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "write result failed")
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, deploy.ExecRunner{}))
}
