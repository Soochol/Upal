package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type RunRepository interface {
	Create(ctx context.Context, record *upal.RunRecord) error
	Get(ctx context.Context, id string) (*upal.RunRecord, error)
	Update(ctx context.Context, record *upal.RunRecord) error
	ListByWorkflow(ctx context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error)
	ListAll(ctx context.Context, limit, offset int, status string) ([]*upal.RunRecord, int, error)
}
