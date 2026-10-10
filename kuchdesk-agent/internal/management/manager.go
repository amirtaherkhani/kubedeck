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
	"sync"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"
)

var segment = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,252}$`)

// Manager exposes bounded Kubernetes operations and excludes the Secret resource.
type Manager struct {
	Dynamic            dynamic.Interface
	Discovery          discovery.DiscoveryInterface
	Kube               kubernetes.Interface
	RESTConfig         *rest.Config
	Logger             *slog.Logger
	Jobs               *WorkloadJobs
	ExecEnabled        bool
	ExecRun            PodExecRunner
	execOnce           sync.Once
	execSlots          chan struct{}
	execTimeout        time.Duration
	PortForwardEnabled bool
	ForwardRun         PortForwardRunner
	forwardOnce        sync.Once
	forwardSlots       chan struct{}
	forwardLifetime    time.Duration
	forwardIdle        time.Duration
	forwardByteLimit   int64
}

func (m *Manager) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/manage/capabilities", m.capabilities)
	mux.HandleFunc("GET /v1/manage/workloads/{kind}/{namespace}/{name}/{action}", m.readWorkload)
	mux.HandleFunc("/v1/manage/resources/{group}/{version}/{resource}", m.resource)
	mux.HandleFunc("GET /v1/manage/pods/{namespace}/{name}/logs", m.logs)
	mux.HandleFunc("GET /v1/manage/events/{namespace}", m.events)
	mux.HandleFunc("GET /v1/manage/batch/jobs/{namespace}/{name}", m.jobStatus)
	mux.HandleFunc("POST /v1/manage/batch/cronjobs/{namespace}/{name}/run", m.runCronJob)
	mux.HandleFunc("DELETE /v1/manage/batch/jobs/{namespace}/{name}", m.deleteJob)
	if m.ExecEnabled {
		mux.HandleFunc("POST /v1/manage/pods/{namespace}/{name}/exec", m.execPod)
	}
	if m.PortForwardEnabled {
		mux.HandleFunc("GET /v1/manage/pods/{namespace}/{name}/port-forward", m.portForwardPod)
	}
	mux.HandleFunc("POST /v1/manage/workloads/{kind}/{namespace}/{name}/{action}", m.workload)
	if m.Jobs != nil {
		mux.HandleFunc("POST /v1/manage/workloads/{kind}/{namespace}/{name}/{action}/jobs", m.submitWorkloadJob)
		mux.HandleFunc("GET /v1/manage/jobs/{id}", m.workloadJobStatus)
		mux.HandleFunc("DELETE /v1/manage/jobs/{id}", m.cancelWorkloadJob)
	}
	return mux
}

func (m *Manager) capabilities(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	if namespace != "" && !segment.MatchString(namespace) {
		http.Error(w, "invalid namespace", http.StatusBadRequest)
		return
	}
	groups, err := m.Discovery.ServerGroups()
	if err != nil {
		writeError(w, err)
		return
	}
	resources := make([]map[string]any, 0)
	allowedByResource := make([][]bool, 0)
	type reviewTask struct {
		resourceIndex int
		verbIndex     int
		attributes    authv1.ResourceAttributes
	}
	var reviews []reviewTask
	seen := make(map[string]bool)
	appendResources := func(list *metav1.APIResourceList) error {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return err
		}
		for _, res := range list.APIResources {
			parts := strings.Split(res.Name, "/")
			if parts[0] == "secrets" || len(parts) > 2 {
				continue
			}
			key := list.GroupVersion + "/" + res.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			subresource := ""
			if len(parts) == 2 {
				subresource = parts[1]
			}
			resourceIndex := len(resources)
			allowedByResource = append(allowedByResource, make([]bool, len(res.Verbs)))
			for verbIndex, verb := range res.Verbs {
				// A namespaced review needs a concrete namespace to be meaningful.
				if res.Namespaced && namespace == "" {
					continue
				}
				reviews = append(reviews, reviewTask{resourceIndex: resourceIndex, verbIndex: verbIndex, attributes: authv1.ResourceAttributes{Namespace: namespaceForReview(res.Namespaced, namespace), Verb: verb, Group: gv.Group, Version: gv.Version, Resource: parts[0], Subresource: subresource}})
			}
			resources = append(resources, map[string]any{"groupVersion": list.GroupVersion, "resource": res.Name, "kind": res.Kind, "namespaced": res.Namespaced, "verbs": res.Verbs})
		}
		return nil
	}
	for _, group := range groups.Groups {
		for _, version := range group.Versions {
			list, err := m.Discovery.ServerResourcesForGroupVersion(version.GroupVersion)
			if err != nil {
				continue
			}
			if err := appendResources(list); err != nil {
				writeError(w, err)
				return
			}
		}
	}
	// Core v1 may not appear in ServerGroups on every API server.
	if list, err := m.Discovery.ServerResourcesForGroupVersion("v1"); err == nil {
		if err := appendResources(list); err != nil {
			writeError(w, err)
			return
		}
	}
	// RulesReview avoids one API request per resource verb for this
	// informational catalog. It is suitable here because Kubernetes
	// still authorizes every actual operation. Fall back to exact access reviews
	// when rules are incomplete or unavailable.
	rulesComplete := false
	if namespace != "" {
		rules, err := m.Kube.AuthorizationV1().SelfSubjectRulesReviews().Create(r.Context(), &authv1.SelfSubjectRulesReview{Spec: authv1.SelfSubjectRulesReviewSpec{Namespace: namespace}}, metav1.CreateOptions{})
		if err == nil && !rules.Status.Incomplete && rules.Status.EvaluationError == "" {
			rulesComplete = true
			for _, task := range reviews {
				allowedByResource[task.resourceIndex][task.verbIndex] = allowedByRules(rules.Status.ResourceRules, task.attributes)
			}
		}
	}
	if !rulesComplete && len(reviews) > 0 {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		jobs := make(chan reviewTask)
		var workers sync.WaitGroup
		var firstErr error
		var once sync.Once
		workerCount := min(8, len(reviews))
		for range workerCount {
			workers.Add(1)
			go func() {
				defer workers.Done()
				for task := range jobs {
					review, err := m.Kube.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &authv1.SelfSubjectAccessReview{Spec: authv1.SelfSubjectAccessReviewSpec{ResourceAttributes: &task.attributes}}, metav1.CreateOptions{})
					if err != nil {
						once.Do(func() { firstErr = err; cancel() })
						return
					}
					allowedByResource[task.resourceIndex][task.verbIndex] = review.Status.Allowed && !review.Status.Denied && review.Status.EvaluationError == ""
				}
			}()
		}
	sendReviews:
		for _, task := range reviews {
			select {
			case jobs <- task:
			case <-ctx.Done():
				break sendReviews
			}
		}
		close(jobs)
		workers.Wait()
		if firstErr != nil {
			writeError(w, firstErr)
			return
		}
		if r.Context().Err() != nil {
			return
		}
	}
	for i, resource := range resources {
		allowed := make(map[string]bool)
		if !resource["namespaced"].(bool) || namespace != "" {
			for j, verb := range resource["verbs"].(metav1.Verbs) {
				allowed[verb] = allowedByResource[i][j]
			}
		}
		resource["allowedVerbs"] = allowed
	}
	metrics := false
	if _, err := m.Discovery.ServerResourcesForGroupVersion("metrics.k8s.io/v1beta1"); err == nil {
		metrics = true
	}
	writeJSON(w, http.StatusOK, map[string]any{"managementEnabled": true, "metricsAvailable": metrics, "resources": resources, "authorizationNamespace": namespace, "workloadActions": []string{"scale", "restart", "status"}, "asyncWorkloadActions": m.Jobs != nil, "podLogs": true, "podExec": m.ExecEnabled, "podPortForward": m.PortForwardEnabled, "batchJobLifecycle": true, "events": true, "note": "allowedVerbs are point-in-time authorization information; each operation is authorized again by Kubernetes"})
}

func allowedByRules(rules []authv1.ResourceRule, attributes authv1.ResourceAttributes) bool {
	resource := attributes.Resource
	if attributes.Subresource != "" {
		resource += "/" + attributes.Subresource
	}
	for _, rule := range rules {
		if len(rule.ResourceNames) > 0 && !contains(rule.ResourceNames, "*") {
			continue
		}
		if !contains(rule.Verbs, attributes.Verb) && !contains(rule.Verbs, "*") {
			continue
		}
		if !contains(rule.APIGroups, attributes.Group) && !contains(rule.APIGroups, "*") {
			continue
		}
		if contains(rule.Resources, "*") || contains(rule.Resources, resource) {
			return true
		}
		if attributes.Subresource != "" && contains(rule.Resources, "*/"+attributes.Subresource) {
			return true
		}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func namespaceForReview(namespaced bool, namespace string) string {
	if namespaced {
		return namespace
	}
	return ""
}

func (m *Manager) readWorkload(w http.ResponseWriter, r *http.Request) {
	kind, namespace, name, action := r.PathValue("kind"), r.PathValue("namespace"), r.PathValue("name"), r.PathValue("action")
	if !segment.MatchString(namespace) || !segment.MatchString(name) {
		http.Error(w, "invalid workload", http.StatusBadRequest)
		return
	}
	if action != "status" && action != "scale" {
		http.Error(w, "unsupported read action", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	var value any
	var err error
	switch kind {
	case "deployments":
		if action == "scale" {
			value, err = m.Kube.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
		} else {
			value, err = m.Kube.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		}
	case "statefulsets":
		if action == "scale" {
			value, err = m.Kube.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
		} else {
			value, err = m.Kube.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		}
	case "daemonsets":
		if action == "scale" {
			http.Error(w, "DaemonSet scale unsupported", http.StatusBadRequest)
			return
		}
		value, err = m.Kube.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	default:
		http.Error(w, "unsupported workload", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
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
	if r.Method != http.MethodGet && r.Header.Get("X-KuchDesk-Confirm") != namespace+"/"+name && r.Method != http.MethodPost {
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
		if r.Header.Get("X-KuchDesk-Confirm") != namespace+"/"+obj.GetName() {
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
		if uid == "" || version == "" || r.Header.Get("X-KuchDesk-Confirm") != namespace+"/"+name {
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
	if action != "status" && r.Header.Get("X-KuchDesk-Confirm") != kind+"/"+namespace+"/"+name {
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
			http.Error(w, "DaemonSet scale unsupported", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
		if err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		replicas, resourceVersion, err := parseScale(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := m.scale(ctx, kind, namespace, name, replicas, resourceVersion, len(dryRun(r)) > 0)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if r.Header.Get("If-Match") == "" {
		http.Error(w, "If-Match resourceVersion required", http.StatusConflict)
		return
	}
	result, err := m.restart(ctx, kind, namespace, name, r.Header.Get("If-Match"), len(dryRun(r)) > 0)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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
	status := statusForError(err)
	http.Error(w, http.StatusText(status), status)
}
func statusForError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	status := http.StatusInternalServerError
	if apierrors.IsNotFound(err) {
		status = http.StatusNotFound
	} else if apierrors.IsForbidden(err) {
		status = http.StatusForbidden
	} else if apierrors.IsConflict(err) {
		status = http.StatusConflict
	} else if apierrors.IsBadRequest(err) {
		status = http.StatusBadRequest
	}
	return status
}
func (m *Manager) audit(r *http.Request, resource, namespace, name, action string) {
	if m.Logger != nil {
		m.Logger.Info("Kubernetes management request", "action", action, "resource", resource, "namespace", namespace, "name", name, "dryRun", len(dryRun(r)) > 0, "requestId", r.Header.Get("X-Request-ID"))
	}
}
