package stream

import (
	"testing"
	"time"
)

func TestBrokerReplayAndLiveDelivery(t *testing.T) {
	t.Parallel()

	broker := NewBroker("homelab", 2)
	first, err := broker.Publish("snapshot", map[string]int{"generation": 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := broker.Publish("snapshot", map[string]int{"generation": 2})
	if err != nil {
		t.Fatal(err)
	}

	replay, live, gap, cancel := broker.Subscribe(first.ID)
	defer cancel()
	if gap {
		t.Fatal("unexpected history gap")
	}
	if len(replay) != 1 || replay[0].ID != second.ID {
		t.Fatalf("unexpected replay: %#v", replay)
	}

	third, err := broker.Publish("snapshot", map[string]int{"generation": 3})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-live:
		if event.ID != third.ID || event.ClusterID != "homelab" {
			t.Fatalf("unexpected live event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live event")
	}
}

func TestBrokerReportsHistoryGap(t *testing.T) {
	t.Parallel()

	broker := NewBroker("homelab", 2)
	for generation := 1; generation <= 4; generation++ {
		if _, err := broker.Publish("snapshot", generation); err != nil {
			t.Fatal(err)
		}
	}
	_, _, gap, cancel := broker.Subscribe(1)
	defer cancel()
	if !gap {
		t.Fatal("expected history gap")
	}
}
