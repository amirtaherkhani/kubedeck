package management

import (
	"maps"
	"net/http"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func batchTarget(r *http.Request) (string, string, bool) {
	namespace, name := r.PathValue("namespace"), r.PathValue("name")
	return namespace, name, segment.MatchString(namespace) && segment.MatchString(name)
}

func (m *Manager) jobStatus(w http.ResponseWriter, r *http.Request) {
	namespace, name, valid := batchTarget(r)
	if !valid {
		http.Error(w, "invalid Job identity", http.StatusBadRequest)
		return
	}
	job, err := m.Kube.BatchV1().Jobs(namespace).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, jobSummary(job))
}

func (m *Manager) runCronJob(w http.ResponseWriter, r *http.Request) {
	namespace, name, valid := batchTarget(r)
	if !valid {
		http.Error(w, "invalid CronJob identity", http.StatusBadRequest)
		return
	}
	defer m.audit(r, "cronjobs", namespace, name, "run")
	if !batchConfirmed(r, "cronjobs/"+namespace+"/"+name+"/run") {
		http.Error(w, "exact confirmation, UID and resourceVersion required", http.StatusConflict)
		return
	}
	cronJob, err := m.Kube.BatchV1().CronJobs(namespace).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, err)
		return
	}
	if !batchIdentityMatches(r, cronJob.UID, cronJob.ResourceVersion) {
		http.Error(w, "CronJob identity changed", http.StatusConflict)
		return
	}
	// GenerateName avoids collisions while keeping the caller's scope limited
	// to this CronJob's template. No arbitrary Job or PodSpec is accepted.
	prefix := name
	if len(prefix) > 48 {
		prefix = prefix[:48]
	}
	labels := maps.Clone(cronJob.Spec.JobTemplate.Labels)
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["kuchdesk.io/source-cronjob"] = name
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: prefix + "-manual-",
			Labels:       labels,
			Annotations:  maps.Clone(cronJob.Spec.JobTemplate.Annotations),
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}
	created, err := m.Kube.BatchV1().Jobs(namespace).Create(r.Context(), job, metav1.CreateOptions{DryRun: dryRun(r)})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, jobSummary(created))
}

func (m *Manager) deleteJob(w http.ResponseWriter, r *http.Request) {
	namespace, name, valid := batchTarget(r)
	if !valid {
		http.Error(w, "invalid Job identity", http.StatusBadRequest)
		return
	}
	defer m.audit(r, "jobs", namespace, name, "delete")
	if !batchConfirmed(r, "jobs/"+namespace+"/"+name+"/delete") {
		http.Error(w, "exact confirmation, UID and resourceVersion required", http.StatusConflict)
		return
	}
	uid := types.UID(r.Header.Get("If-Match-UID"))
	version := r.Header.Get("If-Match")
	policy := metav1.DeletePropagationForeground
	err := m.Kube.BatchV1().Jobs(namespace).Delete(r.Context(), name, metav1.DeleteOptions{
		Preconditions:     &metav1.Preconditions{UID: &uid, ResourceVersion: &version},
		PropagationPolicy: &policy,
		DryRun:            dryRun(r),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"namespace": namespace, "name": name, "deletionAccepted": true, "dryRun": len(dryRun(r)) > 0})
}

func batchConfirmed(r *http.Request, expected string) bool {
	return r.Header.Get("X-KuchDesk-Confirm") == expected && r.Header.Get("If-Match-UID") != "" && r.Header.Get("If-Match") != ""
}

func batchIdentityMatches(r *http.Request, uid types.UID, resourceVersion string) bool {
	return r.Header.Get("If-Match-UID") == string(uid) && r.Header.Get("If-Match") == resourceVersion
}

func jobSummary(job *batchv1.Job) map[string]any {
	conditions := make([]map[string]string, 0, len(job.Status.Conditions))
	for _, condition := range job.Status.Conditions {
		conditions = append(conditions, map[string]string{"type": string(condition.Type), "status": string(condition.Status), "reason": condition.Reason})
	}
	return map[string]any{
		"namespace": job.Namespace, "name": job.Name, "uid": job.UID,
		"resourceVersion": job.ResourceVersion, "active": job.Status.Active,
		"succeeded": job.Status.Succeeded, "failed": job.Status.Failed,
		"conditions": conditions, "completionTime": job.Status.CompletionTime,
	}
}
