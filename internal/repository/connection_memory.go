package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

// MemoryConnectionRepository is a thread-safe in-memory connection store.
type MemoryConnectionRepository struct {
	mu    sync.RWMutex
	conns map[string]*upal.Connection
}

func NewMemoryConnectionRepository() *MemoryConnectionRepository {
	return &MemoryConnectionRepository{conns: make(map[string]*upal.Connection)}
}

func (r *MemoryConnectionRepository) Create(_ context.Context, conn *upal.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.conns[conn.ID]; exists {
		return fmt.Errorf("connection %q already exists", conn.ID)
	}
	r.conns[conn.ID] = conn
	return nil
}

func (r *MemoryConnectionRepository) Get(_ context.Context, id string) (*upal.Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	conn, ok := r.conns[id]
	if !ok {
		return nil, fmt.Errorf("connection %q not found", id)
	}
	return conn, nil
}

func (r *MemoryConnectionRepository) List(_ context.Context) ([]*upal.Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*upal.Connection, 0, len(r.conns))
	for _, c := range r.conns {
		result = append(result, c)
	}
	return result, nil
}

func (r *MemoryConnectionRepository) Update(_ context.Context, conn *upal.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.conns[conn.ID]; !exists {
		return fmt.Errorf("connection %q not found", conn.ID)
	}
	r.conns[conn.ID] = conn
	return nil
}

func (r *MemoryConnectionRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.conns[id]; !exists {
		return fmt.Errorf("connection %q not found", id)
	}
	delete(r.conns, id)
	return nil
}
