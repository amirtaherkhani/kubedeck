package enrollment

import (
	"context"
	"testing"
)

func TestRunAlreadyCanceledDoesNotStartCycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// No backend/store is configured: starting a cycle would be invalid.
	c := &Controller{}
	c.Run(ctx, func(Report) { t.Fatal("canceled controller emitted a cycle") })
}

func TestRunCancellationAfterReportDoesNotStartAnotherCycle(t *testing.T) {
	c, _ := newTestController(t, testPolicy())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reports := 0
	c.Run(ctx, func(r Report) {
		reports++
		if r.Status != "ready" {
			t.Fatalf("unexpected cycle: %s", r.Status)
		}
		cancel()
	})
	if reports != 1 {
		t.Fatalf("got %d cycles after cancellation", reports)
	}
}
