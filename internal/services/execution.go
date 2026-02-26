package services

import (
	"sync"

	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

var _ ports.ExecutionRegistryPort = (*ExecutionRegistry)(nil)

type ExecutionRegistry struct {
	mu      sync.RWMutex
	handles map[string]*upal.ExecutionHandle
}

func NewExecutionRegistry() *ExecutionRegistry {
	return &ExecutionRegistry{handles: make(map[string]*upal.ExecutionHandle)}
}

func (r *ExecutionRegistry) Register(runID string) *upal.ExecutionHandle {
	h := upal.NewExecutionHandle(runID)
	r.mu.Lock()
	r.handles[runID] = h
	r.mu.Unlock()
	return h
}

func (r *ExecutionRegistry) Get(runID string) (*upal.ExecutionHandle, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handles[runID]
	return h, ok
}

func (r *ExecutionRegistry) Unregister(runID string) {
	r.mu.Lock()
	delete(r.handles, runID)
	r.mu.Unlock()
}
