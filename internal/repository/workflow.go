// Package repository defines storage interfaces for domain entities.
package repository

import (
	"context"
	"errors"

	"github.com/soochol/upal/internal/upal"
)

// ErrNotFound is returned when a requested workflow does not exist.
var ErrNotFound = errors.New("workflow not found")

// WorkflowRepository abstracts workflow persistence so callers don't
// need to know whether storage is in-memory, PostgreSQL, or a mix.
type WorkflowRepository interface {
	Create(ctx context.Context, wf *upal.WorkflowDefinition) error
	Get(ctx context.Context, name string) (*upal.WorkflowDefinition, error)
	List(ctx context.Context) ([]*upal.WorkflowDefinition, error)
	Update(ctx context.Context, name string, wf *upal.WorkflowDefinition) error
	Delete(ctx context.Context, name string) error
}
