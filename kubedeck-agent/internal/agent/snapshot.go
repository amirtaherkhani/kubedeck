package agent

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/amirtaherkhani/kubedeck-agent/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type endpointCounts struct {
	ready int
	total int
}

func (c *Collector) refreshSnapshot() error {
	snapshot, err := c.buildSnapshot()
	if err != nil {
		return err
	}
	c.current.Store(&snapshot)
	if _, err := c.broker.Publish("snapshot", snapshot); err != nil {
		return fmt.Errorf("publish snapshot: %w", err)
	}
	return nil
}

func (c *Collector) buildSnapshot() (model.Snapshot, error) {
	selector := labels.Everything()
	nodes, err := c.nodes.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list nodes: %w", err)
	}
	namespaces, err := c.namespaces.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list namespaces: %w", err)
	}
	pods, err := c.pods.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list pods: %w", err)
	}
	services, err := c.services.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list services: %w", err)
	}
	events, err := c.events.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list events: %w", err)
	}
	persistentVolumes, err := c.persistentVolumes.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list persistent volumes: %w", err)
	}
	persistentVolumeClaims, err := c.persistentVolumeClaims.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list persistent volume claims: %w", err)
	}
	deployments, err := c.deployments.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list deployments: %w", err)
	}
	statefulSets, err := c.statefulSets.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list stateful sets: %w", err)
	}
	daemonSets, err := c.daemonSets.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list daemon sets: %w", err)
	}
	ingresses, err := c.ingresses.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list ingresses: %w", err)
	}
	endpointSlices, err := c.endpointSlices.Lister().List(selector)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("list endpoint slices: %w", err)
	}

	now := time.Now().UTC()
	nodeMetrics, podMetrics := c.metricsSnapshot()
	podsByNode := indexPodsByNode(pods)
	podsByNamespace := indexPodsByNamespace(pods)
	endpointsByService := indexEndpointSlices(endpointSlices)
	ingressModels, ingressURLs := buildIngresses(ingresses)

	nodeModels := buildNodes(now, nodes, podsByNode, nodeMetrics)
	podModels := buildPods(pods, podMetrics)
	workloads := buildWorkloads(deployments, statefulSets, daemonSets, podsByNamespace)
	serviceModels := buildServices(
		now,
		services,
		podsByNamespace,
		endpointsByService,
		ingressURLs,
		c.config.ClusterDomain,
	)
	volumes := buildVolumes(persistentVolumes, persistentVolumeClaims)
	eventModels := buildEvents(events, c.config.EventLimit)
	dns := buildDNSProfile(services, endpointsByService, c.config.ClusterDomain)

	summary := model.Summary{
		Nodes:             len(nodeModels),
		Namespaces:        len(namespaces),
		Pods:              len(podModels),
		Workloads:         len(workloads),
		Services:          len(serviceModels),
		Ingresses:         len(ingressModels),
		PersistentVolumes: len(volumes),
		WarningEvents:     countWarningEvents(eventModels),
	}
	for _, node := range nodeModels {
		if node.Ready {
			summary.ReadyNodes++
		}
	}
	for _, pod := range podModels {
		if pod.Ready {
			summary.ReadyPods++
		}
	}
	for _, workload := range workloads {
		if workload.Status == "ready" {
			summary.ReadyWorkloads++
		}
	}
	for _, service := range serviceModels {
		if service.Status == "ready" {
			summary.ReadyServices++
		}
	}

	return model.Snapshot{
		SchemaVersion: model.SchemaVersion,
		GeneratedAt:   now,
		Cluster: model.Cluster{
			ID:                c.config.ClusterID,
			Name:              c.config.ClusterName,
			KubernetesVersion: c.serverVersion,
			Platform:          c.platform,
		},
		Summary:   summary,
		DNS:       dns,
		Nodes:     nodeModels,
		Pods:      podModels,
		Workloads: workloads,
		Services:  serviceModels,
		Ingresses: ingressModels,
		Volumes:   volumes,
		Events:    eventModels,
	}, nil
}

