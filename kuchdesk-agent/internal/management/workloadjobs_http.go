package management

import (
	"context"
	"errors"
	"io"
	"net/http"
)

// submitWorkloadJob uses the same typed scale/restart methods as the synchronous
// route. Only validated action parameters are retained; workload response
// bodies and environment values are never stored in the job record.
func (m *Manager) submitWorkloadJob(w http.ResponseWriter, r *http.Request) {
	kind, namespace, name, action := r.PathValue("kind"), r.PathValue("namespace"), r.PathValue("name"), r.PathValue("action")
	if !segment.MatchString(namespace) || !segment.MatchString(name) || (kind != "deployments" && kind != "statefulsets" && kind != "daemonsets") || (action != "scale" && action != "restart") || (kind == "daemonsets" && action == "scale") {
		http.Error(w, "unsupported workload action", http.StatusBadRequest)
		return
	}
	if r.Header.Get("X-KuchDesk-Confirm") != kind+"/"+namespace+"/"+name {
		http.Error(w, "exact confirmation required", http.StatusConflict)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if action == "restart" && r.Header.Get("If-Match") == "" {
		http.Error(w, "If-Match resourceVersion required", http.StatusConflict)
		return
	}
	var replicas int32
	var resourceVersion string
	if action == "scale" {
		replicas, resourceVersion, err = parseScale(body)
		if err != nil {
			http.Error(w, "replicas 0..100 and resourceVersion required", http.StatusBadRequest)
			return
		}
	} else {
		resourceVersion = r.Header.Get("If-Match")
	}
	isDryRun := len(dryRun(r)) > 0
	requestID := r.Header.Get("X-Request-ID")
	id, err := m.Jobs.Submit(kind+"/"+namespace+"/"+name, func(ctx context.Context) int {
		if m.Logger != nil {
			m.Logger.Info("Kubernetes management request", "action", action, "resource", kind, "namespace", namespace, "name", name, "dryRun", isDryRun, "requestId", requestID)
		}
		var operationErr error
		if action == "scale" {
			_, operationErr = m.scale(ctx, kind, namespace, name, replicas, resourceVersion, isDryRun)
		} else {
			_, operationErr = m.restart(ctx, kind, namespace, name, resourceVersion, isDryRun)
		}
		return statusForError(operationErr)
	})
	if err != nil {
		if errors.Is(err, errJobCapacity) {
			http.Error(w, "operation capacity reached", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "operation unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "state": "queued"})
}

func (m *Manager) workloadJobStatus(w http.ResponseWriter, r *http.Request) {
	job, err := m.Jobs.Status(r.PathValue("id"))
	if err != nil {
		http.Error(w, "operation not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (m *Manager) cancelWorkloadJob(w http.ResponseWriter, r *http.Request) {
	job, err := m.Jobs.Cancel(r.PathValue("id"))
	if err != nil {
		http.Error(w, "operation not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}
