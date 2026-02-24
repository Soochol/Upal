package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// WorkflowGenerator generates workflow definitions from natural language descriptions.
type WorkflowGenerator interface {
	GenerateWorkflow(ctx context.Context, description string) (*upal.WorkflowDefinition, error)
}
