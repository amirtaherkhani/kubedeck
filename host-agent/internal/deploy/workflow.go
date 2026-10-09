package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	namePattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,62}$`)
	contextPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@/-]{0,127}$`)
	imagePattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9./:_-]+$`)
	tagPattern     = regexp.MustCompile(`^git-[0-9a-f]{12}$`)
	digestPattern  = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// Spec is one explicit, local deployment target. No command or shell string is
// accepted from the profile; the workflow owns the Docker, Helm, and kubectl
// argument lists.
type Spec struct {
	SourceDir       string   `json:"sourceDir"`
	Dockerfile      string   `json:"dockerfile"`
	ChartDir        string   `json:"chartDir"`
	ValuesFiles     []string `json:"valuesFiles,omitempty"`
	ImageRepository string   `json:"imageRepository"`
	ImageTag        string   `json:"imageTag"`
	KubeContext     string   `json:"kubeContext"`
	Namespace       string   `json:"namespace"`
	Release         string   `json:"release"`
	Deployment      string   `json:"deployment"`
	Test            string   `json:"test"`
}

type Result struct {
	Image  string `json:"image"`
	Digest string `json:"digest"`
}

type Runner interface {
	Run(context.Context, string, string, ...string) error
	Output(context.Context, string, string, ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = filteredEnvironment()
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}

func (ExecRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Env = filteredEnvironment()
	return command.Output()
}

func filteredEnvironment() []string {
	var result []string
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "INFISICAL_CLIENT_ID=") || strings.HasPrefix(item, "INFISICAL_CLIENT_SECRET=") {
			continue
		}
		result = append(result, item)
	}
	return result
}

type Workflow struct {
	Runner Runner
	OnStep func(string)
}

func (s Spec) Validate() error {
	if !namePattern.MatchString(s.Namespace) || !namePattern.MatchString(s.Release) || !namePattern.MatchString(s.Deployment) || !contextPattern.MatchString(s.KubeContext) {
		return errors.New("namespace, release, deployment, and Kubernetes context are required")
	}
	if !imagePattern.MatchString(s.ImageRepository) || !tagPattern.MatchString(s.ImageTag) || strings.HasSuffix(s.ImageRepository, ":") {
		return errors.New("image repository and immutable git tag are required")
	}
	if s.Test != "" && s.Test != "go" {
		return errors.New("only test=go or no test is supported")
	}
	for _, path := range []string{s.SourceDir, s.Dockerfile, s.ChartDir} {
		if !filepath.IsAbs(path) {
			return errors.New("sourceDir, dockerfile, and chartDir must be absolute paths")
		}
	}
	resolvedSource, sourceErr := filepath.EvalSymlinks(s.SourceDir)
	resolvedDockerfile, dockerfileErr := filepath.EvalSymlinks(s.Dockerfile)
	resolvedChart, chartErr := filepath.EvalSymlinks(s.ChartDir)
	if sourceErr != nil || dockerfileErr != nil || chartErr != nil {
		return errors.New("source, Dockerfile, or chart path cannot be resolved")
	}
	if relative, err := filepath.Rel(resolvedSource, resolvedDockerfile); err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("Dockerfile must be inside the source directory")
	}
	if relative, err := filepath.Rel(resolvedSource, resolvedChart); err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("chart must be inside the source directory")
	}
	for _, path := range []string{s.SourceDir, s.ChartDir} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return errors.New("source or chart directory does not exist")
		}
	}
	for _, path := range append([]string{s.Dockerfile, filepath.Join(s.ChartDir, "Chart.yaml")}, s.ValuesFiles...) {
		if !filepath.IsAbs(path) {
			return errors.New("Dockerfile and values files must use absolute paths")
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return errors.New("Dockerfile, chart, or values file does not exist")
		}
	}
	return nil
}

func (s Spec) Plan() ([]string, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	steps := []string{"validate profile and source revision", "helm lint"}
	if s.Test == "go" {
		steps = append(steps, "go test")
	}
	return append(steps, "docker build", "docker push", "resolve image digest", "helm template", "helm upgrade --install --atomic --wait", "kubectl rollout status"), nil
}

func (w Workflow) Apply(ctx context.Context, s Spec) (Result, error) {
	if _, err := s.Plan(); err != nil {
		return Result{}, err
	}
	if w.Runner == nil {
		return Result{}, errors.New("command runner is required")
	}
	w.progress("validate profile and source revision")
	// A git tag must identify exactly the clean source being built.
	revision, err := w.Runner.Output(ctx, s.SourceDir, "git", "rev-parse", "--short=12", "HEAD")
	if err != nil || s.ImageTag != "git-"+strings.TrimSpace(string(revision)) {
		return Result{}, errors.New("image tag must match the source HEAD")
	}
	changed, err := w.Runner.Output(ctx, s.SourceDir, "git", "status", "--porcelain", "--untracked-files=all", "--", ".")
	if err != nil || len(strings.TrimSpace(string(changed))) != 0 {
		return Result{}, errors.New("source has uncommitted or untracked files")
	}
	lintValues := append(valuesArgs(s.ValuesFiles), "--set-string", "image.repository="+s.ImageRepository, "--set-string", "image.tag="+s.ImageTag)
	if err := w.run(ctx, s.SourceDir, "helm lint", "helm", append([]string{"lint", s.ChartDir}, lintValues...)...); err != nil {
		return Result{}, err
	}
	if s.Test == "go" {
		if err := w.run(ctx, s.SourceDir, "go test", "go", "test", "./..."); err != nil {
			return Result{}, err
		}
	}
	image := s.ImageRepository + ":" + s.ImageTag
	if err := w.run(ctx, s.SourceDir, "docker build", "docker", "build", "--file", s.Dockerfile, "--tag", image, s.SourceDir); err != nil {
		return Result{}, err
	}
	if err := w.run(ctx, s.SourceDir, "docker push", "docker", "push", image); err != nil {
		return Result{}, err
	}
	w.progress("resolve image digest")
	raw, err := w.Runner.Output(ctx, s.SourceDir, "docker", "image", "inspect", "--format", "{{json .RepoDigests}}", image)
	if err != nil {
		return Result{}, errors.New("image digest lookup failed")
	}
	var repoDigests []string
	if json.Unmarshal(raw, &repoDigests) != nil {
		return Result{}, errors.New("image digest response invalid")
	}
	var digest string
	for _, candidate := range repoDigests {
		if strings.HasPrefix(candidate, s.ImageRepository+"@") {
			digest = strings.TrimPrefix(candidate, s.ImageRepository+"@")
			break
		}
	}
	if !digestPattern.MatchString(digest) {
		return Result{}, errors.New("pushed image digest unavailable")
	}
	values := append(valuesArgs(s.ValuesFiles), "--set-string", "image.repository="+s.ImageRepository, "--set-string", "image.tag="+s.ImageTag, "--set-string", "image.digest="+digest)
	base := []string{"--kube-context", s.KubeContext, "--namespace", s.Namespace}
	template := append(append([]string{"template", s.Release, s.ChartDir}, base...), values...)
	if err := w.run(ctx, s.SourceDir, "helm template", "helm", template...); err != nil {
		return Result{}, err
	}
	upgrade := append(append([]string{"upgrade", "--install", s.Release, s.ChartDir}, base...), values...)
	upgrade = append(upgrade, "--atomic", "--wait", "--timeout", "5m")
	if err := w.run(ctx, s.SourceDir, "helm upgrade", "helm", upgrade...); err != nil {
		return Result{}, err
	}
	if err := w.run(ctx, s.SourceDir, "rollout status", "kubectl", "--context", s.KubeContext, "--namespace", s.Namespace, "rollout", "status", "deployment/"+s.Deployment, "--timeout=5m"); err != nil {
		return Result{}, err
	}
	return Result{Image: image, Digest: digest}, nil
}

func valuesArgs(files []string) []string {
	var args []string
	for _, file := range files {
		args = append(args, "--values", file)
	}
	return args
}

func (w Workflow) run(ctx context.Context, dir, step, name string, args ...string) error {
	w.progress(step)
	if err := w.Runner.Run(ctx, dir, name, args...); err != nil {
		return fmt.Errorf("%s failed: %w", step, err)
	}
	return nil
}

func (w Workflow) progress(step string) {
	if w.OnStep != nil {
		w.OnStep(step)
	}
}
