package services

import (
	"sync"

	"github.com/soochol/upal/internal/upal"
)

// ExecutionRegistry tracks active workflow executions.
type ExecutionRegistry struct {
	mu      sync.RWMutex
	handles map[string]*upal.ExecutionHandle
}

func NewExecutionRegistry() *ExecutionRegistry {
	return &ExecutionRegistry{handles: make(map[string]*upal.ExecutionHandle)}
}

// Register adds an execution handle. Returns the handle for use by the runner.
func (r *ExecutionRegistry) Register(runID string) *upal.ExecutionHandle {
	h := upal.NewExecutionHandle(runID)
	r.mu.Lock()
	r.handles[runID] = h
	r.mu.Unlock()
	return h
}

// Get retrieves a handle by run ID.
func (r *ExecutionRegistry) Get(runID string) (*upal.ExecutionHandle, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handles[runID]
	return h, ok
}

// Unregister removes a completed execution.
func (r *ExecutionRegistry) Unregister(runID string) {
	r.mu.Lock()
	delete(r.handles, runID)
	r.mu.Unlock()
}
