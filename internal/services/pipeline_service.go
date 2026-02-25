// internal/services/pipeline_service.go
package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

var _ ports.PipelineRegistry = (*PipelineService)(nil)
var _ ports.PipelineServicePort = (*PipelineService)(nil)

type PipelineService struct {
	repo         repository.PipelineRepository
	runRepo      repository.PipelineRunRepository
	schedulerSvc ports.SchedulerPort
}

// SetSchedulerService injects the scheduler (setter to break circular init dependency).
func (s *PipelineService) SetSchedulerService(svc ports.SchedulerPort) {
	s.schedulerSvc = svc
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
	if err := s.repo.Create(ctx, p); err != nil {
		return err
	}
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(ctx, p); err != nil {
			slog.Warn("pipeline schedule sync failed", "pipeline", p.ID, "err", err)
		} else {
			_ = s.repo.Update(ctx, p)
		}
	}
	return nil
}

func (s *PipelineService) Get(ctx context.Context, id string) (*upal.Pipeline, error) {
	return s.repo.Get(ctx, id)
}

func (s *PipelineService) List(ctx context.Context) ([]*upal.Pipeline, error) {
	return s.repo.List(ctx)
}

func (s *PipelineService) Update(ctx context.Context, p *upal.Pipeline) error {
	p.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, p); err != nil {
		return err
	}
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(ctx, p); err != nil {
			slog.Warn("pipeline schedule sync failed", "pipeline", p.ID, "err", err)
		} else {
			_ = s.repo.Update(ctx, p)
		}
	}
	return nil
}

func (s *PipelineService) Delete(ctx context.Context, id string) error {
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.RemovePipelineSchedules(ctx, id); err != nil {
			slog.Warn("pipeline schedule cleanup failed", "pipeline", id, "err", err)
		}
	}
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

// RejectRun rejects a pipeline run that is waiting for approval.
// It validates the run belongs to the given pipeline and is in waiting status,
// then transitions it to failed.
func (s *PipelineService) RejectRun(ctx context.Context, pipelineID, runID string) (*upal.PipelineRun, error) {
	run, err := s.runRepo.Get(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get run %s: %w", runID, err)
	}
	if run.PipelineID != pipelineID {
		return nil, fmt.Errorf("run %s: %w", runID, repository.ErrNotFound)
	}
	if run.Status != upal.PipelineRunWaiting {
		return nil, fmt.Errorf("run %s status %s: %w", runID, run.Status, upal.ErrInvalidStatus)
	}

	now := time.Now()
	run.Status = upal.PipelineRunFailed
	run.CompletedAt = &now
	if result, ok := run.StageResults[run.CurrentStage]; ok {
		result.Status = upal.StageStatusFailed
		result.Error = "rejected by user"
		result.CompletedAt = &now
	}
	if err := s.runRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("update run %s: %w", runID, err)
	}
	return run, nil
}
