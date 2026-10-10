package enrollment

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

func TestReloadWaitAdjustsDeadline(t *testing.T) {
	for _, tc := range []struct {
		name             string
		initial, changed int
		changeAt, want   time.Duration
	}{
		{"shorten", 120, 30, 5 * time.Second, 30 * time.Second},
		{"extend", 30, 90, 5 * time.Second, 90 * time.Second},
		{"already_due", 120, 30, 40 * time.Second, 40 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				time.Sleep(125 * time.Millisecond) // Exercise nonzero fixed jitter with virtual time.
				start := time.Now()
				p := Policy{IntervalSeconds: tc.initial}
				load := func() (Policy, error) {
					q := p
					if time.Since(start) >= tc.changeAt {
						q.IntervalSeconds = tc.changed
					}
					return q, nil
				}
				if !waitForNextCycle(context.Background(), load, p, 0, 0) {
					t.Fatal("unexpected cancellation")
				}
				elapsed := time.Since(start)
				if elapsed < tc.want || elapsed > tc.want+time.Second {
					t.Fatalf("deadline elapsed %v, want %v through +1s", elapsed, tc.want)
				}
			})
		})
	}
}

func TestReloadWaitRetryFloorAndBackoff(t *testing.T) {
	for _, tc := range []struct {
		name        string
		failures    int
		retry, want time.Duration
	}{
		{"retry_after", 0, 80 * time.Second, 80 * time.Second},
		{"backoff", 2, 0, 120 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				time.Sleep(125 * time.Millisecond) // Exercise nonzero fixed jitter with virtual time.
				start := time.Now()
				p := Policy{IntervalSeconds: 120}
				load := func() (Policy, error) {
					q := p
					if time.Since(start) >= 5*time.Second {
						q.IntervalSeconds = 30
					}
					return q, nil
				}
				if !waitForNextCycle(context.Background(), load, p, tc.failures, tc.retry) {
					t.Fatal("unexpected cancellation")
				}
				if elapsed := time.Since(start); elapsed < tc.want || elapsed > tc.want+time.Second {
					t.Fatalf("floor/backoff violated: %v", elapsed)
				}
			})
		})
	}
}

func TestReloadWaitInvalidPolicyAndCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(125 * time.Millisecond) // Exercise nonzero fixed jitter with virtual time.
		start := time.Now()
		p := Policy{IntervalSeconds: 30}
		if !waitForNextCycle(context.Background(), func() (Policy, error) { return Policy{}, ErrDenied }, p, 0, 0) {
			t.Fatal("unexpected cancellation")
		}
		if elapsed := time.Since(start); elapsed < 30*time.Second || elapsed > 31*time.Second {
			t.Fatalf("fallback deadline: %v", elapsed)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		time.Sleep(125 * time.Millisecond) // Exercise nonzero fixed jitter with virtual time.
		start := time.Now()
		p := Policy{IntervalSeconds: 120}
		go func() { time.Sleep(125 * time.Millisecond); cancel() }()
		if waitForNextCycle(ctx, func() (Policy, error) { return p, nil }, p, 0, 0) {
			t.Fatal("canceled wait returned due")
		}
		if time.Since(start) != 125*time.Millisecond {
			t.Fatal("cancellation delayed")
		}
	})
}
