package dnsconfig

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const (
	managedByAnnotation = "kuchdesk.io/dns-managed-by"
	updatedAtAnnotation = "kuchdesk.io/dns-updated-at"
	maxAliases          = 200
	maxManagedBytes     = 64 * 1024
)

var (
	ErrDisabled    = errors.New("CoreDNS management is disabled")
	ErrUnavailable = errors.New("CoreDNS Corefile is unavailable or unsupported")
	ErrConflict    = errors.New("CoreDNS ConfigMap changed; refresh and try again")
	ErrUnmanaged   = errors.New("CoreDNS managed block contains unrecognized configuration")
	ErrInvalid     = errors.New("invalid DNS alias configuration")
)

type Options struct {
	Enabled       bool
	Namespace     string
	ConfigMapName string
	CorefileKey   string
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
	CorefileKey     string     `json:"corefileKey"`
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
		return State{}, fmt.Errorf("read CoreDNS ConfigMap: %w", err)
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
	if err := validateAliasDirectives(rendered); err != nil {
		return State{}, fmt.Errorf("validate generated CoreDNS aliases: %w", err)
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
		return State{}, fmt.Errorf("read CoreDNS ConfigMap: %w", err)
	}
	if strings.TrimSpace(request.ResourceVersion) == "" ||
		request.ResourceVersion != configMap.ResourceVersion {
		return State{}, ErrConflict
	}
	corefile := configMap.Data[m.options.CorefileKey]
	updatedCorefile, err := updateManagedCorefile(corefile, rendered, m.options.ClusterDomain)
	if err != nil {
		return State{}, err
	}

	candidate := configMap.DeepCopy()
	candidate.Data[m.options.CorefileKey] = updatedCorefile
	if candidate.Annotations == nil {
		candidate.Annotations = make(map[string]string)
	}
	now := time.Now().UTC()
	candidate.Annotations[managedByAnnotation] = "kuchdesk-agent"
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
		return State{}, fmt.Errorf("update CoreDNS ConfigMap: %w", err)
	}
	return m.stateFromConfigMap(updated, false)
}

func (m *Manager) baseState() State {
	return State{
		Enabled:       m.options.Enabled,
		Available:     false,
		Namespace:     m.options.Namespace,
		ConfigMapName: m.options.ConfigMapName,
		CorefileKey:   m.options.CorefileKey,
		Aliases:       []Alias{},
	}
}

func (m *Manager) stateFromConfigMap(configMap *corev1.ConfigMap, dryRun bool) (State, error) {
	state := m.baseState()
	state.Available = true
	state.ResourceVersion = configMap.ResourceVersion
	layout, err := inspectCorefile(configMap.Data[m.options.CorefileKey], m.options.ClusterDomain)
	if err != nil {
		return State{}, err
	}
	state.Rendered = layout.rendered
	state.DryRun = dryRun
	state.Aliases = layout.aliases
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

func normalizeDomain(value string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(value), "."))
}

func invalidf(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, arguments...))
}
