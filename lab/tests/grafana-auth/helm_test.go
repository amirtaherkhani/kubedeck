package grafanaauth_test

import (
	"os/exec"
	"strings"
	"testing"
)

func render(t *testing.T, template string, extra ...string) (string, error) {
	t.Helper()
	helm, e := exec.LookPath("helm")
	if e != nil {
		t.Skip("helm is required for chart contract tests")
	}
	args := []string{"template", "grafana", "../../apps/observability/grafana", "-n", "observability", "-f", "../../apps/observability/grafana/values.homelab.yaml", "--show-only", template}
	out, e := exec.Command(helm, append(args, extra...)...).CombinedOutput()
	return string(out), e
}
func nativeArgs() []string {
	return []string{"--set-string", "infisical.authMethod=kubernetesAuth", "--set-string", "infisical.kubernetesAuth.identityId=11111111-1111-1111-1111-111111111111", "--set-string", "infisical.kubernetesAuth.serviceAccountRef.name=fixture-reader", "--set-string", "infisical.kubernetesAuth.serviceAccountRef.namespace=fixture-namespace"}
}

func TestAuthenticationModesAndUnchangedWorkload(t *testing.T) {
	universal, e := render(t, "templates/infisicalsecret.yaml")
	if e != nil {
		t.Fatal(e, universal)
	}
	if !strings.Contains(universal, "universalAuth:") || strings.Contains(universal, "kubernetesAuth:") {
		t.Fatal("default auth changed")
	}
	native, e := render(t, "templates/infisicalsecret.yaml", nativeArgs()...)
	if e != nil {
		t.Fatal(e, native)
	}
	for _, want := range []string{"kubernetesAuth:", "identityId: \"11111111-1111-1111-1111-111111111111\"", "autoCreateServiceAccountToken: true", "name: \"fixture-reader\"", "namespace: \"fixture-namespace\"", "envSlug: \"dev\"", "secretName: grafana-admin"} {
		if !strings.Contains(native, want) {
			t.Fatalf("missing native contract %q", want)
		}
	}
	for _, forbidden := range []string{"universalAuth:", "credentialsRef:", "infisical-universal-auth"} {
		if strings.Contains(native, forbidden) {
			t.Fatalf("native auth retained %q", forbidden)
		}
	}
	for _, template := range []string{"templates/deployment.yaml", "templates/image-renderer-deployment.yaml"} {
		before, e := render(t, template)
		if e != nil {
			t.Fatal(e, before)
		}
		after, e := render(t, template, nativeArgs()...)
		if e != nil {
			t.Fatal(e, after)
		}
		if before != after {
			t.Fatalf("auth cutover changed workload %s", template)
		}
	}
}
func TestInvalidNativeConfigurationFailsClosed(t *testing.T) {
	for _, args := range [][]string{
		{"--set-string", "infisical.authMethod=unknown"},
		{"--set-string", "infisical.authMethod=kubernetesAuth"},
		append(nativeArgs(), "--set-string", "infisical.kubernetesAuth.serviceAccountRef.name="),
		append(nativeArgs(), "--set-string", "infisical.kubernetesAuth.serviceAccountRef.namespace="),
	} {
		if out, e := render(t, "templates/infisicalsecret.yaml", args...); e == nil {
			t.Fatalf("invalid auth accepted: %s", out)
		}
	}
}
