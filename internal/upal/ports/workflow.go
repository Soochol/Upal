package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// WorkflowExecutor is the port that defines the contract for workflow
// lookup, validation, and execution. Services that need to run workflows
// should depend on this interface rather than on *WorkflowService directly.
type WorkflowExecutor interface {
	Lookup(ctx context.Context, name string) (*upal.WorkflowDefinition, error)
	Validate(wf *upal.WorkflowDefinition) error
	Run(ctx context.Context, wf *upal.WorkflowDefinition, inputs map[string]any) (<-chan upal.WorkflowEvent, <-chan upal.RunResult, error)
}
