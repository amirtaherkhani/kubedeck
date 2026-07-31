package dnsconfig

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coredns/caddy/caddyfile"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const (
	managedByAnnotation = "kubedeck.io/dns-managed-by"
	updatedAtAnnotation = "kubedeck.io/dns-updated-at"
	maxAliases          = 200
	maxOverrideBytes    = 64 * 1024
)

var (
	ErrDisabled    = errors.New("CoreDNS management is disabled")
	ErrUnavailable = errors.New("CoreDNS custom ConfigMap is unavailable")
	ErrConflict    = errors.New("CoreDNS custom ConfigMap changed; refresh and try again")
	ErrUnmanaged   = errors.New("CoreDNS override key contains configuration not managed by KubeDeck")
	ErrInvalid     = errors.New("invalid DNS alias configuration")
)

type Options struct {
	Enabled       bool
	Namespace     string
	ConfigMapName string
	OverrideKey   string
	ClusterDomain string
}

type Alias struct {
	Hostname  string `json:"hostname"`
	Service   string `json:"service"`
	Namespace string `json:"namespace"`
}

type State struct {
	Enabled         bool       `json:"enabled"`
	Available       bool       `json:"available"`
	Namespace       string     `json:"namespace"`
	ConfigMapName   string     `json:"configMapName"`
	OverrideKey     string     `json:"overrideKey"`
	ResourceVersion string     `json:"resourceVersion,omitempty"`
	Aliases         []Alias    `json:"aliases"`
	Rendered        string     `json:"rendered,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
	DryRun          bool       `json:"dryRun,omitempty"`
}

type ReplaceRequest struct {
	ResourceVersion string  `json:"resourceVersion"`
	Aliases         []Alias `json:"aliases"`
	DryRun          bool    `json:"dryRun,omitempty"`
}

type Manager struct {
	kube    kubernetes.Interface
	options Options
}

func New(kube kubernetes.Interface, options Options) *Manager {
	return &Manager{kube: kube, options: options}
}

func (m *Manager) Read(ctx context.Context) (State, error) {
	state := m.baseState()
	if !m.options.Enabled {
		return state, nil
	}

	configMap, err := m.kube.CoreV1().ConfigMaps(m.options.Namespace).Get(
		ctx,
		m.options.ConfigMapName,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return state, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read CoreDNS custom ConfigMap: %w", err)
	}
	return m.stateFromConfigMap(configMap, false)
}

func (m *Manager) Replace(ctx context.Context, request ReplaceRequest) (State, error) {
	if !m.options.Enabled {
		return State{}, ErrDisabled
	}

	aliases, err := m.validateAliases(ctx, request.Aliases)
	if err != nil {
		return State{}, err
	}
	rendered := render(aliases, m.options.ClusterDomain)
	if err := validateCorefile(rendered); err != nil {
		return State{}, fmt.Errorf("validate generated CoreDNS override: %w", err)
	}

	configMap, err := m.kube.CoreV1().ConfigMaps(m.options.Namespace).Get(
		ctx,
		m.options.ConfigMapName,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return State{}, fmt.Errorf("%w: %s/%s does not exist", ErrUnavailable, m.options.Namespace, m.options.ConfigMapName)
	}
	if err != nil {
		return State{}, fmt.Errorf("read CoreDNS custom ConfigMap: %w", err)
	}
	if strings.TrimSpace(request.ResourceVersion) == "" ||
		request.ResourceVersion != configMap.ResourceVersion {
		return State{}, ErrConflict
	}
	if existing := configMap.Data[m.options.OverrideKey]; strings.TrimSpace(existing) != "" {
		if _, err := parseManaged(existing); err != nil {
			return State{}, err
		}
	}

	candidate := configMap.DeepCopy()
	if candidate.Data == nil {
		candidate.Data = make(map[string]string)
	}
	if rendered == "" {
		delete(candidate.Data, m.options.OverrideKey)
	} else {
		candidate.Data[m.options.OverrideKey] = rendered
	}
	if candidate.Annotations == nil {
		candidate.Annotations = make(map[string]string)
	}
	now := time.Now().UTC()
	candidate.Annotations[managedByAnnotation] = "kubedeck-agent"
	candidate.Annotations[updatedAtAnnotation] = now.Format(time.RFC3339)

	if request.DryRun {
		state, err := m.stateFromConfigMap(candidate, true)
		if err != nil {
			return State{}, err
		}
		state.DryRun = true
		return state, nil
	}

	updated, err := m.kube.CoreV1().ConfigMaps(m.options.Namespace).Update(
		ctx,
		candidate,
		metav1.UpdateOptions{},
	)
	if apierrors.IsConflict(err) {
		return State{}, ErrConflict
	}
	if err != nil {
		return State{}, fmt.Errorf("update CoreDNS custom ConfigMap: %w", err)
	}
	return m.stateFromConfigMap(updated, false)
}

func (m *Manager) baseState() State {
	return State{
		Enabled:       m.options.Enabled,
		Available:     false,
		Namespace:     m.options.Namespace,
		ConfigMapName: m.options.ConfigMapName,
		OverrideKey:   m.options.OverrideKey,
		Aliases:       []Alias{},
	}
}

func (m *Manager) stateFromConfigMap(configMap *corev1.ConfigMap, dryRun bool) (State, error) {
	state := m.baseState()
	state.Available = true
	state.ResourceVersion = configMap.ResourceVersion
	state.Rendered = configMap.Data[m.options.OverrideKey]
	state.DryRun = dryRun

	aliases, err := parseManaged(state.Rendered)
	if err != nil {
		return State{}, err
	}
	state.Aliases = aliases
	if value := configMap.Annotations[updatedAtAnnotation]; value != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, value); parseErr == nil {
			state.UpdatedAt = &parsed
		}
	}
	return state, nil
}

func (m *Manager) validateAliases(ctx context.Context, input []Alias) ([]Alias, error) {
	if len(input) > maxAliases {
		return nil, invalidf("aliases cannot contain more than %d entries", maxAliases)
	}

	aliases := append([]Alias(nil), input...)
	seen := make(map[string]struct{}, len(aliases))
	serviceSuffix := ".svc." + strings.ToLower(strings.Trim(m.options.ClusterDomain, "."))
	for index := range aliases {
		alias := &aliases[index]
		alias.Hostname = normalizeDomain(alias.Hostname)
		alias.Service = strings.ToLower(strings.TrimSpace(alias.Service))
		alias.Namespace = strings.ToLower(strings.TrimSpace(alias.Namespace))

		if problems := validation.IsDNS1123Subdomain(alias.Hostname); len(problems) > 0 {
			return nil, invalidf("aliases[%d].hostname is invalid: %s", index, strings.Join(problems, ", "))
		}
		if strings.HasSuffix(alias.Hostname, serviceSuffix) {
			return nil, invalidf("aliases[%d].hostname cannot override Kubernetes service DNS", index)
		}
		if problems := validation.IsDNS1123Label(alias.Service); len(problems) > 0 {
			return nil, invalidf("aliases[%d].service is invalid: %s", index, strings.Join(problems, ", "))
		}
		if problems := validation.IsDNS1123Label(alias.Namespace); len(problems) > 0 {
			return nil, invalidf("aliases[%d].namespace is invalid: %s", index, strings.Join(problems, ", "))
		}
		if _, exists := seen[alias.Hostname]; exists {
			return nil, invalidf("aliases[%d].hostname is duplicated", index)
		}
		seen[alias.Hostname] = struct{}{}

		if _, err := m.kube.CoreV1().Services(alias.Namespace).Get(
			ctx,
			alias.Service,
			metav1.GetOptions{},
		); apierrors.IsNotFound(err) {
			return nil, invalidf("aliases[%d] target Service %s/%s does not exist", index, alias.Namespace, alias.Service)
		} else if err != nil {
			return nil, fmt.Errorf("verify aliases[%d] target Service: %w", index, err)
		}
	}
	return aliases, nil
}

func render(aliases []Alias, clusterDomain string) string {
	if len(aliases) == 0 {
		return ""
	}
	var output strings.Builder
	output.WriteString("# Managed by KubeDeck. Changes to this key are replaced by the agent.\n")
	for _, alias := range aliases {
		fmt.Fprintf(
			&output,
			"rewrite stop name exact %s %s.%s.svc.%s\n",
			alias.Hostname,
			alias.Service,
			alias.Namespace,
			strings.Trim(clusterDomain, "."),
		)
	}
	return output.String()
}

func parseManaged(input string) ([]Alias, error) {
	if len(input) > maxOverrideBytes {
		return nil, fmt.Errorf("%w: override is larger than %d bytes", ErrUnmanaged, maxOverrideBytes)
	}
	if strings.TrimSpace(input) == "" {
		return []Alias{}, nil
	}

	aliases := make([]Alias, 0)
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 6 ||
			fields[0] != "rewrite" ||
			fields[1] != "stop" ||
			fields[2] != "name" ||
			fields[3] != "exact" {
			return nil, fmt.Errorf("%w: unexpected directive %q", ErrUnmanaged, line)
		}
		targetParts := strings.Split(fields[5], ".")
		if len(targetParts) < 5 || targetParts[2] != "svc" {
			return nil, fmt.Errorf("%w: unexpected target %q", ErrUnmanaged, fields[5])
		}
		aliases = append(aliases, Alias{
			Hostname:  normalizeDomain(fields[4]),
			Service:   targetParts[0],
			Namespace: targetParts[1],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read CoreDNS override: %w", err)
	}
	return aliases, nil
}

func validateCorefile(override string) error {
	if override == "" {
		return nil
	}
	wrapped := ".:53 {\n" + override + "}\n"
	_, err := caddyfile.Parse(
		"kubedeck.override",
		strings.NewReader(wrapped),
		[]string{"rewrite"},
	)
	return err
}

func normalizeDomain(value string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(value), "."))
}

func invalidf(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
