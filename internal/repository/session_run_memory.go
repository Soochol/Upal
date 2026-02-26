package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// MemorySessionRunRepository is an in-memory implementation of SessionRunRepository.
type MemorySessionRunRepository struct {
	store *memstore.Store[*upal.Run]
}

// NewMemorySessionRunRepository creates a new in-memory session run repository.
func NewMemorySessionRunRepository() *MemorySessionRunRepository {
	return &MemorySessionRunRepository{
		store: memstore.New(func(r *upal.Run) string { return r.ID }),
	}
}

func (r *MemorySessionRunRepository) Create(ctx context.Context, run *upal.Run) error {
	if r.store.Has(ctx, run.ID) {
		return fmt.Errorf("run %q already exists", run.ID)
	}
	return r.store.Set(ctx, run)
}

func (r *MemorySessionRunRepository) Get(ctx context.Context, id string) (*upal.Run, error) {
	run, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("run %q: %w", id, ErrNotFound)
	}
	return run, err
}

func (r *MemorySessionRunRepository) List(ctx context.Context) ([]*upal.Run, error) {
	return r.store.All(ctx)
}

func (r *MemorySessionRunRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.Run, error) {
	return r.store.Filter(ctx, func(run *upal.Run) bool {
		return run.SessionID == sessionID
	})
}

func (r *MemorySessionRunRepository) ListByStatus(ctx context.Context, status upal.SessionRunStatus) ([]*upal.Run, error) {
	return r.store.Filter(ctx, func(run *upal.Run) bool {
		return run.Status == status
	})
}

func (r *MemorySessionRunRepository) Update(ctx context.Context, run *upal.Run) error {
	if !r.store.Has(ctx, run.ID) {
		return fmt.Errorf("run %q: %w", run.ID, ErrNotFound)
	}
	return r.store.Set(ctx, run)
}

func (r *MemorySessionRunRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("run %q: %w", id, ErrNotFound)
	}
	return err
}

func (r *MemorySessionRunRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	r.store.DeleteWhere(ctx, func(run *upal.Run) bool { return run.SessionID == sessionID })
	return nil
}

// MemoryWorkflowRunRepository is an in-memory implementation of WorkflowRunRepository.
type MemoryWorkflowRunRepository struct {
	mu      sync.RWMutex
	results map[string][]upal.WorkflowRun // runID -> results
}

// NewMemoryWorkflowRunRepository creates a new in-memory workflow run repository.
func NewMemoryWorkflowRunRepository() *MemoryWorkflowRunRepository {
	return &MemoryWorkflowRunRepository{
		results: make(map[string][]upal.WorkflowRun),
	}
}

func (r *MemoryWorkflowRunRepository) Save(_ context.Context, runID string, results []upal.WorkflowRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]upal.WorkflowRun, len(results))
	copy(cp, results)
	r.results[runID] = cp
	return nil
}

func (r *MemoryWorkflowRunRepository) GetByRun(_ context.Context, runID string) ([]upal.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	orig, ok := r.results[runID]
	if !ok {
		return []upal.WorkflowRun{}, nil
	}
	cp := make([]upal.WorkflowRun, len(orig))
	copy(cp, orig)
	return cp, nil
}

func (r *MemoryWorkflowRunRepository) DeleteByRun(_ context.Context, runID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.results, runID)
	return nil
}
