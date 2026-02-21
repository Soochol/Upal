package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// RunRepository abstracts persistence for workflow execution records.
type RunRepository interface {
	Create(ctx context.Context, record *upal.RunRecord) error
	Get(ctx context.Context, id string) (*upal.RunRecord, error)
	Update(ctx context.Context, record *upal.RunRecord) error
	ListByWorkflow(ctx context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error)
	// ListAll returns all runs. status filters by run status when non-empty ("" = all).
	ListAll(ctx context.Context, limit, offset int, status string) ([]*upal.RunRecord, int, error)
}
