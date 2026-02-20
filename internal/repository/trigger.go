package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// TriggerRepository abstracts persistence for event-based triggers.
type TriggerRepository interface {
	Create(ctx context.Context, trigger *upal.Trigger) error
	Get(ctx context.Context, id string) (*upal.Trigger, error)
	Delete(ctx context.Context, id string) error
	ListByWorkflow(ctx context.Context, workflowName string) ([]*upal.Trigger, error)
}
