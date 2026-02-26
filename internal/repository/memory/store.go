package memory

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("not found")

// Store is a generic thread-safe in-memory key-value store.
type Store[V any] struct {
	mu      sync.RWMutex
	data    map[string]V
	keyFunc func(V) string
}

func New[V any](keyFunc func(V) string) *Store[V] {
	return &Store[V]{
		data:    make(map[string]V),
		keyFunc: keyFunc,
	}
}

func (s *Store[V]) Set(_ context.Context, v V) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[s.keyFunc(v)] = v
	return nil
}

func (s *Store[V]) Get(_ context.Context, key string) (V, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	if !ok {
		var zero V
		return zero, ErrNotFound
	}
	return v, nil
}

func (s *Store[V]) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; !ok {
		return ErrNotFound
	}
	delete(s.data, key)
	return nil
}

func (s *Store[V]) All(_ context.Context) ([]V, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]V, 0, len(s.data))
	for _, v := range s.data {
		out = append(out, v)
	}
	return out, nil
}

func (s *Store[V]) Filter(_ context.Context, pred func(V) bool) ([]V, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []V
	for _, v := range s.data {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out, nil
}

func (s *Store[V]) Has(_ context.Context, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[key]
	return ok
}
