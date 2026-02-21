package repository

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

// PersistentRunRepository wraps a MemoryRunRepository with a PostgreSQL backend.
// Writes go to both stores (DB failure is logged but non-fatal).
// Reads try memory first, falling back to the database.
type PersistentRunRepository struct {
	mem *MemoryRunRepository
	db  *db.DB
}

func NewPersistentRunRepository(mem *MemoryRunRepository, database *db.DB) *PersistentRunRepository {
	return &PersistentRunRepository{mem: mem, db: database}
}

func (r *PersistentRunRepository) Create(ctx context.Context, record *upal.RunRecord) error {
	_ = r.mem.Create(ctx, record)
	if err := r.db.CreateRun(ctx, record); err != nil {
		slog.Warn("db create run failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentRunRepository) Get(ctx context.Context, id string) (*upal.RunRecord, error) {
	rec, err := r.mem.Get(ctx, id)
	if err == nil {
		return rec, nil
	}

	dbRec, dbErr := r.db.GetRun(ctx, id)
	if dbErr != nil {
		return nil, err // return original ErrNotFound
	}

	_ = r.mem.Create(ctx, dbRec)
	return dbRec, nil
}

func (r *PersistentRunRepository) Update(ctx context.Context, record *upal.RunRecord) error {
	_ = r.mem.Update(ctx, record)
	if err := r.db.UpdateRun(ctx, record); err != nil {
		slog.Warn("db update run failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentRunRepository) ListByWorkflow(ctx context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error) {
	runs, total, err := r.db.ListRunsByWorkflow(ctx, workflowName, limit, offset)
	if err == nil {
		return runs, total, nil
	}
	slog.Warn("db list runs failed, falling back to in-memory", "err", err)
	return r.mem.ListByWorkflow(ctx, workflowName, limit, offset)
}

func (r *PersistentRunRepository) MarkOrphanedRunsFailed(ctx context.Context) (int64, error) {
	return r.db.MarkOrphanedRunsFailed(ctx)
}

func (r *PersistentRunRepository) ListAll(ctx context.Context, limit, offset int, status string) ([]*upal.RunRecord, int, error) {
	runs, total, err := r.db.ListAllRuns(ctx, limit, offset, status)
	if err == nil {
		return runs, total, nil
	}
	slog.Warn("db list all runs failed, falling back to in-memory", "err", err)
	return r.mem.ListAll(ctx, limit, offset, status)
}
