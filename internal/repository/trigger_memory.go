package repository

import (
	"context"
	"sync"

	"github.com/soochol/upal/internal/upal"
)

// MemoryTriggerRepository stores triggers in memory.
type MemoryTriggerRepository struct {
	mu       sync.RWMutex
	triggers map[string]*upal.Trigger
}

func NewMemoryTriggerRepository() *MemoryTriggerRepository {
	return &MemoryTriggerRepository{
		triggers: make(map[string]*upal.Trigger),
	}
}

func (r *MemoryTriggerRepository) Create(_ context.Context, trigger *upal.Trigger) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.triggers[trigger.ID] = trigger
	return nil
}

func (r *MemoryTriggerRepository) Get(_ context.Context, id string) (*upal.Trigger, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.triggers[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (r *MemoryTriggerRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.triggers[id]; !ok {
		return ErrNotFound
	}
	delete(r.triggers, id)
	return nil
}

func (r *MemoryTriggerRepository) ListByWorkflow(_ context.Context, workflowName string) ([]*upal.Trigger, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*upal.Trigger
	for _, t := range r.triggers {
		if t.WorkflowName == workflowName {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *MemoryTriggerRepository) ListByPipeline(_ context.Context, pipelineID string) ([]*upal.Trigger, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*upal.Trigger
	for _, t := range r.triggers {
		if t.PipelineID == pipelineID {
			result = append(result, t)
		}
	}
	return result, nil
}
