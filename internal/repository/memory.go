package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

type MemoryRepository struct {
	store *memstore.Store[*upal.WorkflowDefinition]
}

func NewMemory() *MemoryRepository {
	return &MemoryRepository{
		store: memstore.New(func(w *upal.WorkflowDefinition) string { return w.Name }),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, wf *upal.WorkflowDefinition) error {
	return r.store.Set(ctx, wf)
}

func (r *MemoryRepository) Get(ctx context.Context, name string) (*upal.WorkflowDefinition, error) {
	wf, err := r.store.Get(ctx, name)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return wf, err
}

func (r *MemoryRepository) List(ctx context.Context) ([]*upal.WorkflowDefinition, error) {
	return r.store.All(ctx)
}

func (r *MemoryRepository) Update(ctx context.Context, name string, wf *upal.WorkflowDefinition) error {
	if name != wf.Name {
		_ = r.store.Delete(ctx, name)
	}
	return r.store.Set(ctx, wf)
}

func (r *MemoryRepository) Delete(ctx context.Context, name string) error {
	_ = r.store.Delete(ctx, name)
	return nil
}
