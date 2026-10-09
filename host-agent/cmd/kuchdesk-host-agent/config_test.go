package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseConfigUsesExplicitClusterAndZone(t *testing.T) {
	cfg, err := parseConfig([]string{"-kube-context", "kind-team", "-zone", "Dev.Example.", "-namespace", "dns-system", "-service", "dns-api", "-admin-secret", "admin.credentials", "-target-ip", "10.2.3.4"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Zone != "dev.example" || cfg.recordName() != "*.dev.example" || cfg.TargetIP != "10.2.3.4" || cfg.TTL != 30 || cfg.Timeout.Seconds() != 30 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	wantArgs := []string{"--context", "kind-team", "-n", "dns-system", "get", "secret", "admin.credentials", "-o", "json"}
	if got := cfg.kubectlArgs("get", "secret", cfg.AdminSecret, "-o", "json"); !reflect.DeepEqual(got, wantArgs) {
		t.Fatalf("kubectl args = %v, want %v", got, wantArgs)
	}
}

func TestParseConfigAcceptsExplicitHostAndRoutableTargets(t *testing.T) {
	for _, target := range []string{"127.0.0.1", "8.8.8.8"} {
		cfg, err := parseConfig([]string{"-kube-context", "kind-team", "-zone", "dev.example", "-target-ip", target})
		if err != nil || cfg.TargetIP != target {
			t.Fatalf("target %s: cfg=%+v err=%v", target, cfg, err)
		}
	}
}

func TestParseConfigRejectsUnsafeOrAmbiguousInputs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing zone", args: []string{"-kube-context", "kind-team"}, want: "-zone"},
		{name: "missing context", args: []string{"-zone", "dev.example"}, want: "-kube-context"},
		{name: "invalid zone", args: []string{"-kube-context", "kind-team", "-zone", "*"}, want: "-zone"},
		{name: "multicast target", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-target-ip", "224.0.0.1"}, want: "-target-ip"},
		{name: "ambiguous target", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-target-ip", "10.2.3.4", "-interface", "en0"}, want: "cannot be used together"},
		{name: "invalid namespace", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-namespace", "dns.system"}, want: "namespace"},
		{name: "invalid port", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-api-port", "0"}, want: "-api-port"},
		{name: "invalid TTL", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-ttl", "0"}, want: "-ttl"},
		{name: "invalid timeout", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-timeout", "0s"}, want: "-timeout"},
		{name: "empty kubectl", args: []string{"-kube-context", "kind-team", "-zone", "dev.example", "-kubectl", " "}, want: "-kubectl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseConfig(%v) error = %v, want %q", tt.args, err, tt.want)
			}
		})
	}
}
