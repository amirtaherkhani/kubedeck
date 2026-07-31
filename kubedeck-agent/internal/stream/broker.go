package stream

import (
	"encoding/json"
	"sync"
	"time"
)

type Event struct {
	ID        uint64          `json:"id"`
	Name      string          `json:"name"`
	ClusterID string          `json:"clusterId"`
	SentAt    time.Time       `json:"sentAt"`
	Data      json.RawMessage `json:"data"`
}

type Broker struct {
	mu          sync.Mutex
	clusterID   string
	historySize int
	nextID      uint64
	history     []Event
	subscribers map[chan Event]struct{}
}

func NewBroker(clusterID string, historySize int) *Broker {
	return &Broker{
		clusterID:   clusterID,
		historySize: historySize,
		subscribers: make(map[chan Event]struct{}),
	}
}

func (b *Broker) Publish(name string, payload any) (Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	event := Event{
		ID:        b.nextID,
		Name:      name,
		ClusterID: b.clusterID,
		SentAt:    time.Now().UTC(),
		Data:      data,
	}
	b.history = append(b.history, event)
	if len(b.history) > b.historySize {
		b.history = append([]Event(nil), b.history[len(b.history)-b.historySize:]...)
	}

	for subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
			delete(b.subscribers, subscriber)
			close(subscriber)
		}
	}
	return event, nil
}

func (b *Broker) Subscribe(afterID uint64) (
	replay []Event,
	live <-chan Event,
	historyGap bool,
	latestID uint64,
	cancel func(),
) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if afterID > 0 && len(b.history) > 0 {
		oldest := b.history[0].ID
		historyGap = afterID+1 < oldest
		for _, event := range b.history {
			if event.ID > afterID {
				replay = append(replay, event)
			}
		}
	}

	channel := make(chan Event, 16)
	b.subscribers[channel] = struct{}{}
	latestID = b.nextID

	var once sync.Once
	cancel = func() {
		once.Do(func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if _, exists := b.subscribers[channel]; exists {
				delete(b.subscribers, channel)
				close(channel)
			}
		})
	}
	return replay, channel, historyGap, latestID, cancel
}

func (b *Broker) LatestID() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.nextID
}
