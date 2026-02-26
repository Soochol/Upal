package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// SessionRunDB defines the database methods used by persistent session run repositories.
type SessionRunDB interface {
	CreateSessionRun(ctx context.Context, userID string, r *upal.Run) error
	GetSessionRun(ctx context.Context, userID string, id string) (*upal.Run, error)
	ListSessionRuns(ctx context.Context, userID string) ([]*upal.Run, error)
	ListSessionRunsBySession(ctx context.Context, userID string, sessionID string) ([]*upal.Run, error)
	ListSessionRunsByStatus(ctx context.Context, userID string, status string) ([]*upal.Run, error)
	UpdateSessionRun(ctx context.Context, userID string, r *upal.Run) error
	DeleteSessionRun(ctx context.Context, userID string, id string) error
	DeleteSessionRunsBySession(ctx context.Context, userID string, sessionID string) error
}

// WorkflowRunDB defines the database methods used by persistent workflow run repositories.
type WorkflowRunDB interface {
	SaveWorkflowRuns(ctx context.Context, userID string, runID string, results []upal.WorkflowRun) error
	GetWorkflowRunsByRun(ctx context.Context, userID string, runID string) ([]upal.WorkflowRun, error)
	DeleteWorkflowRunsByRun(ctx context.Context, userID string, runID string) error
}

// --- PersistentSessionRunRepository ---

type PersistentSessionRunRepository struct {
	mem *MemorySessionRunRepository
	db  SessionRunDB
}

func NewPersistentSessionRunRepository(mem *MemorySessionRunRepository, db SessionRunDB) *PersistentSessionRunRepository {
	return &PersistentSessionRunRepository{mem: mem, db: db}
}

func (r *PersistentSessionRunRepository) Create(ctx context.Context, run *upal.Run) error {
	_ = r.mem.Create(ctx, run)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateSessionRun(ctx, userID, run); err != nil {
		return fmt.Errorf("db create run: %w", err)
	}
	return nil
}

func (r *PersistentSessionRunRepository) Get(ctx context.Context, id string) (*upal.Run, error) {
	if run, err := r.mem.Get(ctx, id); err == nil {
		return run, nil
	}
	userID := upal.UserIDFromContext(ctx)
	run, err := r.db.GetSessionRun(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, run)
	return run, nil
}

func (r *PersistentSessionRunRepository) List(ctx context.Context) ([]*upal.Run, error) {
	userID := upal.UserIDFromContext(ctx)
	runs, err := r.db.ListSessionRuns(ctx, userID)
	if err == nil {
		return runs, nil
	}
	slog.Warn("db list runs failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentSessionRunRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.Run, error) {
	userID := upal.UserIDFromContext(ctx)
	runs, err := r.db.ListSessionRunsBySession(ctx, userID, sessionID)
	if err == nil {
		return runs, nil
	}
	slog.Warn("db list runs by session failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

func (r *PersistentSessionRunRepository) ListByStatus(ctx context.Context, status upal.SessionRunStatus) ([]*upal.Run, error) {
	userID := upal.UserIDFromContext(ctx)
	runs, err := r.db.ListSessionRunsByStatus(ctx, userID, string(status))
	if err == nil {
		return runs, nil
	}
	slog.Warn("db list runs by status failed, falling back to in-memory", "err", err)
	return r.mem.ListByStatus(ctx, status)
}

func (r *PersistentSessionRunRepository) Update(ctx context.Context, run *upal.Run) error {
	_ = r.mem.Update(ctx, run)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateSessionRun(ctx, userID, run); err != nil {
		return fmt.Errorf("db update run: %w", err)
	}
	return nil
}

func (r *PersistentSessionRunRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteSessionRun(ctx, userID, id); err != nil {
		return fmt.Errorf("db delete run: %w", err)
	}
	return nil
}

func (r *PersistentSessionRunRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteSessionRunsBySession(ctx, userID, sessionID); err != nil {
		return fmt.Errorf("db delete runs by session: %w", err)
	}
	return nil
}

// --- PersistentWorkflowRunRepository ---

type PersistentWorkflowRunRepository struct {
	mem *MemoryWorkflowRunRepository
	db  WorkflowRunDB
}

func NewPersistentWorkflowRunRepository(mem *MemoryWorkflowRunRepository, db WorkflowRunDB) *PersistentWorkflowRunRepository {
	return &PersistentWorkflowRunRepository{mem: mem, db: db}
}

func (r *PersistentWorkflowRunRepository) Save(ctx context.Context, runID string, results []upal.WorkflowRun) error {
	_ = r.mem.Save(ctx, runID, results)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.SaveWorkflowRuns(ctx, userID, runID, results); err != nil {
		return fmt.Errorf("db save workflow_runs: %w", err)
	}
	return nil
}

func (r *PersistentWorkflowRunRepository) GetByRun(ctx context.Context, runID string) ([]upal.WorkflowRun, error) {
	if results, err := r.mem.GetByRun(ctx, runID); err == nil && len(results) > 0 {
		return results, nil
	}
	userID := upal.UserIDFromContext(ctx)
	results, err := r.db.GetWorkflowRunsByRun(ctx, userID, runID)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		_ = r.mem.Save(ctx, runID, results)
	}
	return results, nil
}

func (r *PersistentWorkflowRunRepository) DeleteByRun(ctx context.Context, runID string) error {
	_ = r.mem.DeleteByRun(ctx, runID)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteWorkflowRunsByRun(ctx, userID, runID); err != nil {
		return fmt.Errorf("db delete workflow_runs: %w", err)
	}
	return nil
}
