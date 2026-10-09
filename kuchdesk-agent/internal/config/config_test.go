package config

import (
	"strings"
	"testing"
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

func setClusterIdentity(t *testing.T) {
	t.Helper()
	t.Setenv("KUCHDESK_CLUSTER_ID", "test-cluster")
	t.Setenv("KUCHDESK_CLUSTER_NAME", "Test cluster")
}
