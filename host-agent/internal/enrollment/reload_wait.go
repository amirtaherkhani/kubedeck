package enrollment

import (
	"context"
	"time"
)

// waitForNextCycle reloads scheduling policy while waiting. All deadlines share
// the original cycle-end time, so polling never postpones a cycle indefinitely.
func waitForNextCycle(ctx context.Context, load func() (Policy, error), fallback Policy, failures int, retryAfter time.Duration) bool {
	start := time.Now()
	jitter := time.Duration(start.UnixNano() % int64(time.Second))
	for {
		if ctx.Err() != nil {
			return false
		}
		p, err := load()
		if err != nil {
			p = fallback
		}
		d := delay(p, failures)
		if retryAfter > d {
			d = retryAfter
		}
		remaining := time.Until(start.Add(d + jitter))
		if remaining <= 0 {
			return ctx.Err() == nil
		}
		if remaining > time.Second {
			remaining = time.Second
		}
		timer := time.NewTimer(remaining)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
}
