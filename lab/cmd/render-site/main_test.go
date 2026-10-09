package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadProfile(t *testing.T) siteProfile {
	t.Helper()
	data, err := os.ReadFile("../../site.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile siteProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		t.Fatal(err)
	}
	return profile
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected object, got %T", value)
	}
	return result
}

func TestCurrentSiteAndFallbackCertificate(t *testing.T) {
	result, err := render(loadProfile(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if result.summary["secretName"] != "local-dev-tls" {
		t.Fatalf("unexpected Secret: %v", result.summary["secretName"])
	}
	storeSpec := object(t, object(t, result.resources["20-default-tlsstore.json"])["spec"])
	certSpec := object(t, object(t, result.resources["10-certificate-kube-system.json"])["spec"])
	if object(t, storeSpec["defaultCertificate"])["secretName"] != certSpec["secretName"] {
		t.Fatal("fallback TLSStore has no matching Certificate Secret")
	}
	grafana := object(t, result.overlays["grafana-values.json"])
	server := object(t, object(t, grafana["grafana.ini"])["server"])
	if server["root_url"] != "https://grafana.local.dev" {
		t.Fatalf("unexpected root_url: %v", server["root_url"])
	}
}

func TestAlternateDomainAndExistingIssuer(t *testing.T) {
	profile := loadProfile(t)
	profile.TLS.Issuer.Type = "existing"
	profile.TLS.Issuer.Name = "acme-dns01"
	result, err := render(profile, "example.internal")
	if err != nil {
		t.Fatal(err)
	}
	if result.summary["secretName"] != "example-internal-tls" {
		t.Fatalf("unexpected Secret: %v", result.summary["secretName"])
	}
	if _, exists := result.resources["00-selfsigned-issuer.json"]; exists {
		t.Fatal("existing issuer profile generated a private CA")
	}
	certSpec := object(t, object(t, result.resources["10-certificate-kube-system.json"])["spec"])
	if _, exists := certSpec["duration"]; exists {
		t.Fatal("external issuer has an incompatible fixed duration")
	}
	data, err := json.Marshal([]any{result.resources, result.overlays, result.summary})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "local.dev") {
		t.Fatal("alternate domain retained the old site name")
	}
}

func TestRejectsMissingFallbackAndMultilevelHost(t *testing.T) {
	profile := loadProfile(t)
	profile.TLS.CertificateNamespaces = []string{"platform-system", "observability"}
	if _, err := render(profile, ""); err == nil || !strings.Contains(err.Error(), "default TLSStore namespace") {
		t.Fatalf("expected TLSStore namespace error, got %v", err)
	}
	profile = loadProfile(t)
	profile.Hosts.Grafana = "sub.grafana"
	if _, err := render(profile, ""); err == nil || !strings.Contains(err.Error(), "one label") {
		t.Fatalf("expected wildcard host error, got %v", err)
	}
}

func TestRerenderRemovesStaleCAResources(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "site")
	profile := loadProfile(t)
	initial, err := render(profile, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeOutput(dir, initial); err != nil {
		t.Fatal(err)
	}
	profile.TLS.Issuer.Type = "existing"
	profile.TLS.Issuer.Name = "acme-dns01"
	alternate, err := render(profile, "example.internal")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeOutput(dir, alternate); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "manifests", "00-selfsigned-issuer.json")); !os.IsNotExist(err) {
		t.Fatalf("stale CA manifest survived rerender: %v", err)
	}
	value, err := summaryField(dir, "secretName")
	if err != nil {
		t.Fatal(err)
	}
	if value != "example-internal-tls" {
		t.Fatalf("summary returned profile default instead of rendered value: %s", value)
	}
}
