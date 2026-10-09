package management

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	errJobCapacity = errors.New("operation capacity reached")
	errJobMissing  = errors.New("operation not found")
)

type workloadJob struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	cancel     context.CancelFunc
}

type workloadJobKey struct {
	gate chan struct{}
	refs int
}

// WorkloadJobs keeps only bounded, process-local operation state. It does not
// persist or retry mutations after agent restart.
type WorkloadJobs struct {
	mu     sync.Mutex
	ctx    context.Context
	stop   context.CancelFunc
	slots  chan struct{}
	jobs   map[string]*workloadJob
	order  []string
	keys   map[string]*workloadJobKey
	wg     sync.WaitGroup
	closed bool
}

func NewWorkloadJobs(parent context.Context) *WorkloadJobs {
	ctx, stop := context.WithCancel(parent)
	return &WorkloadJobs{ctx: ctx, stop: stop, slots: make(chan struct{}, 4), jobs: make(map[string]*workloadJob), keys: make(map[string]*workloadJobKey)}
}

func (q *WorkloadJobs) Submit(key string, run func(context.Context) int) (string, error) {
	if run == nil {
		return "", errors.New("operation function is required")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", errors.New("operation ID generation failed")
	}
	id := hex.EncodeToString(random[:])
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || q.ctx.Err() != nil {
		return "", context.Canceled
	}
	if len(q.jobs) >= 64 {
		keep := q.order[:0]
		for _, candidate := range q.order {
			job := q.jobs[candidate]
			if len(q.jobs) >= 64 && job.State != "queued" && job.State != "running" {
				delete(q.jobs, candidate)
				continue
			}
			keep = append(keep, candidate)
		}
		q.order = keep
	}
	if len(q.jobs) >= 64 {
		return "", errJobCapacity
	}
	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Second)
	q.jobs[id] = &workloadJob{ID: id, State: "queued", cancel: cancel}
	q.order = append(q.order, id)
	q.wg.Add(1)
	go q.execute(ctx, cancel, id, key, run)
	return id, nil
}

func (q *WorkloadJobs) execute(ctx context.Context, cancel context.CancelFunc, id, key string, run func(context.Context) int) {
	defer q.wg.Done()
	defer cancel()
	defer func() {
		if recover() != nil {
			q.finish(id, nil, 500)
		}
	}()
	select {
	case q.slots <- struct{}{}:
		defer func() { <-q.slots }()
	case <-ctx.Done():
		q.finish(id, ctx.Err(), 0)
		return
	}
	if err := q.acquire(ctx, key); err != nil {
		q.finish(id, err, 0)
		return
	}
	defer q.release(key)
	if err := ctx.Err(); err != nil {
		q.finish(id, err, 0)
		return
	}
	q.mu.Lock()
	q.jobs[id].State = "running"
	q.mu.Unlock()
	status := run(ctx)
	q.finish(id, ctx.Err(), status)
}

func (q *WorkloadJobs) acquire(ctx context.Context, key string) error {
	q.mu.Lock()
	item := q.keys[key]
	if item == nil {
		item = &workloadJobKey{gate: make(chan struct{}, 1)}
		item.gate <- struct{}{}
		q.keys[key] = item
	}
	item.refs++
	q.mu.Unlock()
	select {
	case <-item.gate:
		if err := ctx.Err(); err != nil {
			q.releaseReference(key)
			item.gate <- struct{}{}
			return err
		}
		return nil
	case <-ctx.Done():
		q.releaseReference(key)
		return ctx.Err()
	}
}

func (q *WorkloadJobs) release(key string) {
	q.mu.Lock()
	item := q.keys[key]
	item.refs--
	if item.refs == 0 {
		delete(q.keys, key)
	}
	q.mu.Unlock()
	item.gate <- struct{}{}
}

func (q *WorkloadJobs) releaseReference(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	item := q.keys[key]
	item.refs--
	if item.refs == 0 {
		delete(q.keys, key)
	}
}

func (q *WorkloadJobs) finish(id string, err error, status int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	job := q.jobs[id]
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		job.State = "timed_out"
	case errors.Is(err, context.Canceled):
		job.State = "canceled"
	case status >= 200 && status < 300:
		job.State = "succeeded"
	default:
		job.State = "failed"
	}
	job.HTTPStatus = status
}

func (q *WorkloadJobs) Status(id string) (workloadJob, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	job := q.jobs[id]
	if job == nil {
		return workloadJob{}, errJobMissing
	}
	return workloadJob{ID: job.ID, State: job.State, HTTPStatus: job.HTTPStatus}, nil
}

func (q *WorkloadJobs) Cancel(id string) (workloadJob, error) {
	q.mu.Lock()
	job := q.jobs[id]
	if job == nil {
		q.mu.Unlock()
		return workloadJob{}, errJobMissing
	}
	status := workloadJob{ID: job.ID, State: job.State, HTTPStatus: job.HTTPStatus}
	q.mu.Unlock()
	if status.State == "queued" || status.State == "running" {
		job.cancel()
	}
	return status, nil
}

func (q *WorkloadJobs) Close() {
	q.mu.Lock()
	q.closed = true
	q.stop()
	q.mu.Unlock()
	q.wg.Wait()
}
