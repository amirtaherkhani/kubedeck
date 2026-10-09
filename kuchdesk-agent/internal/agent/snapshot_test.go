package agent

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildNodesPreservesReadyWhileReportingPressureAndCordons(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	heartbeat := metav1.NewTime(now.Add(-30 * time.Second))
	transition := metav1.NewTime(now.Add(-time.Minute))
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "worker-pressure",
			CreationTimestamp: metav1.NewTime(now.Add(-24 * time.Hour)),
		},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
			{
				Type:              corev1.NodeReady,
				Status:            corev1.ConditionTrue,
				LastHeartbeatTime: heartbeat,
			},
			{
				Type:               corev1.NodeMemoryPressure,
				Status:             corev1.ConditionTrue,
				Reason:             "KubeletHasInsufficientMemory",
				LastTransitionTime: transition,
			},
		}},
	}

	result := buildNodes(now, []*corev1.Node{node}, nil, nil)
	if len(result) != 1 || !result[0].Ready {
		t.Fatalf("Kubernetes Ready condition was lost: %#v", result)
	}
	if result[0].Status != "KubeletHasInsufficientMemory" {
		t.Fatalf("pressure status = %q", result[0].Status)
	}
	if result[0].LastHeartbeatAt == nil || !result[0].LastHeartbeatAt.Equal(heartbeat.Time) {
		t.Fatalf("heartbeat = %#v", result[0].LastHeartbeatAt)
	}

	node.Spec.Unschedulable = true
	result = buildNodes(now, []*corev1.Node{node}, nil, nil)
	if result[0].Status != "Unschedulable" || !result[0].Ready {
		t.Fatalf("cordoned node status = %#v", result[0])
	}
}

func TestEndpointSliceIndexCountsAddressesAndExcludesTerminatingEndpoints(t *testing.T) {
	t.Parallel()

	ready := true
	terminating := true
	slices := []*discoveryv1.EndpointSlice{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "apps",
				Labels: map[string]string{
					discoveryv1.LabelServiceName: "api",
				},
			},
			Endpoints: []discoveryv1.Endpoint{
				{Addresses: []string{"10.0.0.1", "10.0.0.2"}},
				{
					Addresses: []string{"10.0.0.3"},
					Conditions: discoveryv1.EndpointConditions{
						Ready:       &ready,
						Terminating: &terminating,
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "apps"},
			Endpoints:  []discoveryv1.Endpoint{{Addresses: []string{"ignored"}}},
		},
	}

	indexed := indexEndpointSlices(slices)
	if len(indexed) != 1 {
		t.Fatalf("endpoint index = %#v", indexed)
	}
	counts := indexed["apps/api"]
	if counts.total != 3 || counts.ready != 2 {
		t.Fatalf("endpoint counts = %#v", counts)
	}
}

func TestBuildDNSProfileReportsMissingAndReadyProviders(t *testing.T) {
	t.Parallel()

	missing := buildDNSProfile(nil, nil, "cluster.example")
	if missing.Provider != "Kubernetes DNS" || missing.Ready ||
		missing.ClusterDomain != "cluster.example" || len(missing.SearchPath) != 3 {
		t.Fatalf("missing DNS profile = %#v", missing)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceSystem,
			Name:      "kube-dns",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.43.0.10",
			Ports:     []corev1.ServicePort{{Port: 53, Protocol: corev1.ProtocolUDP}},
		},
	}
	ready := buildDNSProfile(
		[]*corev1.Service{service},
		map[string]endpointCounts{"kube-system/kube-dns": {ready: 2, total: 3}},
		"cluster.local",
	)
	if !ready.Ready || ready.Provider != "CoreDNS" ||
		ready.ServiceDNS != "kube-dns.kube-system.svc.cluster.local" ||
		ready.ReadyEndpoints != 2 || ready.TotalEndpoints != 3 {
		t.Fatalf("ready DNS profile = %#v", ready)
	}
}
