package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/amirtaherkhani/kubedeck-agent/internal/config"
	"github.com/amirtaherkhani/kubedeck-agent/internal/model"
	"github.com/amirtaherkhani/kubedeck-agent/internal/stream"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"

	"k8s.io/client-go/informers"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1"
	networkinginformers "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Collector struct {
	config                 config.Config
	kube                   kubernetes.Interface
	metrics                metricsclient.Interface
	broker                 *stream.Broker
	logger                 *slog.Logger
	factory                informers.SharedInformerFactory
	nodes                  coreinformers.NodeInformer
	namespaces             coreinformers.NamespaceInformer
	pods                   coreinformers.PodInformer
	services               coreinformers.ServiceInformer
	events                 coreinformers.EventInformer
	persistentVolumes      coreinformers.PersistentVolumeInformer
	persistentVolumeClaims coreinformers.PersistentVolumeClaimInformer
	deployments            appsinformers.DeploymentInformer
	statefulSets           appsinformers.StatefulSetInformer
	daemonSets             appsinformers.DaemonSetInformer
	ingresses              networkinginformers.IngressInformer
	endpointSlices         discoveryinformers.EndpointSliceInformer
	trigger                chan struct{}
	current                atomic.Pointer[model.Snapshot]
	ready                  atomic.Bool
	metricsMu              sync.RWMutex
	nodeMetrics            map[string]metricsv1beta1.NodeMetrics
	podMetrics             map[string]metricsv1beta1.PodMetrics
	serverVersion          string
	platform               string
}

func NewCollector(
	cfg config.Config,
	kube kubernetes.Interface,
	metrics metricsclient.Interface,
	broker *stream.Broker,
	logger *slog.Logger,
) (*Collector, error) {
	factory := informers.NewSharedInformerFactory(kube, 30*time.Minute)
	collector := &Collector{
		config:                 cfg,
		kube:                   kube,
		metrics:                metrics,
		broker:                 broker,
		logger:                 logger,
		factory:                factory,
		nodes:                  factory.Core().V1().Nodes(),
		namespaces:             factory.Core().V1().Namespaces(),
		pods:                   factory.Core().V1().Pods(),
		services:               factory.Core().V1().Services(),
		events:                 factory.Core().V1().Events(),
		persistentVolumes:      factory.Core().V1().PersistentVolumes(),
		persistentVolumeClaims: factory.Core().V1().PersistentVolumeClaims(),
		deployments:            factory.Apps().V1().Deployments(),
		statefulSets:           factory.Apps().V1().StatefulSets(),
		daemonSets:             factory.Apps().V1().DaemonSets(),
		ingresses:              factory.Networking().V1().Ingresses(),
		endpointSlices:         factory.Discovery().V1().EndpointSlices(),
		trigger:                make(chan struct{}, 1),
		nodeMetrics:            make(map[string]metricsv1beta1.NodeMetrics),
		podMetrics:             make(map[string]metricsv1beta1.PodMetrics),
		serverVersion:          "unknown",
	}

	if version, err := kube.Discovery().ServerVersion(); err == nil {
		collector.serverVersion = version.GitVersion
		collector.platform = version.Platform
	} else {
		logger.Warn("Kubernetes version discovery failed", "error", err)
	}
	if err := collector.registerHandlers(); err != nil {
		return nil, err
	}
	return collector, nil
}

func (c *Collector) Run(ctx context.Context) error {
	go c.refreshLoop(ctx)
	c.factory.StartWithContext(ctx)

	if !cache.WaitForCacheSync(
		ctx.Done(),
		c.nodes.Informer().HasSynced,
		c.namespaces.Informer().HasSynced,
		c.pods.Informer().HasSynced,
		c.services.Informer().HasSynced,
		c.events.Informer().HasSynced,
		c.persistentVolumes.Informer().HasSynced,
		c.persistentVolumeClaims.Informer().HasSynced,
		c.deployments.Informer().HasSynced,
		c.statefulSets.Informer().HasSynced,
		c.daemonSets.Informer().HasSynced,
		c.ingresses.Informer().HasSynced,
		c.endpointSlices.Informer().HasSynced,
	) {
		if cause := context.Cause(ctx); cause != nil {
			return fmt.Errorf("Kubernetes informer caches did not synchronize: %w", cause)
		}
		return fmt.Errorf("Kubernetes informer caches did not synchronize")
	}

	if err := c.refreshMetrics(ctx); err != nil {
		c.logger.Warn("initial Kubernetes Metrics API collection failed", "error", err)
	}
	if err := c.refreshSnapshot(); err != nil {
		return fmt.Errorf("build initial cluster snapshot: %w", err)
	}
	c.ready.Store(true)
	go c.metricsLoop(ctx)

	c.logger.Info(
		"Kubernetes caches synchronized",
		"clusterId", c.config.ClusterID,
		"kubernetesVersion", c.serverVersion,
	)
	<-ctx.Done()
	return nil
}

