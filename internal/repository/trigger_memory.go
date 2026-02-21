package repository

import (
	"context"
	"errors"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// MemoryTriggerRepository stores triggers in memory.
type MemoryTriggerRepository struct {
	store *memstore.Store[*upal.Trigger]
}

func NewMemoryTriggerRepository() *MemoryTriggerRepository {
	return &MemoryTriggerRepository{
		store: memstore.New(func(t *upal.Trigger) string { return t.ID }),
	}
}

func (r *MemoryTriggerRepository) Create(ctx context.Context, trigger *upal.Trigger) error {
	return r.store.Set(ctx, trigger)
}

func (r *MemoryTriggerRepository) Get(ctx context.Context, id string) (*upal.Trigger, error) {
	t, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *MemoryTriggerRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (r *MemoryTriggerRepository) ListByWorkflow(ctx context.Context, workflowName string) ([]*upal.Trigger, error) {
	return r.store.Filter(ctx, func(t *upal.Trigger) bool {
		return t.WorkflowName == workflowName
	})
}

func (r *MemoryTriggerRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.Trigger, error) {
	return r.store.Filter(ctx, func(t *upal.Trigger) bool {
		return t.PipelineID == pipelineID
	})
}
