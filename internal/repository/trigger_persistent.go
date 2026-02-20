package repository

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

// PersistentTriggerRepository wraps a MemoryTriggerRepository with a PostgreSQL backend.
// Writes go to both stores (DB failure is logged but non-fatal).
// Reads try memory first, falling back to the database.
type PersistentTriggerRepository struct {
	mem *MemoryTriggerRepository
	db  *db.DB
}

func NewPersistentTriggerRepository(mem *MemoryTriggerRepository, database *db.DB) *PersistentTriggerRepository {
	return &PersistentTriggerRepository{mem: mem, db: database}
}

func (r *PersistentTriggerRepository) Create(ctx context.Context, trigger *upal.Trigger) error {
	_ = r.mem.Create(ctx, trigger)
	if err := r.db.CreateTrigger(ctx, trigger); err != nil {
		slog.Warn("db create trigger failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentTriggerRepository) Get(ctx context.Context, id string) (*upal.Trigger, error) {
	t, err := r.mem.Get(ctx, id)
	if err == nil {
		return t, nil
	}

	dbTrig, dbErr := r.db.GetTrigger(ctx, id)
	if dbErr != nil {
		return nil, err // return original ErrNotFound
	}

	_ = r.mem.Create(ctx, dbTrig)
	return dbTrig, nil
}

func (r *PersistentTriggerRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeleteTrigger(ctx, id); err != nil {
		slog.Warn("db delete trigger failed", "err", err)
	}
	return nil
}

func (r *PersistentTriggerRepository) ListByWorkflow(ctx context.Context, workflowName string) ([]*upal.Trigger, error) {
	triggers, err := r.db.ListTriggersByWorkflow(ctx, workflowName)
	if err == nil {
		return triggers, nil
	}
	slog.Warn("db list triggers failed, falling back to in-memory", "err", err)
	return r.mem.ListByWorkflow(ctx, workflowName)
}
