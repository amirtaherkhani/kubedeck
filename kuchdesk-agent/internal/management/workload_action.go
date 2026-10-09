package management

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func parseScale(body []byte) (int32, string, error) {
	var request struct {
		Replicas        *int32 `json:"replicas"`
		ResourceVersion string `json:"resourceVersion"`
	}
	if json.Unmarshal(body, &request) != nil || request.Replicas == nil || *request.Replicas < 0 || *request.Replicas > 100 || request.ResourceVersion == "" {
		return 0, "", errors.New("replicas 0..100 and resourceVersion required")
	}
	return *request.Replicas, request.ResourceVersion, nil
}

func (m *Manager) scale(ctx context.Context, kind, namespace, name string, replicas int32, resourceVersion string, dryRun bool) (any, error) {
	options := metav1.UpdateOptions{}
	if dryRun {
		options.DryRun = []string{metav1.DryRunAll}
	}
	if kind == "deployments" {
		current, err := m.Kube.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if current.ResourceVersion != resourceVersion {
			return nil, apierrors.NewConflict(schema.GroupResource{Group: "apps", Resource: kind}, name, errors.New("stale resourceVersion"))
		}
		current.Spec.Replicas = replicas
		return m.Kube.AppsV1().Deployments(namespace).UpdateScale(ctx, name, current, options)
	}
	current, err := m.Kube.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if current.ResourceVersion != resourceVersion {
		return nil, apierrors.NewConflict(schema.GroupResource{Group: "apps", Resource: kind}, name, errors.New("stale resourceVersion"))
	}
	current.Spec.Replicas = replicas
	return m.Kube.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, current, options)
}

func (m *Manager) restart(ctx context.Context, kind, namespace, name, resourceVersion string, dryRun bool) (any, error) {
	patch := map[string]any{"metadata": map[string]any{"resourceVersion": resourceVersion}, "spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]string{"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339Nano)}}}}}
	body, _ := json.Marshal(patch)
	options := metav1.PatchOptions{}
	if dryRun {
		options.DryRun = []string{metav1.DryRunAll}
	}
	target := m.Dynamic.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: kind}).Namespace(namespace)
	return target.Patch(ctx, name, types.StrategicMergePatchType, body, options)
}
