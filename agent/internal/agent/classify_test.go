package agent

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestClassifyService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serviceName string
		labels      map[string]string
		annotations map[string]string
		ports       []corev1.ServicePort
		hasIngress  bool
		want        string
	}{
		{
			name:        "explicit annotation wins",
			serviceName: "postgresql",
			annotations: map[string]string{"app.kubedeck.io/category": "AI & MCP"},
			want:        CategoryAIServices,
		},
		{
			name:        "database heuristic",
			serviceName: "shared-postgresql",
			want:        CategoryDatabasesStorage,
		},
		{
			name:        "observability label",
			serviceName: "collector",
			labels:      map[string]string{"app.kubernetes.io/name": "grafana-alloy"},
			want:        CategoryObservability,
		},
		{
			name:        "ingress fallback",
			serviceName: "custom-console",
			hasIngress:  true,
			want:        CategoryWebApplications,
		},
		{
			name:        "http port fallback",
			serviceName: "custom-console",
			ports:       []corev1.ServicePort{{Name: "http", Port: 9099}},
			want:        CategoryWebApplications,
		},
		{
			name:        "unknown service",
			serviceName: "internal-protocol",
			ports:       []corev1.ServicePort{{Name: "tcp", Port: 9444}},
			want:        CategoryOther,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyService(
				test.serviceName,
				test.labels,
				test.annotations,
				test.ports,
				test.hasIngress,
			)
			if got != test.want {
				t.Fatalf("ClassifyService() = %q, want %q", got, test.want)
			}
		})
	}
}
