//go:build !darwin && !linux

package main

import (
	"context"
	"fmt"
)

func defaultInterface(context.Context) (string, error) {
	return "", fmt.Errorf("automatic interface discovery is unavailable on this OS; use -interface or -target-ip")
}
