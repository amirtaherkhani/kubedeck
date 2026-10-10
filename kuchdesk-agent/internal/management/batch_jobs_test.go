package management

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func batchManager(t *testing.T) *Manager {
	t.Helper()
	m := testManager(t)
	m.Kube = kubefake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{Namespace: "apps", Name: "nightly", UID: types.UID("cron-uid"), ResourceVersion: "5"},
			Spec: batchv1.CronJobSpec{JobTemplate: batchv1.JobTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"team": "platform"}, Annotations: map[string]string{"purpose": "daily"}}, Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "worker", Image: "example.invalid/worker:1", Env: []corev1.EnvVar{{Name: "PRIVATE", Value: "do-not-return"}}}}, RestartPolicy: corev1.RestartPolicyNever}},
			}}},
		},
		&batchv1.Job{ObjectMeta: metav1.ObjectMeta{Namespace: "apps", Name: "existing", UID: types.UID("job-uid"), ResourceVersion: "8"}, Status: batchv1.JobStatus{Active: 1}},
	)
	return m
}

func TestBatchJobStatusIsMetadataOnly(t *testing.T) {
	m := batchManager(t)
	response := request(m, http.MethodGet, "/v1/manage/batch/jobs/apps/existing", nil, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"active":1`) || strings.Contains(response.Body.String(), "template") {
		t.Fatalf("unexpected job summary: %s", response.Body.String())
	}
	response = request(m, http.MethodGet, "/v1/manage/batch/jobs/apps/missing", nil, nil)
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing Job status=%d", response.Code)
	}
}

func TestBatchJobStatusRequiresBearer(t *testing.T) {
	m := batchManager(t)
	server := startForwardTestServer(t, m)
	response, err := http.Get(server.URL + "/v1/manage/batch/jobs/apps/existing")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", response.StatusCode)
	}
}

func TestBatchCronJobRunRequiresExactIdentityAndUsesTemplate(t *testing.T) {
	m := batchManager(t)
	path := "/v1/manage/batch/cronjobs/apps/nightly/run"
	header := map[string]string{"X-KuchDesk-Confirm": "cronjobs/apps/nightly/run", "If-Match-UID": "cron-uid", "If-Match": "5"}
	var creates int
	m.Kube.(*kubefake.Clientset).PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		creates++
		job := action.(k8stesting.CreateAction).GetObject().(*batchv1.Job)
		if job.Namespace != "apps" || job.GenerateName != "nightly-manual-" || job.Spec.Template.Spec.Containers[0].Image != "example.invalid/worker:1" || job.Labels["kuchdesk.io/source-cronjob"] != "nightly" || job.Labels["team"] != "platform" || job.Annotations["purpose"] != "daily" {
			t.Errorf("wrong generated Job: %+v", job)
		}
		job = job.DeepCopy()
		job.Name, job.UID, job.ResourceVersion = "nightly-manual-abcde", types.UID("new-uid"), "1"
		return true, job, nil
	})
	for _, tc := range []struct {
		name   string
		header map[string]string
	}{
		{"no-confirmation", nil},
		{"wrong-uid", map[string]string{"X-KuchDesk-Confirm": "cronjobs/apps/nightly/run", "If-Match-UID": "wrong", "If-Match": "5"}},
		{"wrong-version", map[string]string{"X-KuchDesk-Confirm": "cronjobs/apps/nightly/run", "If-Match-UID": "cron-uid", "If-Match": "4"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := request(m, http.MethodPost, path, nil, tc.header)
			if response.Code != http.StatusConflict || creates != 0 {
				t.Fatalf("status=%d creates=%d", response.Code, creates)
			}
		})
	}
	response := request(m, http.MethodPost, path+"?dryRun=true", nil, header)
	if response.Code != http.StatusCreated || creates != 1 || strings.Contains(response.Body.String(), "do-not-return") {
		t.Fatalf("run status=%d creates=%d body=%s", response.Code, creates, response.Body.String())
	}
	var summary map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &summary); err != nil || summary["name"] != "nightly-manual-abcde" {
		t.Fatalf("run summary=%v err=%v", summary, err)
	}
}

func TestBatchJobDeleteRequiresPreconditionsAndForeground(t *testing.T) {
	m := batchManager(t)
	path := "/v1/manage/batch/jobs/apps/existing"
	var deletes int
	m.Kube.(*kubefake.Clientset).PrependReactor("delete", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deletes++
		options := action.(k8stesting.DeleteAction).GetDeleteOptions()
		if options.Preconditions == nil || *options.Preconditions.UID != types.UID("job-uid") || *options.Preconditions.ResourceVersion != "8" || options.PropagationPolicy == nil || *options.PropagationPolicy != metav1.DeletePropagationForeground || len(options.DryRun) == 0 {
			t.Errorf("unsafe delete options: %+v", options)
		}
		return true, nil, nil
	})
	response := request(m, http.MethodDelete, path, nil, nil)
	if response.Code != http.StatusConflict || deletes != 0 {
		t.Fatalf("unguarded delete status=%d calls=%d", response.Code, deletes)
	}
	header := map[string]string{"X-KuchDesk-Confirm": "jobs/apps/existing/delete", "If-Match-UID": "job-uid", "If-Match": "8"}
	response = request(m, http.MethodDelete, path+"?dryRun=true", nil, header)
	if response.Code != http.StatusAccepted || deletes != 1 || !strings.Contains(response.Body.String(), `"dryRun":true`) {
		t.Fatalf("delete status=%d calls=%d body=%s", response.Code, deletes, response.Body.String())
	}
}
