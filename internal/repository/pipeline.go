// internal/repository/pipeline.go
package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type PipelineRepository interface {
	Create(ctx context.Context, p *upal.Pipeline) error
	Get(ctx context.Context, id string) (*upal.Pipeline, error)
	List(ctx context.Context) ([]*upal.Pipeline, error)
	Update(ctx context.Context, p *upal.Pipeline) error
	Delete(ctx context.Context, id string) error
}

type PipelineRunRepository interface {
	Create(ctx context.Context, run *upal.PipelineRun) error
	Get(ctx context.Context, id string) (*upal.PipelineRun, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error)
	Update(ctx context.Context, run *upal.PipelineRun) error
}
