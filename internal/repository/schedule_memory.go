package repository

import (
	"context"
	"errors"
	"time"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// MemoryScheduleRepository stores schedules in memory.
type MemoryScheduleRepository struct {
	store *memstore.Store[*upal.Schedule]
}

func NewMemoryScheduleRepository() *MemoryScheduleRepository {
	return &MemoryScheduleRepository{
		store: memstore.New(func(s *upal.Schedule) string { return s.ID }),
	}
}

func (r *MemoryScheduleRepository) Create(ctx context.Context, schedule *upal.Schedule) error {
	return r.store.Set(ctx, schedule)
}

func (r *MemoryScheduleRepository) Get(ctx context.Context, id string) (*upal.Schedule, error) {
	s, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *MemoryScheduleRepository) Update(ctx context.Context, schedule *upal.Schedule) error {
	if !r.store.Has(ctx, schedule.ID) {
		return ErrNotFound
	}
	return r.store.Set(ctx, schedule)
}

func (r *MemoryScheduleRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (r *MemoryScheduleRepository) List(ctx context.Context) ([]*upal.Schedule, error) {
	return r.store.All(ctx)
}

func (r *MemoryScheduleRepository) ListDue(ctx context.Context, now time.Time) ([]*upal.Schedule, error) {
	return r.store.Filter(ctx, func(s *upal.Schedule) bool {
		return s.Enabled && !s.NextRunAt.After(now)
	})
}

func (r *MemoryScheduleRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.Schedule, error) {
	return r.store.Filter(ctx, func(s *upal.Schedule) bool {
		return s.PipelineID == pipelineID
	})
}
