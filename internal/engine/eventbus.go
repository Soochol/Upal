package engine

import (
	"context"
	"sync"
)

type EventHandler func(Event)

type EventBus struct {
	mu       sync.RWMutex
	handlers []EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

func (b *EventBus) Subscribe(handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, handler)
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.handlers))
	copy(handlers, b.handlers)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(event)
	}
}

func (b *EventBus) Channel(ctx context.Context, bufSize int) <-chan Event {
	ch := make(chan Event, bufSize)
	b.Subscribe(func(e Event) {
		select {
		case ch <- e:
		case <-ctx.Done():
		default:
		}
	})
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch
}
