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
	flag.Parse()
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
	if len(allowed) == 0 && !*allowCreate {
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
	api, err := hostbridge.New(infisical.CommandService{Client: client, AllowedProjects: allowed, AllowProjectCreate: *allowCreate}, doctor.Service{}, bridgeToken)
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
