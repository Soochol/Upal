package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type WorkflowRepository interface {
	Create(ctx context.Context, wf *upal.WorkflowDefinition) error
	Get(ctx context.Context, name string) (*upal.WorkflowDefinition, error)
	List(ctx context.Context) ([]*upal.WorkflowDefinition, error)
	Update(ctx context.Context, name string, wf *upal.WorkflowDefinition) error
	Delete(ctx context.Context, name string) error
}
