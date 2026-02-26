package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// SessionRunRepository defines the data access interface for Run entities.
type SessionRunRepository interface {
	Create(ctx context.Context, r *upal.Run) error
	Get(ctx context.Context, id string) (*upal.Run, error)
	List(ctx context.Context) ([]*upal.Run, error)
	ListBySession(ctx context.Context, sessionID string) ([]*upal.Run, error)
	ListByStatus(ctx context.Context, status upal.SessionRunStatus) ([]*upal.Run, error)
	Update(ctx context.Context, r *upal.Run) error
	Delete(ctx context.Context, id string) error
	DeleteBySession(ctx context.Context, sessionID string) error
}

// WorkflowRunRepository defines the data access interface for workflow execution
// results within a Run's produce phase.
type WorkflowRunRepository interface {
	Save(ctx context.Context, runID string, results []upal.WorkflowRun) error
	GetByRun(ctx context.Context, runID string) ([]upal.WorkflowRun, error)
	DeleteByRun(ctx context.Context, runID string) error
}
