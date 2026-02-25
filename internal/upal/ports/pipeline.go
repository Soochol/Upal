package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// PipelineServicePort defines the pipeline management boundary.
// The API layer should depend on this interface rather than *services.PipelineService directly.
type PipelineServicePort interface {
	Create(ctx context.Context, p *upal.Pipeline) error
	Get(ctx context.Context, id string) (*upal.Pipeline, error)
	List(ctx context.Context) ([]*upal.Pipeline, error)
	Update(ctx context.Context, p *upal.Pipeline) error
	Delete(ctx context.Context, id string) error
	GetRun(ctx context.Context, id string) (*upal.PipelineRun, error)
	ListRuns(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error)
	CreateRun(ctx context.Context, run *upal.PipelineRun) error
	UpdateRun(ctx context.Context, run *upal.PipelineRun) error
	RejectRun(ctx context.Context, pipelineID, runID string) (*upal.PipelineRun, error)
}
