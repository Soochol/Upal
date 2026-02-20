package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

// PersistentScheduleRepository wraps a MemoryScheduleRepository with a PostgreSQL backend.
// Writes go to both stores (DB failure is logged but non-fatal).
// Reads try memory first, falling back to the database.
type PersistentScheduleRepository struct {
	mem *MemoryScheduleRepository
	db  *db.DB
}

func NewPersistentScheduleRepository(mem *MemoryScheduleRepository, database *db.DB) *PersistentScheduleRepository {
	return &PersistentScheduleRepository{mem: mem, db: database}
}

func (r *PersistentScheduleRepository) Create(ctx context.Context, schedule *upal.Schedule) error {
	_ = r.mem.Create(ctx, schedule)
	if err := r.db.CreateSchedule(ctx, schedule); err != nil {
		slog.Warn("db create schedule failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentScheduleRepository) Get(ctx context.Context, id string) (*upal.Schedule, error) {
	s, err := r.mem.Get(ctx, id)
	if err == nil {
		return s, nil
	}

	dbSched, dbErr := r.db.GetSchedule(ctx, id)
	if dbErr != nil {
		return nil, err // return original ErrNotFound
	}

	_ = r.mem.Create(ctx, dbSched)
	return dbSched, nil
}

func (r *PersistentScheduleRepository) Update(ctx context.Context, schedule *upal.Schedule) error {
	_ = r.mem.Update(ctx, schedule)
	if err := r.db.UpdateSchedule(ctx, schedule); err != nil {
		slog.Warn("db update schedule failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentScheduleRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeleteSchedule(ctx, id); err != nil {
		slog.Warn("db delete schedule failed", "err", err)
	}
	return nil
}

func (r *PersistentScheduleRepository) List(ctx context.Context) ([]*upal.Schedule, error) {
	schedules, err := r.db.ListSchedules(ctx)
	if err == nil {
		return schedules, nil
	}
	slog.Warn("db list schedules failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentScheduleRepository) ListDue(ctx context.Context, now time.Time) ([]*upal.Schedule, error) {
	schedules, err := r.db.ListDueSchedules(ctx, now)
	if err == nil {
		return schedules, nil
	}
	slog.Warn("db list due schedules failed, falling back to in-memory", "err", err)
	return r.mem.ListDue(ctx, now)
}
