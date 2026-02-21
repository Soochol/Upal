// internal/repository/pipeline_memory.go
package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// MemoryPipelineRepository implements PipelineRepository in-memory.
type MemoryPipelineRepository struct {
	store *memstore.Store[*upal.Pipeline]
}

func NewMemoryPipelineRepository() *MemoryPipelineRepository {
	return &MemoryPipelineRepository{
		store: memstore.New(func(p *upal.Pipeline) string { return p.ID }),
	}
}

func (r *MemoryPipelineRepository) Create(ctx context.Context, p *upal.Pipeline) error {
	if r.store.Has(ctx, p.ID) {
		return fmt.Errorf("pipeline %q already exists", p.ID)
	}
	return r.store.Set(ctx, p)
}

func (r *MemoryPipelineRepository) Get(ctx context.Context, id string) (*upal.Pipeline, error) {
	p, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("pipeline %q not found", id)
	}
	return p, err
}

func (r *MemoryPipelineRepository) List(ctx context.Context) ([]*upal.Pipeline, error) {
	return r.store.All(ctx)
}

func (r *MemoryPipelineRepository) Update(ctx context.Context, p *upal.Pipeline) error {
	if !r.store.Has(ctx, p.ID) {
		return fmt.Errorf("pipeline %q not found", p.ID)
	}
	return r.store.Set(ctx, p)
}

func (r *MemoryPipelineRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("pipeline %q not found", id)
	}
	return err
}

// MemoryPipelineRunRepository implements PipelineRunRepository in-memory.
type MemoryPipelineRunRepository struct {
	store *memstore.Store[*upal.PipelineRun]
}

func NewMemoryPipelineRunRepository() *MemoryPipelineRunRepository {
	return &MemoryPipelineRunRepository{
		store: memstore.New(func(r *upal.PipelineRun) string { return r.ID }),
	}
}

func (r *MemoryPipelineRunRepository) Create(ctx context.Context, run *upal.PipelineRun) error {
	return r.store.Set(ctx, run)
}

func (r *MemoryPipelineRunRepository) Get(ctx context.Context, id string) (*upal.PipelineRun, error) {
	run, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("pipeline run %q not found", id)
	}
	return run, err
}

func (r *MemoryPipelineRunRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	return r.store.Filter(ctx, func(run *upal.PipelineRun) bool {
		return run.PipelineID == pipelineID
	})
}

func (r *MemoryPipelineRunRepository) Update(ctx context.Context, run *upal.PipelineRun) error {
	if !r.store.Has(ctx, run.ID) {
		return fmt.Errorf("pipeline run %q not found", run.ID)
	}
	return r.store.Set(ctx, run)
}
