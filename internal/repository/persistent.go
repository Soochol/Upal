package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

type PersistentRepository struct {
	mem *MemoryRepository
	db  *db.DB
}

func NewPersistent(mem *MemoryRepository, database *db.DB) *PersistentRepository {
	return &PersistentRepository{mem: mem, db: database}
}

func (r *PersistentRepository) Create(ctx context.Context, wf *upal.WorkflowDefinition) error {
	_ = r.mem.Create(ctx, wf)
	if _, err := r.db.CreateWorkflow(ctx, wf); err != nil {
		return fmt.Errorf("db persist failed: %w", err)
	}
	return nil
}

func (r *PersistentRepository) Get(ctx context.Context, name string) (*upal.WorkflowDefinition, error) {
	wf, err := r.mem.Get(ctx, name)
	if err == nil {
		return wf, nil
	}

	row, dbErr := r.db.GetWorkflow(ctx, name)
	if dbErr != nil {
		return nil, err
	}

	_ = r.mem.Create(ctx, &row.Definition)
	return &row.Definition, nil
}

func (r *PersistentRepository) List(ctx context.Context) ([]*upal.WorkflowDefinition, error) {
	rows, err := r.db.ListWorkflows(ctx)
	if err == nil {
		result := make([]*upal.WorkflowDefinition, len(rows))
		for i := range rows {
			result[i] = &rows[i].Definition
		}
		return result, nil
	}
	slog.Warn("db list workflows failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentRepository) Update(ctx context.Context, name string, wf *upal.WorkflowDefinition) error {
	_ = r.mem.Update(ctx, name, wf)
	// CreateWorkflow uses upsert (INSERT ON CONFLICT DO UPDATE).
	if _, err := r.db.CreateWorkflow(ctx, wf); err != nil {
		return fmt.Errorf("db persist failed: %w", err)
	}
	return nil
}

func (r *PersistentRepository) Delete(ctx context.Context, name string) error {
	_ = r.mem.Delete(ctx, name)
	if err := r.db.DeleteWorkflow(ctx, name); err != nil {
		return fmt.Errorf("db delete failed: %w", err)
	}
	return nil
}
