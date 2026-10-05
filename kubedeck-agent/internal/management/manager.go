package management

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var segment = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,252}$`)

// Manager exposes bounded Kubernetes operations and excludes the Secret resource.
type Manager struct {
	Dynamic   dynamic.Interface
	Discovery discovery.DiscoveryInterface
	Kube      kubernetes.Interface
	Logger    *slog.Logger
}

func (m *Manager) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/manage/capabilities", m.capabilities)
	mux.HandleFunc("/v1/manage/resources/{group}/{version}/{resource}", m.resource)
	mux.HandleFunc("GET /v1/manage/pods/{namespace}/{name}/logs", m.logs)
	mux.HandleFunc("GET /v1/manage/events/{namespace}", m.events)
	mux.HandleFunc("POST /v1/manage/workloads/{kind}/{namespace}/{name}/{action}", m.workload)
	return mux
}

func (m *Manager) capabilities(w http.ResponseWriter, r *http.Request) {
	groups, err := m.Discovery.ServerGroups()
	if err != nil {
		writeError(w, err)
		return
	}
	resources := make([]map[string]any, 0)
	for _, group := range groups.Groups {
		for _, version := range group.Versions {
			list, err := m.Discovery.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				continue
			}
			for _, res := range list.APIResources {
				if strings.Contains(res.Name, "/") || res.Name == "secrets" {
					continue
				}
				resources = append(resources, map[string]any{"groupVersion": version.GroupVersion, "resource": res.Name, "kind": res.Kind, "namespaced": res.Namespaced, "verbs": res.Verbs})
			}
		}
	}
	// Core v1 may not appear in ServerGroups on every API server.
	if list, err := m.Discovery.ServerResourcesForGroupVersion("v1"); err == nil {
		for _, res := range list.APIResources {
			if strings.Contains(res.Name, "/") || res.Name == "secrets" {
				continue
			}
			resources = append(resources, map[string]any{"groupVersion": "v1", "resource": res.Name, "kind": res.Kind, "namespaced": res.Namespaced, "verbs": res.Verbs})
		}
	}
	metrics := false
	if _, err := m.Discovery.ServerResourcesForGroupVersion("metrics.k8s.io/v1beta1"); err == nil {
		metrics = true
	}
	writeJSON(w, http.StatusOK, map[string]any{"managementEnabled": true, "metricsAvailable": metrics, "resources": resources, "workloadActions": []string{"scale", "restart", "status"}, "podLogs": true, "events": true, "note": "resource verbs describe API-server support; Kubernetes RBAC authorizes each request"})
}

func (m *Manager) resource(w http.ResponseWriter, r *http.Request) {
	group, version, resource := r.PathValue("group"), r.PathValue("version"), r.PathValue("resource")
	if !segment.MatchString(group) || !segment.MatchString(version) || !segment.MatchString(resource) || resource == "secrets" {
		http.Error(w, "unsupported resource", http.StatusBadRequest)
		return
	}
	if group == "core" {
		group = ""
	}
	gv := version
	if group != "" {
		gv = group + "/" + version
	}
	available, err := m.Discovery.ServerResourcesForGroupVersion(gv)
	if err != nil {
		writeError(w, err)
		return
	}
	var apiResource *metav1.APIResource
	for i := range available.APIResources {
		if available.APIResources[i].Name == resource {
			apiResource = &available.APIResources[i]
			break
		}
	}
	if apiResource == nil {
		http.Error(w, "resource is not served", http.StatusNotFound)
		return
	}
	namespace, name := r.URL.Query().Get("namespace"), r.URL.Query().Get("name")
	if (namespace != "" && !segment.MatchString(namespace)) || (name != "" && !segment.MatchString(name)) {
		http.Error(w, "invalid name or namespace", http.StatusBadRequest)
		return
	}
	if apiResource.Namespaced && namespace == "" && r.Method != http.MethodGet {
		http.Error(w, "namespace required for writes", http.StatusBadRequest)
		return
	}
	if !apiResource.Namespaced && namespace != "" {
		http.Error(w, "cluster resource cannot have namespace", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodGet {
		defer m.audit(r, resource, namespace, name, strings.ToLower(r.Method))
	}
	if r.Method != http.MethodGet && r.Header.Get("X-KubeDeck-Confirm") != namespace+"/"+name && r.Method != http.MethodPost {
		http.Error(w, "exact confirmation required", http.StatusConflict)
		return
	}
	client := m.Dynamic.Resource(schema.GroupVersionResource{Group: group, Version: version, Resource: resource})
	var target dynamic.ResourceInterface = client
	if apiResource.Namespaced {
		target = client.Namespace(namespace)
	}
	ctx := r.Context()
	var result any
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("watch") == "true" {
			m.watch(w, r, target)
			return
		}
		if name == "" {
			result, err = target.List(ctx, metav1.ListOptions{Limit: 100, Continue: r.URL.Query().Get("continue")})
		} else {
			result, err = target.Get(ctx, name, metav1.GetOptions{})
		}
	case http.MethodPost:
		var obj *unstructured.Unstructured
		obj, err = readObject(w, r, namespace, name, gv, apiResource.Kind)
		if err != nil {
			return
		}
		if r.Header.Get("X-KubeDeck-Confirm") != namespace+"/"+obj.GetName() {
			http.Error(w, "exact confirmation required", http.StatusConflict)
			return
		}
		result, err = target.Create(ctx, obj, metav1.CreateOptions{DryRun: dryRun(r)})
	case http.MethodPut:
		if name == "" {
			http.Error(w, "name required", 400)
			return
		}
		var obj *unstructured.Unstructured
		obj, err = readObject(w, r, namespace, name, gv, apiResource.Kind)
		if err != nil {
			return
		}
		if obj.GetResourceVersion() == "" {
			http.Error(w, "resourceVersion required", 409)
			return
		}
		result, err = target.Update(ctx, obj, metav1.UpdateOptions{DryRun: dryRun(r)})
	case http.MethodPatch:
		if name == "" {
			http.Error(w, "name required", 400)
			return
		}
		body, readErr := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if readErr != nil {
			http.Error(w, "invalid body", 400)
			return
		}
		patchType := types.MergePatchType
		if r.URL.Query().Get("type") == "apply" {
			patchType = types.ApplyPatchType
		}
		if value := r.URL.Query().Get("type"); value != "" && value != "merge" && value != "apply" {
			http.Error(w, "unsupported patch type", 400)
			return
		}
		if patchType == types.ApplyPatchType {
			if r.URL.Query().Get("fieldManager") == "" {
				http.Error(w, "fieldManager required", 400)
				return
			}
		}
		if r.Header.Get("If-Match") == "" {
			http.Error(w, "If-Match resourceVersion required", 409)
			return
		}
		if patchType == types.ApplyPatchType {
			body, err = yaml.YAMLToJSON(body)
			if err != nil {
				http.Error(w, "invalid apply document", 400)
				return
			}
		}
		if patchType == types.MergePatchType {
			var patch map[string]any
			if json.Unmarshal(body, &patch) != nil || patch == nil {
				http.Error(w, "invalid merge patch", 400)
				return
			}
			meta, _ := patch["metadata"].(map[string]any)
			if meta == nil {
				meta = map[string]any{}
				patch["metadata"] = meta
			}
			meta["resourceVersion"] = r.Header.Get("If-Match")
			body, _ = json.Marshal(patch)
		} else {
			var object map[string]any
			if json.Unmarshal(body, &object) != nil {
				http.Error(w, "invalid apply document", 400)
				return
			}
			meta, _ := object["metadata"].(map[string]any)
			if meta == nil || meta["name"] != name || meta["namespace"] != namespace || object["apiVersion"] != gv || object["kind"] != apiResource.Kind {
				http.Error(w, "apply identity mismatch", 400)
				return
			}
			meta["resourceVersion"] = r.Header.Get("If-Match")
			body, _ = json.Marshal(object)
		}
		opts := metav1.PatchOptions{DryRun: dryRun(r), FieldManager: r.URL.Query().Get("fieldManager")}
		result, err = target.Patch(ctx, name, patchType, body, opts)
	case http.MethodDelete:
		if name == "" {
			http.Error(w, "name required", 400)
			return
		}
		uid := r.Header.Get("If-Match-UID")
		version := r.Header.Get("If-Match")
		if uid == "" || version == "" || r.Header.Get("X-KubeDeck-Confirm") != namespace+"/"+name {
			http.Error(w, "UID and version preconditions plus exact confirmation required", 409)
			return
		}
		opts := metav1.DeleteOptions{DryRun: dryRun(r), Preconditions: &metav1.Preconditions{UID: (*types.UID)(&uid), ResourceVersion: &version}}
		err = target.Delete(ctx, name, opts)
		result = map[string]string{"status": "accepted"}
	default:
		w.Header().Set("Allow", "GET, POST, PUT, PATCH, DELETE")
		http.Error(w, "method not allowed", 405)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (m *Manager) watch(w http.ResponseWriter, r *http.Request, target dynamic.ResourceInterface) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	stream, err := target.Watch(ctx, metav1.ListOptions{Watch: true, ResourceVersion: r.URL.Query().Get("resourceVersion")})
	if err != nil {
		writeError(w, err)
		return
	}
	defer stream.Stop()
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-store")
	flusher, _ := w.(http.Flusher)
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-stream.ResultChan():
			if !ok {
				return
			}
			if json.NewEncoder(w).Encode(event) != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func (m *Manager) logs(w http.ResponseWriter, r *http.Request) {
	namespace, name := r.PathValue("namespace"), r.PathValue("name")
	if !segment.MatchString(namespace) || !segment.MatchString(name) {
		http.Error(w, "invalid pod", 400)
		return
	}
	tail := int64(200)
	if value := r.URL.Query().Get("tail"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 || parsed > 1000 {
			http.Error(w, "tail must be 1..1000", 400)
			return
		}
		tail = parsed
	}
	options := &corev1.PodLogOptions{Container: r.URL.Query().Get("container"), TailLines: &tail, LimitBytes: int64Ptr(1 << 20)}
	stream, err := m.Kube.CoreV1().Pods(namespace).GetLogs(name, options).Stream(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	defer stream.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.Copy(w, io.LimitReader(stream, 1<<20))
}

func (m *Manager) events(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	if !segment.MatchString(namespace) {
		http.Error(w, "invalid namespace", 400)
		return
	}
	result, err := m.Kube.CoreV1().Events(namespace).List(r.Context(), metav1.ListOptions{Limit: 100})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (m *Manager) workload(w http.ResponseWriter, r *http.Request) {
	kind, namespace, name, action := r.PathValue("kind"), r.PathValue("namespace"), r.PathValue("name"), r.PathValue("action")
	if !segment.MatchString(namespace) || !segment.MatchString(name) {
		http.Error(w, "invalid workload", 400)
		return
	}
	if kind != "deployments" && kind != "statefulsets" && kind != "daemonsets" {
		http.Error(w, "unsupported workload", 400)
		return
	}
	if action != "status" && action != "scale" && action != "restart" {
		http.Error(w, "unsupported action", 400)
		return
	}
	if action != "status" {
		defer m.audit(r, kind, namespace, name, action)
	}
	if action != "status" && r.Header.Get("X-KubeDeck-Confirm") != kind+"/"+namespace+"/"+name {
		http.Error(w, "exact confirmation required", 409)
		return
	}
	ctx := r.Context()
	if action == "status" {
		var value any
		var err error
		switch kind {
		case "deployments":
			value, err = m.Kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		case "statefulsets":
			value, err = m.Kube.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		default:
			value, err = m.Kube.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		}
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, 200, value)
		return
	}
	if action == "scale" {
		if kind == "daemonsets" {
			http.Error(w, "DaemonSet scale unsupported", 400)
			return
		}
		var req struct {
			Replicas        *int32 `json:"replicas"`
			ResourceVersion string `json:"resourceVersion"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || req.Replicas == nil || *req.Replicas < 0 || *req.Replicas > 100 || req.ResourceVersion == "" {
			http.Error(w, "replicas 0..100 and resourceVersion required", 400)
			return
		}
		if kind == "deployments" {
			current, err := m.Kube.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
			if err != nil {
				writeError(w, err)
				return
			}
			if current.ResourceVersion != req.ResourceVersion {
				http.Error(w, "stale resourceVersion", 409)
				return
			}
			current.Spec.Replicas = *req.Replicas
			result, err := m.Kube.AppsV1().Deployments(namespace).UpdateScale(ctx, name, current, metav1.UpdateOptions{DryRun: dryRun(r)})
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 200, result)
			return
		}
		current, err := m.Kube.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		if err != nil {
			writeError(w, err)
			return
		}
		if current.ResourceVersion != req.ResourceVersion {
			http.Error(w, "stale resourceVersion", 409)
			return
		}
		current.Spec.Replicas = *req.Replicas
		result, err := m.Kube.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, current, metav1.UpdateOptions{DryRun: dryRun(r)})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, 200, result)
		return
	}
	// Restart only changes the pod template annotation; the controller performs rollout.
	if r.Header.Get("If-Match") == "" {
		http.Error(w, "If-Match resourceVersion required", 409)
		return
	}
	patch := map[string]any{"metadata": map[string]any{"resourceVersion": r.Header.Get("If-Match")}, "spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]string{"kubectl.kubernetes.io/restartedAt": time.Now().UTC().Format(time.RFC3339Nano)}}}}}
	body, _ := json.Marshal(patch)
	target := m.Dynamic.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: kind}).Namespace(namespace)
	result, err := target.Patch(ctx, name, types.StrategicMergePatchType, body, metav1.PatchOptions{DryRun: dryRun(r)})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func readObject(w http.ResponseWriter, r *http.Request, namespace, name, groupVersion, kind string) (*unstructured.Unstructured, error) {
	var obj unstructured.Unstructured
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&obj.Object); err != nil {
		http.Error(w, "invalid object JSON", 400)
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		http.Error(w, "request body must contain one JSON object", 400)
		return nil, errors.New("trailing JSON")
	}
	if obj.GetName() == "" || name != "" && obj.GetName() != name || obj.GetNamespace() != namespace || obj.GetAPIVersion() != groupVersion || obj.GetKind() != kind {
		http.Error(w, "object identity mismatch", 400)
		return nil, errors.New("identity mismatch")
	}
	return &obj, nil
}
func dryRun(r *http.Request) []string {
	if r.URL.Query().Get("dryRun") == "true" {
		return []string{metav1.DryRunAll}
	}
	return nil
}
func int64Ptr(value int64) *int64 { return &value }
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, err error) {
	status := 500
	if apierrors.IsNotFound(err) {
		status = 404
	} else if apierrors.IsForbidden(err) {
		status = 403
	} else if apierrors.IsConflict(err) {
		status = 409
	} else if apierrors.IsBadRequest(err) {
		status = 400
	}
	http.Error(w, http.StatusText(status), status)
}
func (m *Manager) audit(r *http.Request, resource, namespace, name, action string) {
	if m.Logger != nil {
		m.Logger.Info("Kubernetes management request", "action", action, "resource", resource, "namespace", namespace, "name", name, "dryRun", len(dryRun(r)) > 0, "requestId", r.Header.Get("X-Request-ID"))
	}
}
