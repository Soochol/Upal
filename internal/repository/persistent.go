package repository

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

// PersistentRepository wraps a MemoryRepository with a PostgreSQL backend.
// Writes go to both stores (DB failure is logged but non-fatal).
// Reads try memory first, falling back to the database.
type PersistentRepository struct {
	mem *MemoryRepository
	db  *db.DB
}

// NewPersistent creates a repository backed by both memory and PostgreSQL.
func NewPersistent(mem *MemoryRepository, database *db.DB) *PersistentRepository {
	return &PersistentRepository{mem: mem, db: database}
}

func (r *PersistentRepository) Create(ctx context.Context, wf *upal.WorkflowDefinition) error {
	_ = r.mem.Create(ctx, wf)
	if _, err := r.db.CreateWorkflow(ctx, wf); err != nil {
		slog.Warn("db create workflow failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentRepository) Get(ctx context.Context, name string) (*upal.WorkflowDefinition, error) {
	// Fast path: in-memory.
	wf, err := r.mem.Get(ctx, name)
	if err == nil {
		return wf, nil
	}

	// Fallback: database.
	row, dbErr := r.db.GetWorkflow(ctx, name)
	if dbErr != nil {
		return nil, err // return original ErrNotFound
	}

	// Cache in memory for future lookups.
	_ = r.mem.Create(ctx, &row.Definition)
	return &row.Definition, nil
}

func (r *PersistentRepository) List(ctx context.Context) ([]*upal.WorkflowDefinition, error) {
	// Prefer DB for durable listing.
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
	if err := r.db.UpdateWorkflow(ctx, name, wf); err != nil {
		slog.Warn("db update workflow failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentRepository) Delete(ctx context.Context, name string) error {
	_ = r.mem.Delete(ctx, name)
	if err := r.db.DeleteWorkflow(ctx, name); err != nil {
		slog.Warn("db delete workflow failed", "err", err)
	}
	return nil
}
