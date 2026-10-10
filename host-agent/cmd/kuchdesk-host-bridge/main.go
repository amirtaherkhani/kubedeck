package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/doctor"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/enrollment"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/hostbridge"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "host bridge stopped:", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("KUCHDESK_HOST_BRIDGE_ENABLED") != "true" {
		return errors.New("set KUCHDESK_HOST_BRIDGE_ENABLED=true to start the network listener")
	}
	listen := flag.String("listen", "127.0.0.1:8181", "explicit host TCP listen address")
	baseURL := flag.String("url", "", "Infisical HTTPS origin")
	certificate := flag.String("tls-cert", "", "TLS certificate file")
	privateKey := flag.String("tls-key", "", "TLS private key file")
	projects := flag.String("project-ids", "", "comma-separated Infisical project ID allowlist")
	allowCreate := flag.Bool("allow-project-create", false, "allow organization-level project creation")
	policyPath := flag.String("enrollment-policy", "", "opt-in private runtime enrollment policy")
	stateDir := flag.String("enrollment-state-dir", "", "existing private enrollment state directory")
	flag.Parse()
	if *allowCreate {
		return errors.New("host bridge is read-only; project creation requires the local CLI or MCP")
	}
	if (*policyPath == "") != (*stateDir == "") {
		return errors.New("enrollment policy and state directory must be supplied together")
	}
	if flag.NArg() != 0 || *baseURL == "" || *certificate == "" || *privateKey == "" {
		return errors.New("Infisical URL and TLS certificate/key files are required")
	}
	if _, _, err := net.SplitHostPort(*listen); err != nil {
		return errors.New("listen must be an explicit host:port")
	}
	if _, err := tls.LoadX509KeyPair(*certificate, *privateKey); err != nil {
		return errors.New("TLS certificate or private key unavailable")
	}
	allowed := make(map[string]bool)
	for _, raw := range strings.Split(*projects, ",") {
		if id := strings.TrimSpace(raw); id != "" {
			allowed[id] = true
		}
	}
	if len(allowed) == 0 && !*allowCreate && *policyPath == "" {
		return errors.New("at least one project ID must be allowed")
	}
	clientID, clientSecret, err := infisical.HostCredentials(os.Getenv)
	if err != nil {
		return err
	}
	client, err := infisical.NewClient(*baseURL, clientID, clientSecret, nil)
	if err != nil || !client.Configured() {
		return errors.New("Infisical host credential is unavailable")
	}
	bridgeToken, err := infisical.HostBridgeToken(os.Getenv)
	if err != nil {
		return err
	}
	commands := infisical.CommandService{Client: client, AllowedProjects: allowed, ReadOnly: true}
	var diagnostics hostbridge.Diagnostician = doctor.Service{}
	var control *enrollment.Controller
	if *policyPath != "" {
		var store *enrollment.Store
		control, store, err = enrollment.OpenRuntime(*baseURL, *policyPath, *stateDir, client)
		if err != nil {
			return err
		}
		defer store.Close()
		commands.ProjectPolicy = control.Access
		diagnostics = enrollment.Diagnostics{Controller: control}
	}
	api, err := hostbridge.New(commands, diagnostics, bridgeToken)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              *listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       35 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if control != nil {
		done := make(chan struct{})
		go func() { defer close(done); control.Run(ctx, nil) }()
		defer func() { stop(); <-done }()
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	if err := server.ListenAndServeTLS(*certificate, *privateKey); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
