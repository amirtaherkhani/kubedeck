package main

import (
	"fmt"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/enrollmentcli"
	"os"
)

func main() {
	if err := enrollmentcli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
