package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	maxCommandOutput = 1 << 20
	defaultMinDisk   = 2 << 30
)

var (
	identifier  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@/-]{0,127}$`)
	serviceName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,62}$`)
	versionName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.+_-]{0,39}$`)
)

type Target struct {
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
}

type Config struct {
	Domain       string   `json:"domain"`
	KubeContext  string   `json:"kubeContext"`
	RegistryURL  string   `json:"registryUrl"`
	DiskPath     string   `json:"diskPath"`
	MinFreeBytes uint64   `json:"minFreeBytes,omitempty"`
	Services     []Target `json:"services,omitempty"`
}

func (c Config) Validate() error {
	if !validDomain(c.Domain) || !identifier.MatchString(c.KubeContext) || !filepath.IsAbs(c.DiskPath) {
		return errors.New("domain, Kubernetes context, and absolute disk path are required")
	}
	u, err := url.Parse(c.RegistryURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("registry URL must be an HTTP origin without credentials")
	}
	if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1" {
		return errors.New("registry URL must use host loopback")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return errors.New("registry URL requires a valid explicit port")
	}
	if len(c.Services) > 16 {
		return errors.New("at most 16 deployment checks are supported")
	}
	seen := make(map[string]bool)
	for _, service := range c.Services {
		if !serviceName.MatchString(service.Namespace) || !serviceName.MatchString(service.Deployment) {
			return errors.New("deployment targets require valid namespace and name")
		}
		key := service.Namespace + "/" + service.Deployment
		if seen[key] {
			return errors.New("duplicate deployment target")
		}
		seen[key] = true
	}
	return nil
}

func validDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-') {
				return false
			}
		}
	}
	return true
}

type Check struct {
	ID             string         `json:"id"`
	Status         string         `json:"status"`
	Message        string         `json:"message"`
	Evidence       map[string]any `json:"evidence,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
}

type Finding struct {
	CheckID        string         `json:"checkId"`
	Severity       string         `json:"severity"`
	Problem        string         `json:"problem"`
	Evidence       map[string]any `json:"evidence,omitempty"`
	Recommendation string         `json:"recommendation"`
}

type AIItem struct {
	CheckID     string `json:"checkId"`
	Explanation string `json:"explanation"`
}

type AIResult struct {
	Status string   `json:"status"`
	Reason string   `json:"reason,omitempty"`
	Items  []AIItem `json:"items,omitempty"`
}

type Report struct {
	SchemaVersion string    `json:"schemaVersion"`
	Healthy       bool      `json:"healthy"`
	Checks        []Check   `json:"checks"`
	Findings      []Finding `json:"findings"`
	AI            *AIResult `json:"ai,omitempty"`
}

type CommandRunner interface {
	Output(context.Context, string, ...string) ([]byte, error)
}

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

// Provider receives only the whitelisted report facts, never command output,
// credentials, URLs, addresses, or Kubernetes object bodies.
type Provider interface {
	Interpret(context.Context, []Check) ([]byte, error)
}

type Service struct {
	Commands CommandRunner
	Resolver Resolver
	HTTP     *http.Client
	DiskFree func(string) (uint64, error)
	Provider Provider
}

type execRunner struct{}

func (execRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = filteredEnvironment()
	result, err := command.Output()
	if len(result) > maxCommandOutput {
		return nil, errors.New("command response exceeds limit")
	}
	return result, err
}

func filteredEnvironment() []string {
	var result []string
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "INFISICAL_CLIENT_ID=") || strings.HasPrefix(item, "INFISICAL_CLIENT_SECRET=") || strings.HasPrefix(item, "KUCHDESK_AGENT_TOKEN=") {
			continue
		}
		result = append(result, item)
	}
	return result
}

func freeBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

func (s Service) withDefaults() Service {
	if s.Commands == nil {
		s.Commands = execRunner{}
	}
	if s.Resolver == nil {
		s.Resolver = net.DefaultResolver
	}
	if s.HTTP == nil {
		s.HTTP = &http.Client{Timeout: 5 * time.Second}
	}
	if s.DiskFree == nil {
		s.DiskFree = freeBytes
	}
	return s
}

