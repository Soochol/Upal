package repository

import (
	"context"
	"sync"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// MemoryScheduleRepository stores schedules in memory.
type MemoryScheduleRepository struct {
	mu        sync.RWMutex
	schedules map[string]*upal.Schedule
}

func NewMemoryScheduleRepository() *MemoryScheduleRepository {
	return &MemoryScheduleRepository{
		schedules: make(map[string]*upal.Schedule),
	}
}

func (r *MemoryScheduleRepository) Create(_ context.Context, schedule *upal.Schedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules[schedule.ID] = schedule
	return nil
}

func (r *MemoryScheduleRepository) Get(_ context.Context, id string) (*upal.Schedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.schedules[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *MemoryScheduleRepository) Update(_ context.Context, schedule *upal.Schedule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.schedules[schedule.ID]; !ok {
		return ErrNotFound
	}
	r.schedules[schedule.ID] = schedule
	return nil
}

func (r *MemoryScheduleRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.schedules[id]; !ok {
		return ErrNotFound
	}
	delete(r.schedules, id)
	return nil
}

func (r *MemoryScheduleRepository) List(_ context.Context) ([]*upal.Schedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*upal.Schedule, 0, len(r.schedules))
	for _, s := range r.schedules {
		result = append(result, s)
	}
	return result, nil
}

func (r *MemoryScheduleRepository) ListDue(_ context.Context, now time.Time) ([]*upal.Schedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var due []*upal.Schedule
	for _, s := range r.schedules {
		if s.Enabled && !s.NextRunAt.After(now) {
			due = append(due, s)
		}
	}
	return due, nil
}

func (r *MemoryScheduleRepository) ListByPipeline(_ context.Context, pipelineID string) ([]*upal.Schedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*upal.Schedule
	for _, s := range r.schedules {
		if s.PipelineID == pipelineID {
			result = append(result, s)
		}
	}
	return result, nil
}
