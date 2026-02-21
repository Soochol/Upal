// internal/repository/pipeline_persistent.go
package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// PipelineDB defines the DB-layer methods needed by the persistent pipeline repo.
// *db.DB satisfies this interface.
type PipelineDB interface {
	CreatePipeline(ctx context.Context, p *upal.Pipeline) error
	GetPipeline(ctx context.Context, id string) (*upal.Pipeline, error)
	ListPipelines(ctx context.Context) ([]*upal.Pipeline, error)
	UpdatePipeline(ctx context.Context, p *upal.Pipeline) error
	DeletePipeline(ctx context.Context, id string) error
}

// PipelineRunDB defines the DB-layer methods needed by the persistent run repo.
type PipelineRunDB interface {
	CreatePipelineRun(ctx context.Context, run *upal.PipelineRun) error
	GetPipelineRun(ctx context.Context, id string) (*upal.PipelineRun, error)
	ListPipelineRunsByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error)
	UpdatePipelineRun(ctx context.Context, run *upal.PipelineRun) error
}

// PersistentPipelineRepository wraps MemoryPipelineRepository with a PostgreSQL backend.
// Writes go to both. Reads try memory first; on miss, fall back to DB and cache.
type PersistentPipelineRepository struct {
	mem *MemoryPipelineRepository
	db  PipelineDB
}

func NewPersistentPipelineRepository(mem *MemoryPipelineRepository, db PipelineDB) *PersistentPipelineRepository {
	return &PersistentPipelineRepository{mem: mem, db: db}
}

func (r *PersistentPipelineRepository) Create(ctx context.Context, p *upal.Pipeline) error {
	_ = r.mem.Create(ctx, p)
	if err := r.db.CreatePipeline(ctx, p); err != nil {
		return fmt.Errorf("db create pipeline: %w", err)
	}
	return nil
}

func (r *PersistentPipelineRepository) Get(ctx context.Context, id string) (*upal.Pipeline, error) {
	if p, err := r.mem.Get(ctx, id); err == nil {
		return p, nil
	}
	p, err := r.db.GetPipeline(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, p)
	return p, nil
}

func (r *PersistentPipelineRepository) List(ctx context.Context) ([]*upal.Pipeline, error) {
	pipelines, err := r.db.ListPipelines(ctx)
	if err == nil {
		return pipelines, nil
	}
	slog.Warn("db list pipelines failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentPipelineRepository) Update(ctx context.Context, p *upal.Pipeline) error {
	_ = r.mem.Update(ctx, p)
	if err := r.db.UpdatePipeline(ctx, p); err != nil {
		return fmt.Errorf("db update pipeline: %w", err)
	}
	return nil
}

func (r *PersistentPipelineRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeletePipeline(ctx, id); err != nil {
		return fmt.Errorf("db delete pipeline: %w", err)
	}
	return nil
}

// PersistentPipelineRunRepository wraps MemoryPipelineRunRepository with a PostgreSQL backend.
type PersistentPipelineRunRepository struct {
	mem *MemoryPipelineRunRepository
	db  PipelineRunDB
}

func NewPersistentPipelineRunRepository(mem *MemoryPipelineRunRepository, db PipelineRunDB) *PersistentPipelineRunRepository {
	return &PersistentPipelineRunRepository{mem: mem, db: db}
}

func (r *PersistentPipelineRunRepository) Create(ctx context.Context, run *upal.PipelineRun) error {
	_ = r.mem.Create(ctx, run)
	if err := r.db.CreatePipelineRun(ctx, run); err != nil {
		return fmt.Errorf("db create pipeline_run: %w", err)
	}
	return nil
}

func (r *PersistentPipelineRunRepository) Get(ctx context.Context, id string) (*upal.PipelineRun, error) {
	if run, err := r.mem.Get(ctx, id); err == nil {
		return run, nil
	}
	run, err := r.db.GetPipelineRun(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, run)
	return run, nil
}

func (r *PersistentPipelineRunRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	runs, err := r.db.ListPipelineRunsByPipeline(ctx, pipelineID)
	if err == nil {
		return runs, nil
	}
	slog.Warn("db list pipeline_runs failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipeline(ctx, pipelineID)
}

func (r *PersistentPipelineRunRepository) Update(ctx context.Context, run *upal.PipelineRun) error {
	_ = r.mem.Update(ctx, run)
	if err := r.db.UpdatePipelineRun(ctx, run); err != nil {
		return fmt.Errorf("db update pipeline_run: %w", err)
	}
	return nil
}