func (c *Collector) Ready() bool {
	return c.ready.Load()
}

func (c *Collector) Snapshot() model.Snapshot {
	current := c.current.Load()
	if current == nil {
		return model.Snapshot{
			SchemaVersion: model.SchemaVersion,
			Cluster:       model.Cluster{ID: c.config.ClusterID, Name: c.config.ClusterName},
		}
	}
	return *current
}

func (c *Collector) registerHandlers() error {
	informersByKind := []struct {
		kind     string
		informer cache.SharedIndexInformer
	}{
		{"Node", c.nodes.Informer()},
		{"Namespace", c.namespaces.Informer()},
		{"Pod", c.pods.Informer()},
		{"Service", c.services.Informer()},
		{"Event", c.events.Informer()},
		{"PersistentVolume", c.persistentVolumes.Informer()},
		{"PersistentVolumeClaim", c.persistentVolumeClaims.Informer()},
		{"Deployment", c.deployments.Informer()},
		{"StatefulSet", c.statefulSets.Informer()},
		{"DaemonSet", c.daemonSets.Informer()},
		{"Ingress", c.ingresses.Informer()},
		{"EndpointSlice", c.endpointSlices.Informer()},
	}

	for _, entry := range informersByKind {
		kind := entry.kind
		if _, err := entry.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(any) {
				c.queueRefresh()
			},
			UpdateFunc: func(_, _ any) {
				c.queueRefresh()
			},
			DeleteFunc: func(any) {
				c.queueRefresh()
			},
		}); err != nil {
			return fmt.Errorf("register %s event handler: %w", kind, err)
		}
	}
	return nil
}

func (c *Collector) queueRefresh() {
	select {
	case c.trigger <- struct{}{}:
	default:
	}
}

func (c *Collector) refreshLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.trigger:
		}

		timer := time.NewTimer(c.config.RefreshDebounce)
		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-c.trigger:
				if !timer.Stop() {
					<-timer.C
				}
				timer.Reset(c.config.RefreshDebounce)
			case <-timer.C:
				if c.ready.Load() {
					if err := c.refreshSnapshot(); err != nil {
						c.logger.Error("cluster snapshot refresh failed", "error", err)
					}
				}
				goto refreshed
			}
		}
	refreshed:
	}
}

func (c *Collector) metricsLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.MetricsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.refreshMetrics(ctx); err != nil {
				c.logger.Warn("Kubernetes Metrics API collection failed", "error", err)
				continue
			}
			c.queueRefresh()
		}
	}
}

func (c *Collector) refreshMetrics(ctx context.Context) error {
	nodeList, nodeErr := c.metrics.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	podList, podErr := c.metrics.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if nodeErr != nil && podErr != nil {
		return fmt.Errorf("nodes: %v; pods: %v", nodeErr, podErr)
	}

	c.metricsMu.Lock()
	defer c.metricsMu.Unlock()
	if nodeErr == nil {
		c.nodeMetrics = make(map[string]metricsv1beta1.NodeMetrics, len(nodeList.Items))
		for _, item := range nodeList.Items {
			c.nodeMetrics[item.Name] = item
		}
	}
	if podErr == nil {
		c.podMetrics = make(map[string]metricsv1beta1.PodMetrics, len(podList.Items))
		for _, item := range podList.Items {
			c.podMetrics[item.Namespace+"/"+item.Name] = item
		}
	}
	return nil
}
