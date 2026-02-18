package engine

import (
	"context"
	"sync"
)

type EventHandler func(Event)

type EventBus struct {
	mu       sync.RWMutex
	handlers map[int]EventHandler
	nextID   int
}

func NewEventBus() *EventBus {
	return &EventBus{handlers: make(map[int]EventHandler)}
}

// Subscribe registers a handler and returns an unsubscribe function.
func (b *EventBus) Subscribe(handler EventHandler) func() {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.handlers[id] = handler
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.handlers, id)
	}
}

func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	handlers := make([]EventHandler, 0, len(b.handlers))
	for _, h := range b.handlers {
		handlers = append(handlers, h)
	}
	b.mu.RUnlock()
	for _, h := range handlers {
		h(event)
	}
}

func (b *EventBus) Channel(ctx context.Context, bufSize int) <-chan Event {
	ch := make(chan Event, bufSize)
	unsub := b.Subscribe(func(e Event) {
		select {
		case ch <- e:
		case <-ctx.Done():
		default:
		}
	})
	go func() {
		<-ctx.Done()
		unsub()
		close(ch)
	}()
	return ch
}