func (c *Collector) metricsSnapshot() (
	map[string]metricsv1beta1.NodeMetrics,
	map[string]metricsv1beta1.PodMetrics,
) {
	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()

	nodes := make(map[string]metricsv1beta1.NodeMetrics, len(c.nodeMetrics))
	for key, value := range c.nodeMetrics {
		nodes[key] = value
	}
	pods := make(map[string]metricsv1beta1.PodMetrics, len(c.podMetrics))
	for key, value := range c.podMetrics {
		pods[key] = value
	}
	return nodes, pods
}

func buildNodes(
	now time.Time,
	nodes []*corev1.Node,
	podsByNode map[string][]*corev1.Pod,
	metrics map[string]metricsv1beta1.NodeMetrics,
) []model.Node {
	result := make([]model.Node, 0, len(nodes))
	for _, node := range nodes {
		ready, status, heartbeat := nodeStatus(node)
		nodePods := activePods(podsByNode[node.Name])
		capacity := model.NodeCapacity{
			CPUMilli:              node.Status.Allocatable.Cpu().MilliValue(),
			MemoryBytes:           node.Status.Allocatable.Memory().Value(),
			EphemeralStorageBytes: node.Status.Allocatable.StorageEphemeral().Value(),
			Pods:                  node.Status.Allocatable.Pods().Value(),
		}
		usage := model.NodeUsage{
			Pods:                           len(nodePods),
			PodAllocationPercent:           percentage(int64(len(nodePods)), capacity.Pods),
			EphemeralStorageRequestedBytes: ephemeralStorageRequests(nodePods),
		}
		usage.EphemeralStorageRequestPercent = percentage(
			usage.EphemeralStorageRequestedBytes,
			capacity.EphemeralStorageBytes,
		)
		if nodeMetric, ok := metrics[node.Name]; ok {
			usage.MetricsAvailable = true
			usage.CPUMilli = nodeMetric.Usage.Cpu().MilliValue()
			usage.MemoryBytes = nodeMetric.Usage.Memory().Value()
			usage.CPUPercent = percentage(usage.CPUMilli, capacity.CPUMilli)
			usage.MemoryPercent = percentage(usage.MemoryBytes, capacity.MemoryBytes)
		}

		addresses := map[corev1.NodeAddressType]string{}
		for _, address := range node.Status.Addresses {
			addresses[address.Type] = address.Address
		}
		createdAt := node.CreationTimestamp.Time.UTC()
		result = append(result, model.Node{
			Name:             node.Name,
			Role:             nodeRole(node.Labels),
			Ready:            ready,
			Status:           status,
			InternalIP:       addresses[corev1.NodeInternalIP],
			Hostname:         addresses[corev1.NodeHostName],
			KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
			OSImage:          node.Status.NodeInfo.OSImage,
			OperatingSystem:  node.Status.NodeInfo.OperatingSystem,
			Architecture:     node.Status.NodeInfo.Architecture,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			CreatedAt:        createdAt,
			UptimeSeconds:    max(0, int64(now.Sub(createdAt).Seconds())),
			LastHeartbeatAt:  heartbeat,
			Capacity:         capacity,
			Usage:            usage,
			Conditions:       nodeConditions(node.Status.Conditions),
			Unschedulable:    node.Spec.Unschedulable,
			PodCIDR:          node.Spec.PodCIDR,
			Labels:           cloneMap(node.Labels),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func buildPods(
	pods []*corev1.Pod,
	metrics map[string]metricsv1beta1.PodMetrics,
) []model.Pod {
	result := make([]model.Pod, 0, len(pods))
	for _, pod := range pods {
		ready := podReady(pod)
		status := "attention"
		if ready {
			status = "ready"
		} else if pod.Status.Phase == corev1.PodSucceeded {
			status = "completed"
		} else if pod.Status.Phase == corev1.PodFailed {
			status = "failed"
		} else if pod.Status.Phase == corev1.PodPending {
			status = "pending"
		}

		containerStatus := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
		var restarts int32
		for _, item := range pod.Status.ContainerStatuses {
			containerStatus[item.Name] = item
			restarts += item.RestartCount
		}
		containers := make([]model.Container, 0, len(pod.Spec.Containers))
		var startedAt *time.Time
		for _, container := range pod.Spec.Containers {
			statusItem := containerStatus[container.Name]
			state, containerStartedAt := describeContainerState(statusItem.State)
			if containerStartedAt != nil && (startedAt == nil || containerStartedAt.Before(*startedAt)) {
				startedAt = containerStartedAt
			}
			containers = append(containers, model.Container{
				Name:         container.Name,
				Image:        container.Image,
				Ready:        statusItem.Ready,
				RestartCount: statusItem.RestartCount,
				State:        state,
				StartedAt:    containerStartedAt,
			})
		}

		usage := model.PodUsage{}
		if metric, ok := metrics[pod.Namespace+"/"+pod.Name]; ok {
			usage.MetricsAvailable = true
			for _, container := range metric.Containers {
				usage.CPUMilli += container.Usage.Cpu().MilliValue()
				usage.MemoryBytes += container.Usage.Memory().Value()
			}
		}

		ownerKind, ownerName := controllerOwner(pod.OwnerReferences)
		result = append(result, model.Pod{
			UID:        string(pod.UID),
			Namespace:  pod.Namespace,
			Name:       pod.Name,
			NodeName:   pod.Spec.NodeName,
			Phase:      string(pod.Status.Phase),
			Status:     status,
			Reason:     firstNonEmpty(pod.Status.Reason, waitingReason(pod.Status.ContainerStatuses)),
			Ready:      ready,
			Restarts:   restarts,
			CreatedAt:  pod.CreationTimestamp.Time.UTC(),
			StartedAt:  startedAt,
			OwnerKind:  ownerKind,
			OwnerName:  ownerName,
			PodIP:      pod.Status.PodIP,
			HostIP:     pod.Status.HostIP,
			Containers: containers,
			Usage:      usage,
			Labels:     cloneMap(pod.Labels),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace == result[j].Namespace {
			return result[i].Name < result[j].Name
		}
		return result[i].Namespace < result[j].Namespace
	})
	return result
}

func buildWorkloads(
	deployments []*appsv1.Deployment,
	statefulSets []*appsv1.StatefulSet,
	daemonSets []*appsv1.DaemonSet,
	podsByNamespace map[string][]*corev1.Pod,
) []model.Workload {
	result := make([]model.Workload, 0, len(deployments)+len(statefulSets)+len(daemonSets))
	for _, deployment := range deployments {
		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}
		lastDeploy := latestMatchingPod(
			podsByNamespace[deployment.Namespace],
			deployment.Spec.Selector,
		)
		result = append(result, model.Workload{
			UID:            string(deployment.UID),
			Kind:           "Deployment",
			Namespace:      deployment.Namespace,
			Name:           deployment.Name,
			Desired:        desired,
			Ready:          deployment.Status.ReadyReplicas,
			Available:      deployment.Status.AvailableReplicas,
			Status:         workloadStatus(desired, deployment.Status.ReadyReplicas),
			CreatedAt:      deployment.CreationTimestamp.Time.UTC(),
			LastDeployedAt: lastDeploy,
			Labels:         cloneMap(deployment.Labels),
		})
	}
	for _, statefulSet := range statefulSets {
		desired := int32(1)
		if statefulSet.Spec.Replicas != nil {
			desired = *statefulSet.Spec.Replicas
		}
		lastDeploy := latestMatchingPod(
			podsByNamespace[statefulSet.Namespace],
			statefulSet.Spec.Selector,
		)
		result = append(result, model.Workload{
			UID:            string(statefulSet.UID),
			Kind:           "StatefulSet",
			Namespace:      statefulSet.Namespace,
			Name:           statefulSet.Name,
			Desired:        desired,
			Ready:          statefulSet.Status.ReadyReplicas,
			Available:      statefulSet.Status.AvailableReplicas,
			Status:         workloadStatus(desired, statefulSet.Status.ReadyReplicas),
			CreatedAt:      statefulSet.CreationTimestamp.Time.UTC(),
			LastDeployedAt: lastDeploy,
			Labels:         cloneMap(statefulSet.Labels),
		})
	}
	for _, daemonSet := range daemonSets {
		desired := daemonSet.Status.DesiredNumberScheduled
		lastDeploy := latestMatchingPod(
			podsByNamespace[daemonSet.Namespace],
			daemonSet.Spec.Selector,
		)
		result = append(result, model.Workload{
			UID:            string(daemonSet.UID),
			Kind:           "DaemonSet",
			Namespace:      daemonSet.Namespace,
			Name:           daemonSet.Name,
			Desired:        desired,
			Ready:          daemonSet.Status.NumberReady,
			Available:      daemonSet.Status.NumberAvailable,
			Status:         workloadStatus(desired, daemonSet.Status.NumberReady),
			CreatedAt:      daemonSet.CreationTimestamp.Time.UTC(),
			LastDeployedAt: lastDeploy,
			Labels:         cloneMap(daemonSet.Labels),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace == result[j].Namespace {
			if result[i].Kind == result[j].Kind {
				return result[i].Name < result[j].Name
			}
			return result[i].Kind < result[j].Kind
		}
		return result[i].Namespace < result[j].Namespace
	})
	return result
}

func buildServices(
	now time.Time,
	services []*corev1.Service,
	podsByNamespace map[string][]*corev1.Pod,
	endpoints map[string]endpointCounts,
	ingressURLs map[string][]string,
	clusterDomain string,
) []model.Service {
	result := make([]model.Service, 0, len(services))
	for _, service := range services {
		key := service.Namespace + "/" + service.Name
		matchedPods := matchingPods(podsByNamespace[service.Namespace], service.Spec.Selector)
		readyPods := 0
		var (
			lastDeploy *time.Time
			oldestRun  *time.Time
		)
		for _, pod := range matchedPods {
			if podReady(pod) {
				readyPods++
			}
			createdAt := pod.CreationTimestamp.Time.UTC()
			if lastDeploy == nil || createdAt.After(*lastDeploy) {
				lastDeploy = timePtr(createdAt)
			}
			if started := podStartedAt(pod); started != nil &&
				(oldestRun == nil || started.Before(*oldestRun)) {
				oldestRun = started
			}
		}

		counts := endpoints[key]
		status := "offline"
		if counts.ready > 0 || (counts.total == 0 && readyPods > 0) {
			status = "ready"
		} else if counts.total > 0 || len(matchedPods) > 0 {
			status = "attention"
		} else if service.Spec.Type == corev1.ServiceTypeExternalName {
			status = "unknown"
		}

		externalIPs := append([]string(nil), service.Spec.ExternalIPs...)
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				externalIPs = append(externalIPs, ingress.IP)
			}
			if ingress.Hostname != "" {
				externalIPs = append(externalIPs, ingress.Hostname)
			}
		}
		externalIPs = uniqueStrings(externalIPs)
		urls := uniqueStrings(ingressURLs[key])
		category := ClassifyService(
			service.Name,
			service.Labels,
			service.Annotations,
			service.Spec.Ports,
			len(urls) > 0,
		)

		uptimeSeconds := int64(0)
		if oldestRun != nil {
			uptimeSeconds = max(0, int64(now.Sub(*oldestRun).Seconds()))
		}
		result = append(result, model.Service{
			UID:            string(service.UID),
			Namespace:      service.Namespace,
			Name:           service.Name,
			Category:       category,
			Type:           string(service.Spec.Type),
			Status:         status,
			ClusterDNS:     fmt.Sprintf("%s.%s.svc.%s", service.Name, service.Namespace, clusterDomain),
			ClusterIP:      normalizedClusterIP(service.Spec.ClusterIP),
			ExternalIPs:    externalIPs,
			ExternalURLs:   urls,
			Ports:          servicePorts(service.Spec.Ports),
			Selector:       cloneMap(service.Spec.Selector),
			ReadyEndpoints: counts.ready,
			TotalEndpoints: counts.total,
			ReadyPods:      readyPods,
			TotalPods:      len(matchedPods),
			LastDeployedAt: lastDeploy,
			UptimeSeconds:  uptimeSeconds,
			Labels:         cloneMap(service.Labels),
			Annotations:    cloneMap(service.Annotations),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category == result[j].Category {
			if result[i].Namespace == result[j].Namespace {
				return result[i].Name < result[j].Name
			}
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Category < result[j].Category
	})
	return result
}

func buildIngresses(ingresses []*networkingv1.Ingress) (
	[]model.Ingress,
	map[string][]string,
) {
	result := make([]model.Ingress, 0, len(ingresses))
	serviceURLs := make(map[string][]string)
	for _, ingress := range ingresses {
		tlsHosts := make(map[string]struct{})
		tlsList := make([]string, 0)
		for _, tls := range ingress.Spec.TLS {
			for _, host := range tls.Hosts {
				tlsHosts[host] = struct{}{}
				tlsList = append(tlsList, host)
			}
		}

		routes := make([]model.IngressRoute, 0)
		if ingress.Spec.DefaultBackend != nil {
			route := ingressRoute(
				"",
				"/",
				"",
				*ingress.Spec.DefaultBackend,
				tlsHosts,
			)
			routes = append(routes, route)
			if route.URL != "" {
				key := ingress.Namespace + "/" + route.ServiceName
				serviceURLs[key] = append(serviceURLs[key], route.URL)
			}
		}
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				pathType := ""
				if path.PathType != nil {
					pathType = string(*path.PathType)
				}
				route := ingressRoute(
					rule.Host,
					firstNonEmpty(path.Path, "/"),
					pathType,
					path.Backend,
					tlsHosts,
				)
				routes = append(routes, route)
				if route.URL != "" {
					key := ingress.Namespace + "/" + route.ServiceName
					serviceURLs[key] = append(serviceURLs[key], route.URL)
				}
			}
		}

		addresses := make([]string, 0, len(ingress.Status.LoadBalancer.Ingress))
		for _, address := range ingress.Status.LoadBalancer.Ingress {
			addresses = append(addresses, firstNonEmpty(address.IP, address.Hostname))
		}
		className := ""
		if ingress.Spec.IngressClassName != nil {
			className = *ingress.Spec.IngressClassName
		}
		result = append(result, model.Ingress{
			UID:       string(ingress.UID),
			Namespace: ingress.Namespace,
			Name:      ingress.Name,
			ClassName: className,
			Routes:    routes,
			TLSHosts:  uniqueStrings(tlsList),
			Addresses: uniqueStrings(addresses),
			CreatedAt: ingress.CreationTimestamp.Time.UTC(),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace == result[j].Namespace {
			return result[i].Name < result[j].Name
		}
		return result[i].Namespace < result[j].Namespace
	})
	return result, serviceURLs
}

func buildVolumes(
	persistentVolumes []*corev1.PersistentVolume,
	claims []*corev1.PersistentVolumeClaim,
) []model.Volume {
	result := make([]model.Volume, 0, len(persistentVolumes)+len(claims))
	for _, volume := range persistentVolumes {
		result = append(result, model.Volume{
			UID:           string(volume.UID),
			Name:          volume.Name,
			Kind:          "PersistentVolume",
			Status:        string(volume.Status.Phase),
			StorageClass:  volume.Spec.StorageClassName,
			CapacityBytes: volume.Spec.Capacity.Storage().Value(),
			AccessModes:   accessModes(volume.Spec.AccessModes),
			CreatedAt:     volume.CreationTimestamp.Time.UTC(),
			Labels:        cloneMap(volume.Labels),
		})
	}
	for _, claim := range claims {
		capacity := claim.Status.Capacity.Storage().Value()
		if capacity == 0 {
			capacity = claim.Spec.Resources.Requests.Storage().Value()
		}
		storageClass := ""
		if claim.Spec.StorageClassName != nil {
			storageClass = *claim.Spec.StorageClassName
		}
		result = append(result, model.Volume{
			UID:           string(claim.UID),
			Namespace:     claim.Namespace,
			Name:          claim.Name,
			Kind:          "PersistentVolumeClaim",
			Status:        string(claim.Status.Phase),
			StorageClass:  storageClass,
			CapacityBytes: capacity,
			AccessModes:   accessModes(claim.Spec.AccessModes),
			VolumeName:    claim.Spec.VolumeName,
			CreatedAt:     claim.CreationTimestamp.Time.UTC(),
			Labels:        cloneMap(claim.Labels),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind == result[j].Kind {
			if result[i].Namespace == result[j].Namespace {
				return result[i].Name < result[j].Name
			}
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Kind < result[j].Kind
	})
	return result
}

func buildEvents(events []*corev1.Event, limit int) []model.KubernetesEvent {
	result := make([]model.KubernetesEvent, 0, len(events))
	for _, event := range events {
		count := event.Count
		if event.Series != nil {
			count = event.Series.Count
		}
		result = append(result, model.KubernetesEvent{
			UID:                string(event.UID),
			Namespace:          event.Namespace,
			Name:               event.Name,
			Type:               event.Type,
			Reason:             event.Reason,
			Message:            event.Message,
			RegardingKind:      event.InvolvedObject.Kind,
			RegardingNamespace: event.InvolvedObject.Namespace,
			RegardingName:      event.InvolvedObject.Name,
			Count:              count,
			FirstSeenAt:        eventFirstSeen(event),
			LastSeenAt:         eventLastSeen(event),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		left := eventSortTime(result[i])
		right := eventSortTime(result[j])
		return left.After(right)
	})
	if limit >= 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

func buildDNSProfile(
	services []*corev1.Service,
	endpoints map[string]endpointCounts,
	clusterDomain string,
) model.DNSProfile {
	profile := model.DNSProfile{
		Provider:      "Kubernetes DNS",
		ClusterDomain: clusterDomain,
		SearchPath: []string{
			"<namespace>.svc." + clusterDomain,
			"svc." + clusterDomain,
			clusterDomain,
		},
	}

	var selected *corev1.Service
	for _, service := range services {
		if service.Namespace != metav1.NamespaceSystem {
			continue
		}
		if service.Name == "kube-dns" ||
			service.Name == "coredns" ||
			service.Labels["k8s-app"] == "kube-dns" ||
			service.Labels["app.kubernetes.io/name"] == "coredns" {
			selected = service
			if service.Name == "kube-dns" {
				break
			}
		}
	}
	if selected == nil {
		return profile
	}

	profile.Provider = "CoreDNS"
	profile.ServiceName = selected.Namespace + "/" + selected.Name
	profile.ServiceDNS = fmt.Sprintf(
		"%s.%s.svc.%s",
		selected.Name,
		selected.Namespace,
		clusterDomain,
	)
	profile.ServiceIP = normalizedClusterIP(selected.Spec.ClusterIP)
	profile.Ports = servicePorts(selected.Spec.Ports)
	profile.Ready = endpoints[selected.Namespace+"/"+selected.Name].ready > 0
	return profile
}

func indexPodsByNode(pods []*corev1.Pod) map[string][]*corev1.Pod {
	result := make(map[string][]*corev1.Pod)
	for _, pod := range pods {
		result[pod.Spec.NodeName] = append(result[pod.Spec.NodeName], pod)
	}
	return result
}

func indexPodsByNamespace(pods []*corev1.Pod) map[string][]*corev1.Pod {
	result := make(map[string][]*corev1.Pod)
	for _, pod := range pods {
		result[pod.Namespace] = append(result[pod.Namespace], pod)
	}
	return result
}

func indexEndpointSlices(slices []*discoveryv1.EndpointSlice) map[string]endpointCounts {
	result := make(map[string]endpointCounts)
	for _, slice := range slices {
		serviceName := slice.Labels[discoveryv1.LabelServiceName]
		if serviceName == "" {
			continue
		}
		key := slice.Namespace + "/" + serviceName
		counts := result[key]
		for _, endpoint := range slice.Endpoints {
			addressCount := len(endpoint.Addresses)
			counts.total += addressCount
			ready := endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready
			terminating := endpoint.Conditions.Terminating != nil && *endpoint.Conditions.Terminating
			if ready && !terminating {
				counts.ready += addressCount
			}
		}
		result[key] = counts
	}
	return result
}

func activePods(pods []*corev1.Pod) []*corev1.Pod {
	result := make([]*corev1.Pod, 0, len(pods))
	for _, pod := range pods {
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			result = append(result, pod)
		}
	}
	return result
}

func matchingPods(pods []*corev1.Pod, selectorMap map[string]string) []*corev1.Pod {
	if len(selectorMap) == 0 {
		return nil
	}
	selector := labels.SelectorFromSet(selectorMap)
	result := make([]*corev1.Pod, 0)
	for _, pod := range pods {
		if selector.Matches(labels.Set(pod.Labels)) {
			result = append(result, pod)
		}
	}
	return result
}

func latestMatchingPod(pods []*corev1.Pod, selector *metav1.LabelSelector) *time.Time {
	parsed, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil
	}
	var latest *time.Time
	for _, pod := range pods {
		if !parsed.Matches(labels.Set(pod.Labels)) {
			continue
		}
		createdAt := pod.CreationTimestamp.Time.UTC()
		if latest == nil || createdAt.After(*latest) {
			latest = timePtr(createdAt)
		}
	}
	return latest
}

func podReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func podStartedAt(pod *corev1.Pod) *time.Time {
	var earliest *time.Time
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Running == nil {
			continue
		}
		startedAt := status.State.Running.StartedAt.Time.UTC()
		if earliest == nil || startedAt.Before(*earliest) {
			earliest = timePtr(startedAt)
		}
	}
	if earliest == nil && !pod.Status.StartTime.IsZero() {
		return timePtr(pod.Status.StartTime.Time.UTC())
	}
	return earliest
}

func describeContainerState(state corev1.ContainerState) (string, *time.Time) {
	switch {
	case state.Running != nil:
		return "running", timePtr(state.Running.StartedAt.Time.UTC())
	case state.Waiting != nil:
		return "waiting", nil
	case state.Terminated != nil:
		return "terminated", timePtr(state.Terminated.StartedAt.Time.UTC())
	default:
		return "unknown", nil
	}
}

func waitingReason(statuses []corev1.ContainerStatus) string {
	for _, status := range statuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			return status.State.Waiting.Reason
		}
		if status.LastTerminationState.Terminated != nil &&
			status.LastTerminationState.Terminated.Reason != "" {
			return status.LastTerminationState.Terminated.Reason
		}
	}
	return ""
}

func controllerOwner(owners []metav1.OwnerReference) (string, string) {
	for _, owner := range owners {
		if owner.Controller != nil && *owner.Controller {
			return owner.Kind, owner.Name
		}
	}
	if len(owners) > 0 {
		return owners[0].Kind, owners[0].Name
	}
	return "", ""
}

func workloadStatus(desired, ready int32) string {
	if desired == 0 {
		return "scaled-down"
	}
	if ready >= desired {
		return "ready"
	}
	if ready == 0 {
		return "offline"
	}
	return "attention"
}

func nodeStatus(node *corev1.Node) (bool, string, *time.Time) {
	status := "Unknown"
	var heartbeat *time.Time
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if !condition.LastHeartbeatTime.IsZero() {
				heartbeat = timePtr(condition.LastHeartbeatTime.Time.UTC())
			}
			if condition.Status == corev1.ConditionTrue {
				return true, "Ready", heartbeat
			}
			status = firstNonEmpty(condition.Reason, string(condition.Status))
		}
	}
	for _, condition := range node.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && condition.Type != corev1.NodeReady {
			return false, firstNonEmpty(condition.Reason, string(condition.Type)), heartbeat
		}
	}
	return false, status, heartbeat
}

