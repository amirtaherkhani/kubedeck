//go:build darwin

package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func defaultInterface(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "route", "-n", "get", "default").Output()
	if err != nil {
		return "", fmt.Errorf("find default route: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "interface:" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("default route has no interface; use -interface or -target-ip")
}
