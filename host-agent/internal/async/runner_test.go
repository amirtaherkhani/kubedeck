package async

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func waitState(t *testing.T, r *Runner, id string, want State) Snapshot {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		status, err := r.Status(id)
		if err != nil {
			t.Fatal(err)
		}
		if status.State == want {
			return status
		}
		select {
		case <-deadline:
			t.Fatalf("operation %s stayed in %s; wanted %s", id, status.State, want)
		case <-time.After(time.Millisecond):
		}
	}
}

func TestIndependentOperationsRunInParallel(t *testing.T) {
	r := NewRunner(context.Background(), 2, 4)
	defer r.Close()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	task := func(ctx context.Context) (any, error) {
		started <- struct{}{}
		select {
		case <-release:
			return "done", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	first, err := r.Submit("resource-a", time.Second, task)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.Submit("resource-b", time.Second, task)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("independent operations did not run concurrently")
		}
	}
	close(release)
	for _, id := range []string{first, second} {
		if got := waitState(t, r, id, Succeeded); got.Result != "done" {
			t.Fatalf("unexpected result: %+v", got)
		}
	}
}

func TestSameKeySerializesAndCancellationReleasesCapacity(t *testing.T) {
	r := NewRunner(context.Background(), 2, 3)
	defer r.Close()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var running atomic.Int32
	task := func(ctx context.Context) (any, error) {
		if running.Add(1) != 1 {
			t.Error("same-key operations overlapped")
		}
		defer running.Add(-1)
		entered <- struct{}{}
		select {
		case <-release:
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	first, err := r.Submit("same", time.Second, task)
	if err != nil {
		t.Fatal(err)
	}
	<-entered
	second, err := r.Submit("same", time.Second, task)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
		t.Fatal("same-key operation started before first completed")
	case <-time.After(20 * time.Millisecond):
	}
	if _, err := r.Cancel(first); err != nil {
		t.Fatal(err)
	}
	waitState(t, r, first, Canceled)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("waiting operation did not resume")
	}
	close(release)
	waitState(t, r, second, Succeeded)
}

func TestBoundedRecordsAndQueuedCancellation(t *testing.T) {
	r := NewRunner(context.Background(), 1, 2)
	defer r.Close()
	started := make(chan struct{})
	first, err := r.Submit("", time.Second, func(ctx context.Context) (any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	second, err := r.Submit("", time.Second, func(context.Context) (any, error) {
		t.Error("canceled queued operation ran")
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Submit("", time.Second, func(context.Context) (any, error) { return nil, nil }); !errors.Is(err, ErrCapacity) {
		t.Fatalf("expected capacity error, got %v", err)
	}
	if _, err := r.Cancel(second); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Cancel(first); err != nil {
		t.Fatal(err)
	}
	waitState(t, r, first, Canceled)
	waitState(t, r, second, Canceled)
	if _, err := r.Submit("", time.Second, func(context.Context) (any, error) { return nil, nil }); err != nil {
		t.Fatalf("completed records should be pruned for a new operation: %v", err)
	}
}

func TestDeadlineStopsOperation(t *testing.T) {
	r := NewRunner(context.Background(), 1, 2)
	defer r.Close()
	id, err := r.Submit("", 10*time.Millisecond, func(ctx context.Context) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	waitState(t, r, id, TimedOut)
}

func TestPanickingOperationIsContained(t *testing.T) {
	r := NewRunner(context.Background(), 1, 2)
	defer r.Close()
	id, err := r.Submit("", time.Second, func(context.Context) (any, error) {
		panic("sensitive internal value")
	})
	if err != nil {
		t.Fatal(err)
	}
	job := waitState(t, r, id, Failed)
	if job.Error != "operation_failed" {
		t.Fatalf("panic detail leaked: %q", job.Error)
	}
}
