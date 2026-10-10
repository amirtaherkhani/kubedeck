package config

import (
	"os"
	"os/exec"
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
	t.Setenv("KUCHDESK_COREDNS_CONFIGMAP", "custom-dns")
	t.Setenv("KUCHDESK_COREDNS_COREFILE_KEY", "CustomCorefile")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.DNSManagementEnabled ||
		cfg.CoreDNSNamespace != "dns-system" ||
		cfg.CoreDNSConfigMap != "custom-dns" ||
		cfg.CoreDNSCorefileKey != "CustomCorefile" {
		t.Fatalf("unexpected DNS configuration: %#v", cfg)
	}
}

func TestRejectsLegacyCoreDNSSettings(t *testing.T) {
	setClusterIdentity(t)
	for _, legacy := range []string{"KUCHDESK_COREDNS_CUSTOM_CONFIGMAP", "KUCHDESK_COREDNS_OVERRIDE_KEY"} {
		t.Run(legacy, func(t *testing.T) {
			t.Setenv(legacy, "old")
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), legacy) {
				t.Fatalf("legacy setting error = %v", err)
			}
		})
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

func TestExecRequiresExplicitManagement(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_EXEC_ENABLED", "true")
	t.Setenv("KUCHDESK_MANAGEMENT_ENABLED", "false")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "KUCHDESK_MANAGEMENT_ENABLED") {
		t.Fatalf("exec gate error = %v", err)
	}
	t.Setenv("KUCHDESK_MANAGEMENT_ENABLED", "true")
	t.Setenv("KUCHDESK_AGENT_TOKEN", "test-token")
	cfg, err := Load()
	if err != nil || !cfg.ExecEnabled {
		t.Fatalf("explicit exec config = %#v, %v", cfg, err)
	}
}

func TestPortForwardRequiresExplicitManagement(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_PORT_FORWARD_ENABLED", "true")
	t.Setenv("KUCHDESK_MANAGEMENT_ENABLED", "false")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "KUCHDESK_MANAGEMENT_ENABLED") {
		t.Fatalf("port-forward gate error = %v", err)
	}
	t.Setenv("KUCHDESK_MANAGEMENT_ENABLED", "true")
	t.Setenv("KUCHDESK_AGENT_TOKEN", "test-token")
	cfg, err := Load()
	if err != nil || !cfg.PortForwardEnabled {
		t.Fatalf("explicit port-forward config = %#v, %v", cfg, err)
	}
}

func TestHostBridgeRequiresManagementAndDedicatedBearer(t *testing.T) {
	setClusterIdentity(t)
	t.Setenv("KUCHDESK_HOST_BRIDGE_ENABLED", "true")
	t.Setenv("KUCHDESK_HOST_BRIDGE_URL", "https://host.docker.internal:8181")
	t.Setenv("KUCHDESK_HOST_BRIDGE_TOKEN", "0123456789abcdef0123456789abcdef")
	t.Setenv("KUCHDESK_AGENT_TOKEN", "agent-token")
	if _, err := Load(); err == nil {
		t.Fatal("host bridge started without management")
	}
	t.Setenv("KUCHDESK_MANAGEMENT_ENABLED", "true")
	if cfg, err := Load(); err != nil || !cfg.HostBridgeEnabled {
		t.Fatalf("valid host bridge config = %#v, %v", cfg, err)
	}
	t.Setenv("KUCHDESK_HOST_BRIDGE_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("host bridge started without separate bearer")
	}
}

func TestChartHostBridgeUsesOnlySecretReferences(t *testing.T) {
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm is not installed")
	}
	chart := filepath.Join("..", "..", "chart")
	base := []string{"template", "agent", chart, "--set", "cluster.id=example", "--set", "cluster.name=Example", "--set", "image.repository=example.invalid/agent", "--set", "image.tag=git-123456789abc"}
	render := func(extra ...string) (string, error) {
		command := exec.Command(helm, append(append([]string{}, base...), extra...)...)
		output, err := command.CombinedOutput()
		return string(output), err
	}
	output, err := render()
	if err != nil || !strings.Contains(output, "KUCHDESK_HOST_BRIDGE_ENABLED") || strings.Contains(output, "KUCHDESK_HOST_BRIDGE_TOKEN") {
		t.Fatalf("disabled chart render = %v", err)
	}
	output, err = render("--set", "agent.managementEnabled=true", "--set", "rbac.clusterAdmin=true", "--set", "hostBridge.enabled=true", "--set", "hostBridge.url=https://host.docker.internal:8181", "--set", "hostBridge.tokenSecretName=host-bridge-auth", "--set", "hostBridge.caConfigMapName=host-bridge-ca")
	if err != nil || !strings.Contains(output, "name: \"host-bridge-auth\"") || !strings.Contains(output, "secretKeyRef:") || !strings.Contains(output, "host-bridge-ca") {
		t.Fatalf("enabled chart missing named Secret or CA reference: %v", err)
	}
	if strings.Contains(output, "INFISICAL_CLIENT_SECRET") || strings.Contains(output, "INFISICAL_CLIENT_ID") {
		t.Fatal("Infisical bootstrap credential appeared in rendered chart")
	}
	if _, err := render("--set", "hostBridge.token=plaintext-canary"); err == nil {
		t.Fatal("chart accepted a literal bridge token in Helm values")
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
