package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/agent"
	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/config"
	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/dnsconfig"
	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/httpapi"
	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/management"
	"github.com/amirtaherkhani/kuchdesk/kuchdesk-agent/internal/stream"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	if err := run(logger); err != nil {
		logger.Error("KuchDesk agent stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	signalContext, stopSignals := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stopSignals()
	restConfig, err := config.RESTConfig(cfg)
	if err != nil {
		return err
	}
	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	metrics, err := metricsclient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	broker := stream.NewBroker(cfg.ClusterID, cfg.SSEHistory)
	collector, err := agent.NewCollector(cfg, kube, metrics, broker, logger)
	if err != nil {
		return err
	}
	dnsManager := dnsconfig.New(kube, dnsconfig.Options{
		Enabled:       cfg.DNSManagementEnabled,
		Namespace:     cfg.CoreDNSNamespace,
		ConfigMapName: cfg.CoreDNSConfigMap,
		CorefileKey:   cfg.CoreDNSCorefileKey,
		ClusterDomain: cfg.ClusterDomain,
	})
	api := httpapi.New(
		collector,
		broker,
		dnsManager,
		cfg.BearerToken,
		cfg.SSEHeartbeat,
		logger,
	)
	if cfg.ManagementEnabled {
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		jobs := management.NewWorkloadJobs(signalContext)
		defer jobs.Close()
		api.SetManagementHandler((&management.Manager{Dynamic: dynamicClient, Discovery: kube.Discovery(), Kube: kube, RESTConfig: restConfig, Logger: logger, Jobs: jobs, ExecEnabled: cfg.ExecEnabled, PortForwardEnabled: cfg.PortForwardEnabled}).Handler())
	}
	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       0,
		WriteTimeout:      0,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, cancel := context.WithCancel(signalContext)
	defer cancel()

	collectorErrors := make(chan error, 1)
	go func() {
		collectorErrors <- collector.Run(ctx)
	}()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(
			"KuchDesk agent HTTP server started",
			"address", cfg.ListenAddress,
			"clusterId", cfg.ClusterID,
			"authenticationEnabled", cfg.BearerToken != "",
			"dnsManagementEnabled", cfg.DNSManagementEnabled,
			"managementEnabled", cfg.ManagementEnabled,
			"execEnabled", cfg.ExecEnabled,
			"portForwardEnabled", cfg.PortForwardEnabled,
		)
		serverErrors <- server.ListenAndServe()
	}()

	var runError error
	select {
	case <-signalContext.Done():
	case err = <-collectorErrors:
		if err != nil && !errors.Is(err, context.Canceled) {
			runError = err
		}
	case err = <-serverErrors:
		if err != nil && !httpapi.IsServerClosed(err) {
			runError = err
		}
	}
	cancel()

	shutdownContext, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownContext); err != nil && runError == nil {
		runError = err
	}
	return runError
}