func (s Service) Run(ctx context.Context, config Config, ai bool) (Report, error) {
	if err := config.Validate(); err != nil {
		return Report{}, err
	}
	s = s.withDefaults()
	if config.MinFreeBytes == 0 {
		config.MinFreeBytes = defaultMinDisk
	}
	tasks := []func(context.Context) Check{
		func(ctx context.Context) Check { return s.network(ctx) },
		func(ctx context.Context) Check { return s.dns(ctx, config.Domain) },
		func(ctx context.Context) Check { return s.docker(ctx) },
		func(ctx context.Context) Check { return s.kind(ctx, config.KubeContext) },
		func(ctx context.Context) Check { return s.registry(ctx, config.RegistryURL) },
		func(context.Context) Check { return s.disk(config.DiskPath, config.MinFreeBytes) },
		func(ctx context.Context) Check { return s.metrics(ctx, config.KubeContext) },
	}
	for _, target := range config.Services {
		target := target
		tasks = append(tasks, func(ctx context.Context) Check { return s.deployment(ctx, config.KubeContext, target) })
	}
	checks := make([]Check, len(tasks))
	gate := make(chan struct{}, 4)
	var workers sync.WaitGroup
	for index, task := range tasks {
		workers.Add(1)
		go func() {
			defer workers.Done()
			select {
			case gate <- struct{}{}:
				defer func() { <-gate }()
			case <-ctx.Done():
				checks[index] = failure("canceled", "Doctor check canceled", "Run Doctor again when the request can complete")
				return
			}
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			checks[index] = task(checkCtx)
		}()
	}
	workers.Wait()
	report := Report{SchemaVersion: "kuchdesk.doctor/v1", Healthy: true, Checks: checks, Findings: []Finding{}}
	for _, check := range checks {
		if check.Status == "fail" {
			report.Healthy = false
		}
		if check.Status != "ok" {
			report.Findings = append(report.Findings, Finding{CheckID: check.ID, Severity: check.Status, Problem: check.Message, Evidence: check.Evidence, Recommendation: check.Recommendation})
		}
	}
	if ai {
		report.AI = s.interpret(ctx, checks)
	}
	return report, nil
}

func ok(id, message string, evidence map[string]any) Check {
	return Check{ID: id, Status: "ok", Message: message, Evidence: evidence}
}
func failure(id, message, recommendation string) Check {
	return Check{ID: id, Status: "fail", Message: message, Recommendation: recommendation}
}
func warning(id, message, recommendation string) Check {
	return Check{ID: id, Status: "warn", Message: message, Recommendation: recommendation}
}

func (s Service) network(ctx context.Context) Check {
	raw, err := s.Commands.Output(ctx, "route", "-n", "get", "default")
	if err != nil {
		return failure("network", "Default route unavailable", "Check the Mac network route and VPN state")
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 2 && fields[0] == "interface:" && serviceName.MatchString(fields[1]) {
			if strings.HasPrefix(fields[1], "utun") {
				return Check{ID: "network", Status: "warn", Message: "VPN owns the default route", Evidence: map[string]any{"interface": fields[1]}, Recommendation: "Verify the physical interface used for local ingress and DNS"}
			}
			return ok("network", "Default route found", map[string]any{"interface": fields[1]})
		}
	}
	return failure("network", "Default route interface unknown", "Inspect the Mac route table")
}

func (s Service) dns(ctx context.Context, domain string) Check {
	addresses, err := s.Resolver.LookupIPAddr(ctx, domain)
	if err != nil || len(addresses) == 0 {
		return failure("dns", "Configured domain does not resolve", "Check Technitium and the host DNS resolver")
	}
	return ok("dns", "Configured domain resolves", map[string]any{"answerCount": len(addresses)})
}

func (s Service) docker(ctx context.Context) Check {
	raw, err := s.Commands.Output(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	version := strings.TrimSpace(string(raw))
	if err != nil || !versionName.MatchString(version) {
		return failure("docker", "Docker daemon unavailable", "Start Docker Desktop and check the selected context")
	}
	return ok("docker", "Docker daemon responds", map[string]any{"serverVersion": version})
}

func (s Service) kind(ctx context.Context, kubeContext string) Check {
	raw, err := s.Commands.Output(ctx, "kubectl", "--context", kubeContext, "get", "nodes", "-o", "json")
	var body struct {
		Items []struct {
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}
	if err != nil || len(raw) > maxCommandOutput || json.Unmarshal(raw, &body) != nil || len(body.Items) == 0 {
		return failure("kind", "Kubernetes nodes unavailable", "Check Docker Desktop Kubernetes and the selected context")
	}
	ready := 0
	for _, node := range body.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				ready++
				break
			}
		}
	}
	evidence := map[string]any{"nodes": len(body.Items), "readyNodes": ready}
	if ready != len(body.Items) {
		return Check{ID: "kind", Status: "fail", Message: "Kubernetes has unready nodes", Evidence: evidence, Recommendation: "Inspect node conditions before deployment"}
	}
	return ok("kind", "Kubernetes nodes ready", evidence)
}

func (s Service) registry(ctx context.Context, origin string) Check {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(origin, "/")+"/v2/", nil)
	if err != nil {
		return failure("registry", "Registry URL invalid", "Correct the registry origin")
	}
	client := *s.HTTP
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return failure("registry", "Local registry unreachable", "Check the registry Deployment and host port-forward")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Check{ID: "registry", Status: "fail", Message: "Local registry rejected the probe", Evidence: map[string]any{"httpStatus": response.StatusCode}, Recommendation: "Check registry authentication and host forwarding"}
	}
	return ok("registry", "Local registry responds", map[string]any{"httpStatus": response.StatusCode})
}

func (s Service) disk(path string, minimum uint64) Check {
	free, err := s.DiskFree(path)
	if err != nil {
		return failure("disk", "Free disk space unavailable", "Check the source volume and filesystem")
	}
	evidence := map[string]any{"freeGiB": free / (1 << 30), "minimumGiB": minimum / (1 << 30)}
	if free < minimum {
		return Check{ID: "disk", Status: "fail", Message: "Insufficient free disk space", Evidence: evidence, Recommendation: "Free space before building or pulling images"}
	}
	return ok("disk", "Disk space sufficient", evidence)
}

