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

	"github.com/amirtaherkhani/kubedeck-agent/internal/agent"
	"github.com/amirtaherkhani/kubedeck-agent/internal/config"
	"github.com/amirtaherkhani/kubedeck-agent/internal/dnsconfig"
	"github.com/amirtaherkhani/kubedeck-agent/internal/httpapi"
	"github.com/amirtaherkhani/kubedeck-agent/internal/stream"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	if err := run(logger); err != nil {
		logger.Error("KubeDeck agent stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
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
		ConfigMapName: cfg.CoreDNSCustomConfigMap,
		OverrideKey:   cfg.CoreDNSOverrideKey,
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
	server := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       0,
		WriteTimeout:      0,
		MaxHeaderBytes:    1 << 20,
	}

	signalContext, stopSignals := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stopSignals()
	ctx, cancel := context.WithCancel(signalContext)
	defer cancel()

	collectorErrors := make(chan error, 1)
	go func() {
		collectorErrors <- collector.Run(ctx)
	}()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(
			"KubeDeck agent HTTP server started",
			"address", cfg.ListenAddress,
			"clusterId", cfg.ClusterID,
			"authenticationEnabled", cfg.BearerToken != "",
			"dnsManagementEnabled", cfg.DNSManagementEnabled,
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
