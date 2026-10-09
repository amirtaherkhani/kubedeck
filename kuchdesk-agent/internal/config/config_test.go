package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestDNSManagementRequiresAuthenticatedWrites(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_DNS_MANAGEMENT_ENABLED", "true")
	t.Setenv("KUCHDESK_AGENT_TOKEN", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "KUCHDESK_AGENT_TOKEN") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestDNSManagementLoadsCoreDNSTarget(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_DNS_MANAGEMENT_ENABLED", "true")
	t.Setenv("KUCHDESK_AGENT_TOKEN", "test-token")
	t.Setenv("KUCHDESK_COREDNS_NAMESPACE", "dns-system")
	t.Setenv("KUCHDESK_COREDNS_CUSTOM_CONFIGMAP", "custom-dns")
	t.Setenv("KUCHDESK_COREDNS_OVERRIDE_KEY", "services.override")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DNSManagementEnabled ||
		cfg.CoreDNSNamespace != "dns-system" ||
		cfg.CoreDNSCustomConfigMap != "custom-dns" ||
		cfg.CoreDNSOverrideKey != "services.override" {
		t.Fatalf("unexpected DNS configuration: %#v", cfg)
	}
}

func TestManagementRequiresBearerToken(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_MANAGEMENT_ENABLED", "true")
	t.Setenv("KUCHDESK_AGENT_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("management started without bearer token")
	}
	t.Setenv("KUCHDESK_AGENT_TOKEN", "test-token")
	cfg, err := Load()
	if err != nil || !cfg.ManagementEnabled {
		t.Fatalf("management config: %v, %#v", err, cfg)
	}
}

func TestClusterIdentityRequired(t *testing.T) {
	for _, missing := range []string{"KUCHDESK_CLUSTER_ID", "KUCHDESK_CLUSTER_NAME"} {
		t.Run(missing, func(t *testing.T) {
			setClusterIdentity(t)
			t.Setenv(missing, " ")
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("Load() error = %v; want %s", err, missing)
			}
		})
	}
}

func TestLoadReadsExplicitLocalKubeContext(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_KUBE_CONTEXT", "  kind-team  ")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.KubeContext != "kind-team" {
		t.Fatalf("context = %q", cfg.KubeContext)
	}
}

func setClusterIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("KUCHDESK_CLUSTER_ID", "test-cluster")
	t.Setenv("KUCHDESK_CLUSTER_NAME", "Test cluster")
}

func TestRESTConfigUsesHostKubeconfigWithoutInClusterCredentials(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	path := writeKubeconfig(t, "local", "https://local.example.test")
	t.Setenv("KUBECONFIG", path)

	restConfig, err := RESTConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if restConfig.Host != "https://local.example.test" {
		t.Fatalf("host = %q", restConfig.Host)
	}
}

func TestRESTConfigSupportsKubeconfigListAndExplicitContext(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	first := writeKubeconfig(t, "first", "https://first.example.test")
	second := writeKubeconfig(t, "second", "https://second.example.test")

	restConfig, err := RESTConfig(Config{
		Kubeconfig:  strings.Join([]string{first, second}, string(os.PathListSeparator)),
		KubeContext: "second",
	})
	if err != nil {
		t.Fatal(err)
	}
	if restConfig.Host != "https://second.example.test" {
		t.Fatalf("host = %q", restConfig.Host)
	}
}

func writeKubeconfig(t *testing.T, name, server string) string {
	t.Helper()
	config := clientcmdapi.NewConfig()
	config.Clusters[name] = &clientcmdapi.Cluster{Server: server}
	config.AuthInfos[name] = &clientcmdapi.AuthInfo{Token: "test-token"}
	config.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name}
	config.CurrentContext = name
	path := filepath.Join(t.TempDir(), "config")
	if err := clientcmd.WriteToFile(*config, path); err != nil {
		t.Fatal(err)
	}
	return path
}