func (s Service) metrics(ctx context.Context, kubeContext string) Check {
	raw, err := s.Commands.Output(ctx, "kubectl", "--context", kubeContext, "top", "nodes", "--no-headers")
	if err != nil || len(raw) > maxCommandOutput {
		return warning("resources", "Node metrics unavailable", "Check metrics-server before resource-sensitive deployment")
	}
	count, maxCPU, maxMemory := 0, 0, 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 5 {
			return warning("resources", "Node metrics unavailable", "Check metrics-server before resource-sensitive deployment")
		}
		cpu, cpuErr := strconv.Atoi(strings.TrimSuffix(fields[2], "%"))
		memory, memoryErr := strconv.Atoi(strings.TrimSuffix(fields[4], "%"))
		if cpuErr != nil || memoryErr != nil || !strings.HasSuffix(fields[2], "%") || !strings.HasSuffix(fields[4], "%") || cpu < 0 || cpu > 100 || memory < 0 || memory > 100 {
			return warning("resources", "Node metrics unavailable", "Check metrics-server before resource-sensitive deployment")
		}
		count++
		maxCPU = max(maxCPU, cpu)
		maxMemory = max(maxMemory, memory)
	}
	if count == 0 {
		return warning("resources", "Node metrics unavailable", "Check metrics-server before resource-sensitive deployment")
	}
	evidence := map[string]any{"metricNodes": count, "maxCPUPercent": maxCPU, "maxMemoryPercent": maxMemory}
	if maxCPU >= 95 || maxMemory >= 95 {
		return Check{ID: "resources", Status: "fail", Message: "Node resource use is critical", Evidence: evidence, Recommendation: "Reduce load or add capacity before deployment"}
	}
	if maxCPU >= 85 || maxMemory >= 85 {
		return Check{ID: "resources", Status: "warn", Message: "Node resource use is high", Evidence: evidence, Recommendation: "Check node capacity before deployment"}
	}
	return ok("resources", "Node resource use within limits", evidence)
}

func (s Service) deployment(ctx context.Context, kubeContext string, target Target) Check {
	id := "service:" + target.Namespace + "/" + target.Deployment
	raw, err := s.Commands.Output(ctx, "kubectl", "--context", kubeContext, "-n", target.Namespace, "get", "deployment", target.Deployment, "-o", "json")
	var body struct {
		Spec struct {
			Replicas *int `json:"replicas"`
		} `json:"spec"`
		Status struct {
			ReadyReplicas int `json:"readyReplicas"`
		} `json:"status"`
	}
	if err != nil || len(raw) > maxCommandOutput || json.Unmarshal(raw, &body) != nil {
		return failure(id, "Deployment unavailable", "Inspect the named Kubernetes Deployment")
	}
	desired := 1
	if body.Spec.Replicas != nil {
		desired = *body.Spec.Replicas
	}
	evidence := map[string]any{"desired": desired, "ready": body.Status.ReadyReplicas}
	if desired == 0 || body.Status.ReadyReplicas < desired {
		return Check{ID: id, Status: "fail", Message: "Deployment not ready", Evidence: evidence, Recommendation: "Inspect Pods, events, and rollout status"}
	}
	return ok(id, "Deployment ready", evidence)
}

func (s Service) interpret(ctx context.Context, checks []Check) *AIResult {
	if s.Provider == nil {
		return &AIResult{Status: "unavailable", Reason: "provider_not_configured"}
	}
	aiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	facts := make([]Check, len(checks))
	for index, check := range checks {
		facts[index] = check
		facts[index].Evidence = maps.Clone(check.Evidence)
	}
	raw, err := s.Provider.Interpret(aiCtx, facts)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(aiCtx.Err(), context.DeadlineExceeded) {
			return &AIResult{Status: "error", Reason: "ai_timeout"}
		}
		return &AIResult{Status: "error", Reason: "ai_provider_failed"}
	}
	if len(raw) == 0 || len(raw) > 4096 {
		return &AIResult{Status: "error", Reason: "malformed_ai_output"}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var parsed struct {
		Items []AIItem `json:"items"`
	}
	if decoder.Decode(&parsed) != nil || !errors.Is(decoder.Decode(&struct{}{}), io.EOF) || len(parsed.Items) > 8 {
		return &AIResult{Status: "error", Reason: "malformed_ai_output"}
	}
	if aiCtx.Err() != nil {
		return &AIResult{Status: "error", Reason: "ai_timeout"}
	}
	known := make(map[string]bool)
	for _, check := range checks {
		known[check.ID] = true
	}
	seen := make(map[string]bool)
	for _, item := range parsed.Items {
		if !known[item.CheckID] || seen[item.CheckID] || len(item.Explanation) == 0 || len(item.Explanation) > 500 {
			return &AIResult{Status: "error", Reason: "malformed_ai_output"}
		}
		seen[item.CheckID] = true
	}
	return &AIResult{Status: "ok", Items: parsed.Items}
}
