package main

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func lanIP(ctx context.Context, override string) (net.IP, error) {
	device := override
	if device == "" {
		var err error
		device, err = defaultInterface(ctx)
		if err != nil {
			return nil, err
		}
	}
	if device == "" || tunnelInterface(device) {
		return nil, fmt.Errorf("no physical LAN interface from default route (%s); use -interface or -target-ip", device)
	}
	ifc, err := net.InterfaceByName(device)
	if err != nil {
		return nil, fmt.Errorf("find interface %s: %w", device, err)
	}
	if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
		return nil, fmt.Errorf("interface %s is not an active LAN interface", device)
	}
	addrs, err := ifc.Addrs()
	if err != nil {
		return nil, err
	}
	ip, err := privateIPv4(addrs)
	if err != nil {
		return nil, fmt.Errorf("interface %s: %w", device, err)
	}
	return ip, nil
}

func privateIPv4(addrs []net.Addr) (net.IP, error) {
	var selected net.IP
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err == nil && ip.To4() != nil && ip.IsPrivate() && !ip.IsLinkLocalUnicast() {
			if selected != nil && !selected.Equal(ip.To4()) {
				return nil, fmt.Errorf("multiple private IPv4 addresses; use -target-ip")
			}
			selected = ip.To4()
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no private IPv4 address")
	}
	return selected, nil
}

func tunnelInterface(name string) bool {
	for _, prefix := range []string{"utun", "tun", "tap", "wg", "ppp"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func parseLinuxDefaultRoute(output string) (string, error) {
	var device string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] != "dev" {
				continue
			}
			if device != "" && device != fields[i+1] {
				return "", fmt.Errorf("multiple default route interfaces; use -interface or -target-ip")
			}
			device = fields[i+1]
			break
		}
	}
	if device == "" {
		return "", fmt.Errorf("default route has no interface; use -interface or -target-ip")
	}
	return device, nil
}
