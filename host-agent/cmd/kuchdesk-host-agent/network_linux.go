//go:build linux

package main

import (
	"context"
	"fmt"
	"os/exec"
)

func defaultInterface(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "ip", "-4", "route", "show", "default").Output()
	if err != nil {
		return "", fmt.Errorf("find default route with iproute2: %w", err)
	}
	return parseLinuxDefaultRoute(string(out))
}
