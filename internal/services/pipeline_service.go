// internal/services/pipeline_service.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

type PipelineService struct {
	repo    repository.PipelineRepository
	runRepo repository.PipelineRunRepository
}

func NewPipelineService(repo repository.PipelineRepository, runRepo repository.PipelineRunRepository) *PipelineService {
	return &PipelineService{repo: repo, runRepo: runRepo}
}

func (s *PipelineService) Create(ctx context.Context, p *upal.Pipeline) error {
	if p.Name == "" {
		return fmt.Errorf("pipeline name is required")
	}
	if p.ID == "" {
		p.ID = upal.GenerateID("pipe")
	}
	for i := range p.Stages {
		if p.Stages[i].ID == "" {
			p.Stages[i].ID = fmt.Sprintf("stage-%d", i+1)
		}
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return s.repo.Create(ctx, p)
}

func (s *PipelineService) Get(ctx context.Context, id string) (*upal.Pipeline, error) {
	return s.repo.Get(ctx, id)
}

func (s *PipelineService) List(ctx context.Context) ([]*upal.Pipeline, error) {
	return s.repo.List(ctx)
}

func (s *PipelineService) Update(ctx context.Context, p *upal.Pipeline) error {
	p.UpdatedAt = time.Now()
	return s.repo.Update(ctx, p)
}

func (s *PipelineService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *PipelineService) GetRun(ctx context.Context, runID string) (*upal.PipelineRun, error) {
	return s.runRepo.Get(ctx, runID)
}

func (s *PipelineService) ListRuns(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	return s.runRepo.ListByPipeline(ctx, pipelineID)
}

func (s *PipelineService) CreateRun(ctx context.Context, run *upal.PipelineRun) error {
	if run.ID == "" {
		run.ID = upal.GenerateID("prun")
	}
	run.StartedAt = time.Now()
	return s.runRepo.Create(ctx, run)
}

func (s *PipelineService) UpdateRun(ctx context.Context, run *upal.PipelineRun) error {
	return s.runRepo.Update(ctx, run)
}
