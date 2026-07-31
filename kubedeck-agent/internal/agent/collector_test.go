package agent

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/amirtaherkhani/kubedeck-agent/internal/config"
	"github.com/amirtaherkhani/kubedeck-agent/internal/model"
	"github.com/amirtaherkhani/kubedeck-agent/internal/stream"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
)

func TestCollectorBuildsDashboardSnapshotFromInformerCaches(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Add(-time.Minute)
	objects := []runtime.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "apps"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "worker-01",
				CreationTimestamp: metav1.NewTime(now.Add(-24 * time.Hour)),
				Labels:            map[string]string{"node-role.kubernetes.io/worker": ""},
			},
			Status: corev1.NodeStatus{
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("2"),
					corev1.ResourceMemory:           resource.MustParse("4Gi"),
					corev1.ResourcePods:             resource.MustParse("20"),
					corev1.ResourceEphemeralStorage: resource.MustParse("100Gi"),
				},
				Conditions: []corev1.NodeCondition{{
					Type:              corev1.NodeReady,
					Status:            corev1.ConditionTrue,
					LastHeartbeatTime: metav1.NewTime(now),
				}},
				NodeInfo: corev1.NodeSystemInfo{
					KubeletVersion:          "v1.36.1",
					OperatingSystem:         "linux",
					Architecture:            "arm64",
					ContainerRuntimeVersion: "containerd://2",
				},
				Addresses: []corev1.NodeAddress{{
					Type:    corev1.NodeInternalIP,
					Address: "10.0.0.21",
				}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "apps",
				Name:              "postgresql-0",
				UID:               types.UID("pod-1"),
				CreationTimestamp: metav1.NewTime(now),
				Labels:            map[string]string{"app": "postgresql"},
			},
			Spec: corev1.PodSpec{
				NodeName: "worker-01",
				Containers: []corev1.Container{{
					Name:  "postgresql",
					Image: "postgres:18",
				}},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				Conditions: []corev1.PodCondition{{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				}},
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "postgresql",
					Ready: true,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(now)},
					},
				}},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "apps",
				Name:      "postgresql",
				UID:       types.UID("service-1"),
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.43.10.20",
				Selector:  map[string]string{"app": "postgresql"},
				Ports: []corev1.ServicePort{{
					Name: "postgresql",
					Port: 5432,
				}},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kube-dns",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.43.0.10",
				Ports: []corev1.ServicePort{{
					Name:     "dns",
					Port:     53,
					Protocol: corev1.ProtocolUDP,
				}},
			},
		},
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "apps",
				Name:      "postgresql-abc",
				Labels: map[string]string{
					discoveryv1.LabelServiceName: "postgresql",
				},
			},
			AddressType: discoveryv1.AddressTypeIPv4,
			Endpoints: []discoveryv1.Endpoint{{
				Addresses: []string{"10.42.0.9"},
				Conditions: discoveryv1.EndpointConditions{
					Ready: boolPtr(true),
				},
			}},
		},
		&discoveryv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kube-dns-abc",
				Labels: map[string]string{
					discoveryv1.LabelServiceName: "kube-dns",
				},
			},
			AddressType: discoveryv1.AddressTypeIPv4,
			Endpoints: []discoveryv1.Endpoint{{
				Addresses: []string{"10.42.0.10"},
				Conditions: discoveryv1.EndpointConditions{
					Ready: boolPtr(true),
				},
			}},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "apps",
				Name:              "postgresql",
				UID:               types.UID("deployment-1"),
				CreationTimestamp: metav1.NewTime(now.Add(-time.Hour)),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "postgresql"}},
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		},
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "apps",
				Name:              "state-store",
				UID:               types.UID("statefulset-1"),
				CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "state-store"}},
			},
			Status: appsv1.StatefulSetStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 1,
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "apps",
				Name:              "node-agent",
				UID:               types.UID("daemonset-1"),
				CreationTimestamp: metav1.NewTime(now.Add(-3 * time.Hour)),
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "node-agent"}},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 1,
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "apps",
				Name:              "postgresql-admin",
				UID:               types.UID("ingress-1"),
				CreationTimestamp: metav1.NewTime(now),
			},
			Spec: networkingv1.IngressSpec{
				TLS: []networkingv1.IngressTLS{{Hosts: []string{"database.example.test"}}},
				Rules: []networkingv1.IngressRule{{
					Host: "database.example.test",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Path:     "/admin",
								PathType: pathTypePtr(networkingv1.PathTypePrefix),
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "postgresql",
										Port: networkingv1.ServiceBackendPort{Number: 5432},
									},
								},
							}},
						},
					},
				}},
			},
		},
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "data-volume",
				UID:               types.UID("pv-1"),
				CreationTimestamp: metav1.NewTime(now.Add(-24 * time.Hour)),
			},
			Spec: corev1.PersistentVolumeSpec{
				Capacity:         corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				StorageClassName: "local-path",
			},
			Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeBound},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "apps",
				Name:              "postgresql-data",
				UID:               types.UID("pvc-1"),
				CreationTimestamp: metav1.NewTime(now.Add(-23 * time.Hour)),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
				},
				StorageClassName: stringPtr("local-path"),
				VolumeName:       "data-volume",
			},
			Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Namespace: "apps", Name: "pod-warning", UID: types.UID("event-1")},
			Type:           corev1.EventTypeWarning,
			Reason:         "BackOff",
			Message:        "Container restarted",
			Count:          2,
			LastTimestamp:  metav1.NewTime(now),
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "apps", Name: "postgresql-0"},
		},
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Namespace: "apps", Name: "pod-normal", UID: types.UID("event-2")},
			Type:           corev1.EventTypeNormal,
			Reason:         "Pulled",
			Message:        "Image pulled",
			Count:          1,
			LastTimestamp:  metav1.NewTime(now.Add(-time.Minute)),
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "apps", Name: "postgresql-0"},
		},
	}
	kube := kubernetesfake.NewSimpleClientset(objects...)
	nodeMetric := metricsv1beta1.NodeMetrics{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metricsv1beta1.SchemeGroupVersion.String(),
			Kind:       "NodeMetrics",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "worker-01"},
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}
	podMetric := metricsv1beta1.PodMetrics{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metricsv1beta1.SchemeGroupVersion.String(),
			Kind:       "PodMetrics",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "apps", Name: "postgresql-0"},
		Containers: []metricsv1beta1.ContainerMetrics{{
			Name: "postgresql",
			Usage: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		}},
	}
	metrics := metricsfake.NewSimpleClientset()
	metrics.PrependReactor("list", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &metricsv1beta1.NodeMetricsList{Items: []metricsv1beta1.NodeMetrics{nodeMetric}}, nil
	})
	metrics.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &metricsv1beta1.PodMetricsList{Items: []metricsv1beta1.PodMetrics{podMetric}}, nil
	})
	nodeMetricList, err := metrics.MetricsV1beta1().NodeMetricses().List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil || len(nodeMetricList.Items) != 1 {
		t.Fatalf("seed node metrics: count=%d error=%v", len(nodeMetricList.Items), err)
	}
	cfg := config.Config{
		ClusterID:       "test",
		ClusterName:     "Test cluster",
		ClusterDomain:   "cluster.local",
		MetricsInterval: time.Hour,
		RefreshDebounce: 10 * time.Millisecond,
		EventLimit:      20,
	}
	broker := stream.NewBroker("test", 8)
	collector, err := NewCollector(
		cfg,
		kube,
		metrics,
		broker,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runResult := make(chan error, 1)
	go func() { runResult <- collector.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-runResult:
			if err != nil {
				t.Errorf("collector Run() = %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("collector did not stop")
		}
	})

	deadline := time.Now().Add(5 * time.Second)
	for !collector.Ready() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !collector.Ready() {
		t.Fatal("collector did not become ready")
	}

	snapshot := collector.Snapshot()
	if snapshot.Summary.Nodes != 1 || snapshot.Summary.ReadyNodes != 1 {
		t.Fatalf("unexpected node summary: %#v", snapshot.Summary)
	}
	if len(snapshot.Nodes) != 1 || snapshot.Nodes[0].Usage.CPUPercent != 25 {
		t.Fatalf("unexpected node metrics: %#v", snapshot.Nodes)
	}
	if len(snapshot.Pods) != 1 || !snapshot.Pods[0].Usage.MetricsAvailable {
		t.Fatalf("unexpected pod snapshot: %#v", snapshot.Pods)
	}
	if len(snapshot.Services) != 2 {
		t.Fatalf("service count = %d", len(snapshot.Services))
	}
	var databaseFound bool
	for _, service := range snapshot.Services {
		if service.Name == "postgresql" {
			databaseFound = true
			if service.Category != CategoryDatabasesStorage ||
				service.Status != "ready" ||
				service.ClusterDNS != "postgresql.apps.svc.cluster.local" ||
				service.LastDeployedAt == nil {
				t.Fatalf("unexpected database service: %#v", service)
			}
		}
	}
	if !databaseFound {
		t.Fatal("postgresql service was not discovered")
	}
	if snapshot.DNS.ServiceIP != "10.43.0.10" {
		t.Fatalf("DNS profile = %#v", snapshot.DNS)
	}
	if !snapshot.DNS.Ready || snapshot.DNS.ReadyEndpoints != 1 || snapshot.DNS.TotalEndpoints != 1 {
		t.Fatalf("DNS readiness = %#v", snapshot.DNS)
	}
	if snapshot.Summary.Workloads != 3 || snapshot.Summary.ReadyWorkloads != 2 {
		t.Fatalf("workload summary = %#v", snapshot.Summary)
	}
	if snapshot.Summary.Ingresses != 1 || snapshot.Summary.PersistentVolumes != 2 || snapshot.Summary.WarningEvents != 1 {
		t.Fatalf("resource summary = %#v", snapshot.Summary)
	}
	if len(snapshot.Ingresses) != 1 || snapshot.Ingresses[0].Routes[0].URL != "https://database.example.test/admin" {
		t.Fatalf("ingress snapshot = %#v", snapshot.Ingresses)
	}
	if len(snapshot.Volumes) != 2 || len(snapshot.Events) != 2 || snapshot.Events[0].Type != corev1.EventTypeWarning {
		t.Fatalf("volume/event snapshot = volumes=%#v events=%#v", snapshot.Volumes, snapshot.Events)
	}
	for _, service := range snapshot.Services {
		if service.Name == "postgresql" {
			if len(service.Workloads) != 1 || service.Workloads[0].Kind != "Deployment" {
				t.Fatalf("service workload references = %#v", service.Workloads)
			}
			if len(service.ExternalURLs) != 1 || service.ExternalURLs[0] != "https://database.example.test/admin" {
				t.Fatalf("service external URLs = %#v", service.ExternalURLs)
			}
		}
	}

	_, live, _, _, unsubscribe := broker.Subscribe(broker.LatestID())
	defer unsubscribe()
	if _, err := kube.CoreV1().Services("apps").Create(
		context.Background(),
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "new-api", Namespace: "apps"}},
		metav1.CreateOptions{},
	); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-live:
		if event.Name != "snapshot" {
			t.Fatalf("watch event name = %q", event.Name)
		}
		var updated model.Snapshot
		if err := json.Unmarshal(event.Data, &updated); err != nil {
			t.Fatal(err)
		}
		if updated.Summary.Services != 3 {
			t.Fatalf("watched service summary = %#v", updated.Summary)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for informer-driven snapshot")
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func pathTypePtr(value networkingv1.PathType) *networkingv1.PathType {
	return &value
}
