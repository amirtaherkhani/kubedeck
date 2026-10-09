package main

import (
	"net"
	"strings"
	"testing"
)

func TestPrivateIPv4RejectsAmbiguousInterfaceAddresses(t *testing.T) {
	address := func(value string) net.Addr {
		t.Helper()
		ip, network, err := net.ParseCIDR(value)
		if err != nil {
			t.Fatal(err)
		}
		return &net.IPNet{IP: ip, Mask: network.Mask}
	}
	selected, err := privateIPv4([]net.Addr{address("fe80::1/64"), address("192.168.1.25/24")})
	if err != nil || selected == nil || selected.String() != "192.168.1.25" {
		t.Fatalf("selected = %v, error = %v", selected, err)
	}
	_, err = privateIPv4([]net.Addr{address("192.168.1.25/24"), address("10.0.0.2/24")})
	if err == nil || !strings.Contains(err.Error(), "multiple private IPv4") {
		t.Fatalf("expected ambiguous address error, got %v", err)
	}
}

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
