package management

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func waitWorkloadJob(t *testing.T, jobs *WorkloadJobs, id, want string) workloadJob {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		job, err := jobs.Status(id)
		if err != nil {
			t.Fatal(err)
		}
		if job.State == want {
			return job
		}
		select {
		case <-deadline:
			t.Fatalf("job %s stayed %s; wanted %s", id, job.State, want)
		case <-time.After(time.Millisecond):
		}
	}
}

func TestAsyncScaleUsesExistingGuardsAndReportsCompletion(t *testing.T) {
	m := testManager(t)
	m.Jobs = NewWorkloadJobs(context.Background())
	t.Cleanup(m.Jobs.Close)
	m.Kube.(*kubefake.Clientset).PrependReactor("get", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "apps", ResourceVersion: "7"}}, nil
	})
	m.Kube.(*kubefake.Clientset).PrependReactor("update", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, action.(k8stesting.UpdateAction).GetObject(), nil
	})
	path := "/v1/manage/workloads/deployments/apps/sample/scale/jobs"
	body := []byte(`{"replicas":2,"resourceVersion":"7"}`)
	if bad := request(m, http.MethodPost, path, body, nil); bad.Code != http.StatusConflict {
		t.Fatalf("missing confirmation returned %d", bad.Code)
	}
	response := request(m, http.MethodPost, path, body, map[string]string{"X-KuchDesk-Confirm": "deployments/apps/sample", "Authorization": "Bearer should-not-be-retained"})
	if response.Code != http.StatusAccepted {
		t.Fatalf("async scale returned %d: %s", response.Code, response.Body.String())
	}
	var accepted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &accepted); err != nil || accepted.ID == "" {
		t.Fatalf("invalid job response: %s", response.Body.String())
	}
	job := waitWorkloadJob(t, m.Jobs, accepted.ID, "succeeded")
	if job.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected job result: %+v", job)
	}
	status := request(m, http.MethodGet, "/v1/manage/jobs/"+accepted.ID, nil, nil)
	if status.Code != http.StatusOK || !bytes.Contains(status.Body.Bytes(), []byte(`"state":"succeeded"`)) || bytes.Contains(status.Body.Bytes(), []byte("should-not-be-retained")) {
		t.Fatalf("unsafe status response: %d %s", status.Code, status.Body.String())
	}
	stale := request(m, http.MethodPost, path, []byte(`{"replicas":3,"resourceVersion":"6"}`), map[string]string{"X-KuchDesk-Confirm": "deployments/apps/sample"})
	if stale.Code != http.StatusAccepted {
		t.Fatalf("stale request was not queued for version checking: %d", stale.Code)
	}
	if err := json.Unmarshal(stale.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if staleJob := waitWorkloadJob(t, m.Jobs, accepted.ID, "failed"); staleJob.HTTPStatus != http.StatusConflict {
		t.Fatalf("stale version did not conflict: %+v", staleJob)
	}
}

func TestWorkloadJobCancellationIsBounded(t *testing.T) {
	jobs := NewWorkloadJobs(context.Background())
	defer jobs.Close()
	started := make(chan struct{})
	id, err := jobs.Submit("deployment/apps/test", func(ctx context.Context) int {
		close(started)
		<-ctx.Done()
		return http.StatusRequestTimeout
	})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if _, err := jobs.Cancel(id); err != nil {
		t.Fatal(err)
	}
	waitWorkloadJob(t, jobs, id, "canceled")
}

func TestWorkloadJobsSerializeConflictsAndRunIndependentActions(t *testing.T) {
	jobs := NewWorkloadJobs(context.Background())
	defer jobs.Close()
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	first, err := jobs.Submit("deployments/apps/a", func(ctx context.Context) int {
		close(firstStarted)
		select {
		case <-releaseFirst:
			return http.StatusOK
		case <-ctx.Done():
			return http.StatusRequestTimeout
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	<-firstStarted
	secondStarted := make(chan struct{})
	second, err := jobs.Submit("deployments/apps/a", func(context.Context) int {
		close(secondStarted)
		return http.StatusOK
	})
	if err != nil {
		t.Fatal(err)
	}
	independentStarted := make(chan struct{})
	independent, err := jobs.Submit("deployments/apps/b", func(context.Context) int {
		close(independentStarted)
		return http.StatusOK
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-independentStarted:
	case <-time.After(time.Second):
		t.Fatal("independent action was blocked")
	}
	select {
	case <-secondStarted:
		t.Fatal("conflicting action ran before the first finished")
	case <-time.After(20 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		t.Fatal("conflicting action did not resume")
	}
	for _, id := range []string{first, second, independent} {
		waitWorkloadJob(t, jobs, id, "succeeded")
	}
}

func TestWorkloadJobPanicIsContained(t *testing.T) {
	jobs := NewWorkloadJobs(context.Background())
	defer jobs.Close()
	id, err := jobs.Submit("a", func(context.Context) int {
		panic("sensitive internal value")
	})
	if err != nil {
		t.Fatal(err)
	}
	job := waitWorkloadJob(t, jobs, id, "failed")
	if job.HTTPStatus != http.StatusInternalServerError {
		t.Fatalf("unexpected panic result: %+v", job)
	}
}
