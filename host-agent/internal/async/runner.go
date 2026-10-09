package async

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrCapacity = errors.New("agent operation capacity reached")
	ErrNotFound = errors.New("agent operation not found")
)

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Succeeded State = "succeeded"
	Failed    State = "failed"
	Canceled  State = "canceled"
	TimedOut  State = "timed_out"
)

type Snapshot struct {
	ID     string `json:"id"`
	State  State  `json:"state"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type operation struct {
	Snapshot
	cancel context.CancelFunc
}

type keySlot struct {
	gate chan struct{}
	refs int
}

// Runner is a bounded in-process executor, not a durable queue. Independent
// operations may run concurrently; operations with the same nonempty key are
// serialized. Its state is lost when the process exits.
type Runner struct {
	mu         sync.Mutex
	ctx        context.Context
	cancelAll  context.CancelFunc
	active     chan struct{}
	maxRecords int
	jobs       map[string]*operation
	order      []string
	keys       map[string]*keySlot
	wg         sync.WaitGroup
	closed     bool
}

func NewRunner(parent context.Context, maxConcurrent, maxRecords int) *Runner {
	if maxConcurrent < 1 || maxRecords < maxConcurrent {
		panic("invalid agent operation limits")
	}
	ctx, cancel := context.WithCancel(parent)
	return &Runner{ctx: ctx, cancelAll: cancel, active: make(chan struct{}, maxConcurrent), maxRecords: maxRecords, jobs: make(map[string]*operation), keys: make(map[string]*keySlot)}
}

func (r *Runner) Submit(key string, timeout time.Duration, task func(context.Context) (any, error)) (string, error) {
	if task == nil || timeout <= 0 {
		return "", errors.New("operation task and positive timeout are required")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", errors.New("operation ID generation failed")
	}
	id := hex.EncodeToString(random[:])
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.ctx.Err() != nil {
		return "", context.Canceled
	}
	if len(r.jobs) >= r.maxRecords {
		r.pruneCompleted()
	}
	if len(r.jobs) >= r.maxRecords {
		return "", ErrCapacity
	}
	ctx, cancel := context.WithTimeout(r.ctx, timeout)
	r.jobs[id] = &operation{Snapshot: Snapshot{ID: id, State: Queued}, cancel: cancel}
	r.order = append(r.order, id)
	r.wg.Add(1)
	go r.run(ctx, cancel, id, key, task)
	return id, nil
}

func (r *Runner) run(ctx context.Context, cancel context.CancelFunc, id, key string, task func(context.Context) (any, error)) {
	defer r.wg.Done()
	defer cancel()
	defer func() {
		if recover() != nil {
			r.finish(id, nil, errors.New("operation_failed"))
		}
	}()
	select {
	case r.active <- struct{}{}:
		defer func() { <-r.active }()
	case <-ctx.Done():
		r.finish(id, nil, ctx.Err())
		return
	}
	if key != "" {
		if err := r.acquireKey(ctx, key); err != nil {
			r.finish(id, nil, err)
			return
		}
		defer r.releaseKey(key)
	}
	if err := ctx.Err(); err != nil {
		r.finish(id, nil, err)
		return
	}
	r.mu.Lock()
	r.jobs[id].State = Running
	r.mu.Unlock()
	result, err := task(ctx)
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	r.finish(id, result, err)
}

func (r *Runner) acquireKey(ctx context.Context, key string) error {
	r.mu.Lock()
	slot := r.keys[key]
	if slot == nil {
		slot = &keySlot{gate: make(chan struct{}, 1)}
		slot.gate <- struct{}{}
		r.keys[key] = slot
	}
	slot.refs++
	r.mu.Unlock()
	select {
	case <-slot.gate:
		if err := ctx.Err(); err != nil {
			r.releaseKeyReference(key)
			slot.gate <- struct{}{}
			return err
		}
		return nil
	case <-ctx.Done():
		r.releaseKeyReference(key)
		return ctx.Err()
	}
}

func (r *Runner) releaseKey(key string) {
	r.mu.Lock()
	slot := r.keys[key]
	slot.refs--
	if slot.refs == 0 {
		delete(r.keys, key)
	}
	r.mu.Unlock()
	slot.gate <- struct{}{}
}

func (r *Runner) releaseKeyReference(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	slot := r.keys[key]
	slot.refs--
	if slot.refs == 0 {
		delete(r.keys, key)
	}
}

func (r *Runner) finish(id string, result any, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[id]
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		job.State = TimedOut
	case errors.Is(err, context.Canceled):
		job.State = Canceled
	case err != nil:
		job.State = Failed
		job.Error = err.Error()
	default:
		job.State = Succeeded
		job.Result = result
	}
}

func (r *Runner) Status(id string) (Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobs[id]
	if job == nil {
		return Snapshot{}, ErrNotFound
	}
	return job.Snapshot, nil
}

func (r *Runner) Cancel(id string) (Snapshot, error) {
	r.mu.Lock()
	job := r.jobs[id]
	if job == nil {
		r.mu.Unlock()
		return Snapshot{}, ErrNotFound
	}
	snapshot := job.Snapshot
	r.mu.Unlock()
	if snapshot.State == Queued || snapshot.State == Running {
		job.cancel()
	}
	return snapshot, nil
}

func (r *Runner) pruneCompleted() {
	keep := r.order[:0]
	for _, id := range r.order {
		job := r.jobs[id]
		if len(r.jobs) >= r.maxRecords && job.State != Queued && job.State != Running {
			delete(r.jobs, id)
			continue
		}
		keep = append(keep, id)
	}
	r.order = keep
}

func (r *Runner) Close() {
	r.mu.Lock()
	r.closed = true
	r.cancelAll()
	r.mu.Unlock()
	r.wg.Wait()
}
