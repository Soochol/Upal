package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventBus_Subscribe_Publish(t *testing.T) {
	bus := NewEventBus()
	var received []Event
	var mu sync.Mutex
	bus.Subscribe(func(e Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	})
	bus.Publish(Event{ID: "ev-1", Type: EventNodeStarted, Timestamp: time.Now()})
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("received: got %d, want 1", len(received))
	}
	if received[0].ID != "ev-1" {
		t.Errorf("event ID: got %q, want ev-1", received[0].ID)
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus()
	var count1, count2 int
	var mu sync.Mutex
	bus.Subscribe(func(e Event) { mu.Lock(); count1++; mu.Unlock() })
	bus.Subscribe(func(e Event) { mu.Lock(); count2++; mu.Unlock() })
	bus.Publish(Event{ID: "ev-1", Type: EventNodeStarted, Timestamp: time.Now()})
	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Errorf("counts: got %d/%d, want 1/1", count1, count2)
	}
}

func TestEventBus_Channel(t *testing.T) {
	bus := NewEventBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := bus.Channel(ctx, 10)
	bus.Publish(Event{ID: "ev-1", Type: EventNodeStarted, Timestamp: time.Now()})
	select {
	case ev := <-ch:
		if ev.ID != "ev-1" {
			t.Errorf("event ID: got %q, want ev-1", ev.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}
