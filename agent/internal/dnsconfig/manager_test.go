package dnsconfig

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func TestManagerReplacesServiceAliasesWithOptimisticConcurrency(t *testing.T) {
	t.Parallel()

	kube := kubernetesfake.NewSimpleClientset(
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "grafana", Namespace: "monitoring"}},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "coredns-custom",
				Namespace:       "kube-system",
				ResourceVersion: "12",
			},
		},
	)
	manager := New(kube, Options{
		Enabled:       true,
		Namespace:     "kube-system",
		ConfigMapName: "coredns-custom",
		OverrideKey:   "kubedeck.override",
		ClusterDomain: "cluster.local",
	})
	ctx := context.Background()

	initial, err := manager.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !initial.Enabled || !initial.Available || initial.ResourceVersion != "12" {
		t.Fatalf("unexpected initial state: %#v", initial)
	}

	request := ReplaceRequest{
		ResourceVersion: initial.ResourceVersion,
		Aliases: []Alias{{
			Hostname:  "Grafana.Home.Arpa.",
			Service:   "grafana",
			Namespace: "monitoring",
		}},
		DryRun: true,
	}
	preview, err := manager.Replace(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || !strings.Contains(
		preview.Rendered,
		"rewrite stop name exact grafana.home.arpa grafana.monitoring.svc.cluster.local",
	) {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	stored, err := kube.CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns-custom", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if stored.Data["kubedeck.override"] != "" {
		t.Fatalf("dry run changed ConfigMap: %#v", stored.Data)
	}

	request.DryRun = false
	applied, err := manager.Replace(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if applied.DryRun || len(applied.Aliases) != 1 ||
		applied.Aliases[0].Hostname != "grafana.home.arpa" ||
		applied.UpdatedAt == nil {
		t.Fatalf("unexpected applied state: %#v", applied)
	}
	stored, err = kube.CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns-custom", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if stored.Annotations[managedByAnnotation] != "kubedeck-agent" {
		t.Fatalf("missing manager annotation: %#v", stored.Annotations)
	}
}

func TestManagerRejectsUnsafeOrStaleAliases(t *testing.T) {
	t.Parallel()

	kube := kubernetesfake.NewSimpleClientset(
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "apps"}},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "coredns-custom",
				Namespace:       "kube-system",
				ResourceVersion: "21",
			},
		},
	)
	manager := New(kube, Options{
		Enabled:       true,
		Namespace:     "kube-system",
		ConfigMapName: "coredns-custom",
		OverrideKey:   "kubedeck.override",
		ClusterDomain: "cluster.local",
	})
	ctx := context.Background()

	_, err := manager.Replace(ctx, ReplaceRequest{
		ResourceVersion: "21",
		Aliases: []Alias{{
			Hostname:  "api.apps.svc.cluster.local",
			Service:   "api",
			Namespace: "apps",
		}},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("native service alias error = %v", err)
	}

	_, err = manager.Replace(ctx, ReplaceRequest{
		ResourceVersion: "stale",
		Aliases: []Alias{{
			Hostname:  "api.home.arpa",
			Service:   "api",
			Namespace: "apps",
		}},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale resourceVersion error = %v", err)
	}

	_, err = manager.Replace(ctx, ReplaceRequest{
		ResourceVersion: "21",
		Aliases: []Alias{{
			Hostname:  "missing.home.arpa",
			Service:   "missing",
			Namespace: "apps",
		}},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing target error = %v", err)
	}
}

func TestManagerDoesNotOverwriteUnmanagedOverride(t *testing.T) {
	t.Parallel()

	kube := kubernetesfake.NewSimpleClientset(
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "apps"}},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "coredns-custom",
				Namespace:       "kube-system",
				ResourceVersion: "4",
			},
			Data: map[string]string{
				"kubedeck.override": "forward example.test 10.0.0.53\n",
			},
		},
	)
	manager := New(kube, Options{
		Enabled:       true,
		Namespace:     "kube-system",
		ConfigMapName: "coredns-custom",
		OverrideKey:   "kubedeck.override",
		ClusterDomain: "cluster.local",
	})

	_, err := manager.Replace(context.Background(), ReplaceRequest{
		ResourceVersion: "4",
		Aliases: []Alias{{
			Hostname:  "api.home.arpa",
			Service:   "api",
			Namespace: "apps",
		}},
	})
	if !errors.Is(err, ErrUnmanaged) {
		t.Fatalf("unmanaged override error = %v", err)
	}
}

func TestManagerReportsMissingConfigMapWithoutCreatingIt(t *testing.T) {
	t.Parallel()

	manager := New(kubernetesfake.NewSimpleClientset(), Options{
		Enabled:       true,
		Namespace:     "kube-system",
		ConfigMapName: "coredns-custom",
		OverrideKey:   "kubedeck.override",
		ClusterDomain: "cluster.local",
	})

	state, err := manager.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Available {
		t.Fatalf("missing ConfigMap reported available: %#v", state)
	}
	_, err = manager.Replace(context.Background(), ReplaceRequest{ResourceVersion: "1"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing ConfigMap replace error = %v", err)
	}
}
