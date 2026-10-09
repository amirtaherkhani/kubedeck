package main

import "testing"

func TestParseLinuxDefaultRouteRejectsAmbiguousInterfaces(t *testing.T) {
	device, err := parseLinuxDefaultRoute("default via 10.0.0.1 dev eth0 metric 100\ndefault via 10.1.0.1 dev eth1 metric 200\n")
	if err == nil || device != "" {
		t.Fatalf("device = %q, error = %v; want ambiguity", device, err)
	}
	device, err = parseLinuxDefaultRoute("default via 10.0.0.1 dev eth0 proto dhcp\n")
	if err != nil || device != "eth0" {
		t.Fatalf("device = %q, error = %v; want eth0", device, err)
	}
}
