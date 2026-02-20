package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

// MemoryRepository is a thread-safe in-memory WorkflowRepository.
type MemoryRepository struct {
	mu        sync.RWMutex
	workflows map[string]*upal.WorkflowDefinition
}

// NewMemory creates an empty in-memory repository.
func NewMemory() *MemoryRepository {
	return &MemoryRepository{workflows: make(map[string]*upal.WorkflowDefinition)}
}

func (r *MemoryRepository) Create(_ context.Context, wf *upal.WorkflowDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.workflows[wf.Name] = wf
	return nil
}

func (r *MemoryRepository) Get(_ context.Context, name string) (*upal.WorkflowDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	wf, ok := r.workflows[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return wf, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]*upal.WorkflowDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*upal.WorkflowDefinition, 0, len(r.workflows))
	for _, wf := range r.workflows {
		result = append(result, wf)
	}
	return result, nil
}

func (r *MemoryRepository) Update(_ context.Context, name string, wf *upal.WorkflowDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if name != wf.Name {
		delete(r.workflows, name)
	}
	r.workflows[wf.Name] = wf
	return nil
}

func (r *MemoryRepository) Delete(_ context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workflows, name)
	return nil
}
