package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	ListenAddress          string
	ClusterID              string
	ClusterName            string
	ClusterDomain          string
	Kubeconfig             string
	BearerToken            string
	DNSManagementEnabled   bool
	CoreDNSNamespace       string
	CoreDNSCustomConfigMap string
	CoreDNSOverrideKey     string
	MetricsInterval        time.Duration
	RefreshDebounce        time.Duration
	SSEHeartbeat           time.Duration
	SSEHistory             int
	EventLimit             int
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddress:          envOrDefault("KUBEDECK_AGENT_LISTEN_ADDRESS", ":8080"),
		ClusterID:              envOrDefault("KUBEDECK_CLUSTER_ID", "default"),
		ClusterName:            envOrDefault("KUBEDECK_CLUSTER_NAME", "Kubernetes"),
		ClusterDomain:          strings.Trim(envOrDefault("KUBEDECK_CLUSTER_DOMAIN", "cluster.local"), "."),
		Kubeconfig:             strings.TrimSpace(os.Getenv("KUBECONFIG")),
		BearerToken:            strings.TrimSpace(os.Getenv("KUBEDECK_AGENT_TOKEN")),
		CoreDNSNamespace:       envOrDefault("KUBEDECK_COREDNS_NAMESPACE", "kube-system"),
		CoreDNSCustomConfigMap: envOrDefault("KUBEDECK_COREDNS_CUSTOM_CONFIGMAP", "coredns-custom"),
		CoreDNSOverrideKey:     envOrDefault("KUBEDECK_COREDNS_OVERRIDE_KEY", "kubedeck.override"),
		MetricsInterval:        10 * time.Second,
		RefreshDebounce:        250 * time.Millisecond,
		SSEHeartbeat:           15 * time.Second,
		SSEHistory:             256,
		EventLimit:             100,
	}

	var err error
	if cfg.DNSManagementEnabled, err = boolEnv("KUBEDECK_DNS_MANAGEMENT_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.MetricsInterval, err = durationEnv("KUBEDECK_METRICS_INTERVAL", cfg.MetricsInterval); err != nil {
		return Config{}, err
	}
	if cfg.RefreshDebounce, err = durationEnv("KUBEDECK_REFRESH_DEBOUNCE", cfg.RefreshDebounce); err != nil {
		return Config{}, err
	}
	if cfg.SSEHeartbeat, err = durationEnv("KUBEDECK_SSE_HEARTBEAT", cfg.SSEHeartbeat); err != nil {
		return Config{}, err
	}
	if cfg.SSEHistory, err = intEnv("KUBEDECK_SSE_HISTORY", cfg.SSEHistory, 8, 4096); err != nil {
		return Config{}, err
	}
	if cfg.EventLimit, err = intEnv("KUBEDECK_EVENT_LIMIT", cfg.EventLimit, 0, 1000); err != nil {
		return Config{}, err
	}

	if cfg.ClusterID == "" {
		return Config{}, errors.New("KUBEDECK_CLUSTER_ID cannot be empty")
	}
	if cfg.ClusterDomain == "" {
		return Config{}, errors.New("KUBEDECK_CLUSTER_DOMAIN cannot be empty")
	}
	if cfg.DNSManagementEnabled {
		if cfg.BearerToken == "" {
			return Config{}, errors.New("KUBEDECK_AGENT_TOKEN is required when DNS management is enabled")
		}
		if cfg.CoreDNSNamespace == "" || cfg.CoreDNSCustomConfigMap == "" {
			return Config{}, errors.New("CoreDNS namespace and custom ConfigMap cannot be empty")
		}
		if !strings.HasSuffix(cfg.CoreDNSOverrideKey, ".override") {
			return Config{}, errors.New("KUBEDECK_COREDNS_OVERRIDE_KEY must end with .override")
		}
	}
	return cfg, nil
}

func RESTConfig(cfg Config) (*rest.Config, error) {
	var (
		restConfig *rest.Config
		err        error
	)
	if cfg.Kubeconfig != "" {
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	} else {
		restConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("load Kubernetes client configuration: %w", err)
	}

	restConfig.UserAgent = "kubedeck-agent/0.1.1"
	restConfig.QPS = 30
	restConfig.Burst = 60
	restConfig.Timeout = 30 * time.Second
	return restConfig, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return parsed, nil
}

func intEnv(key string, fallback, minimum, maximum int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum || parsed > maximum {
		return 0, fmt.Errorf("%s must be between %d and %d", key, minimum, maximum)
	}
	return parsed, nil
}