func nodeConditions(conditions []corev1.NodeCondition) []model.NodeCondition {
	result := make([]model.NodeCondition, 0, len(conditions))
	for _, condition := range conditions {
		var transition *time.Time
		if !condition.LastTransitionTime.IsZero() {
			transition = timePtr(condition.LastTransitionTime.Time.UTC())
		}
		result = append(result, model.NodeCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: transition,
		})
	}
	return result
}

func nodeRole(nodeLabels map[string]string) string {
	for _, role := range []string{"control-plane", "master"} {
		if _, ok := nodeLabels["node-role.kubernetes.io/"+role]; ok {
			return "control-plane"
		}
	}
	roles := make([]string, 0)
	for key := range nodeLabels {
		const prefix = "node-role.kubernetes.io/"
		if strings.HasPrefix(key, prefix) {
			role := strings.TrimPrefix(key, prefix)
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	sort.Strings(roles)
	if len(roles) > 0 {
		return strings.Join(roles, ",")
	}
	return "worker"
}

func ephemeralStorageRequests(pods []*corev1.Pod) int64 {
	var total int64
	for _, pod := range pods {
		var regular int64
		for _, container := range pod.Spec.Containers {
			regular += container.Resources.Requests.StorageEphemeral().Value()
		}
		var initMaximum int64
		for _, container := range pod.Spec.InitContainers {
			initMaximum = max(
				initMaximum,
				container.Resources.Requests.StorageEphemeral().Value(),
			)
		}
		request := max(regular, initMaximum)
		if pod.Spec.Overhead != nil {
			request += pod.Spec.Overhead.StorageEphemeral().Value()
		}
		total += request
	}
	return total
}

func percentage(value, capacity int64) float64 {
	if value <= 0 || capacity <= 0 {
		return 0
	}
	calculated := float64(value) / float64(capacity) * 100
	return math.Round(min(calculated, 100)*100) / 100
}

func servicePorts(ports []corev1.ServicePort) []model.Port {
	result := make([]model.Port, 0, len(ports))
	for _, port := range ports {
		result = append(result, model.Port{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: port.TargetPort.String(),
			NodePort:   port.NodePort,
		})
	}
	return result
}

func ingressRoute(
	host, path, pathType string,
	backend networkingv1.IngressBackend,
	tlsHosts map[string]struct{},
) model.IngressRoute {
	route := model.IngressRoute{
		Host:        host,
		Path:        path,
		PathType:    pathType,
		ServiceName: backend.Service.Name,
		ServicePort: ingressServicePort(backend.Service.Port),
	}
	if host != "" {
		scheme := "http"
		if _, tls := tlsHosts[host]; tls {
			scheme = "https"
		}
		route.URL = scheme + "://" + host + path
	}
	return route
}

func ingressServicePort(port networkingv1.ServiceBackendPort) string {
	if port.Name != "" {
		return port.Name
	}
	return fmt.Sprintf("%d", port.Number)
}

func normalizedClusterIP(clusterIP string) string {
	if clusterIP == corev1.ClusterIPNone {
		return ""
	}
	return clusterIP
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	result := make([]string, 0, len(modes))
	for _, mode := range modes {
		result = append(result, string(mode))
	}
	return result
}

func eventFirstSeen(event *corev1.Event) *time.Time {
	if !event.FirstTimestamp.IsZero() {
		return timePtr(event.FirstTimestamp.Time.UTC())
	}
	if !event.CreationTimestamp.IsZero() {
		return timePtr(event.CreationTimestamp.Time.UTC())
	}
	return nil
}

func eventLastSeen(event *corev1.Event) *time.Time {
	if event.Series != nil && !event.Series.LastObservedTime.IsZero() {
		return timePtr(event.Series.LastObservedTime.Time.UTC())
	}
	if !event.EventTime.IsZero() {
		return timePtr(event.EventTime.Time.UTC())
	}
	if !event.LastTimestamp.IsZero() {
		return timePtr(event.LastTimestamp.Time.UTC())
	}
	return eventFirstSeen(event)
}

func eventSortTime(event model.KubernetesEvent) time.Time {
	if event.LastSeenAt != nil {
		return *event.LastSeenAt
	}
	if event.FirstSeenAt != nil {
		return *event.FirstSeenAt
	}
	return time.Time{}
}

func countWarningEvents(events []model.KubernetesEvent) int {
	count := 0
	for _, event := range events {
		if strings.EqualFold(event.Type, corev1.EventTypeWarning) {
			count++
		}
	}
	return count
}

func cloneMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func timePtr(value time.Time) *time.Time {
	return &value
}
