// internal/services/pipeline_runner.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// StageExecutor is the interface for executing a pipeline stage.
// Implement this interface to add new stage types.
type StageExecutor interface {
	Type() string
	Execute(ctx context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error)
}

// PipelineRunner orchestrates sequential execution of pipeline stages.
type PipelineRunner struct {
	executors map[string]StageExecutor
	runRepo   repository.PipelineRunRepository
}

func NewPipelineRunner(runRepo repository.PipelineRunRepository) *PipelineRunner {
	return &PipelineRunner{
		executors: make(map[string]StageExecutor),
		runRepo:   runRepo,
	}
}

func (r *PipelineRunner) RegisterExecutor(exec StageExecutor) {
	r.executors[exec.Type()] = exec
}

func (r *PipelineRunner) Start(ctx context.Context, pipeline *upal.Pipeline) (*upal.PipelineRun, error) {
	run := &upal.PipelineRun{
		ID:           upal.GenerateID("prun"),
		PipelineID:   pipeline.ID,
		Status:       "running",
		StageResults: make(map[string]*upal.StageResult),
		StartedAt:    time.Now(),
	}
	r.runRepo.Create(ctx, run)

	var prevResult *upal.StageResult

	for _, stage := range pipeline.Stages {
		executor, ok := r.executors[stage.Type]
		if !ok {
			run.Status = "failed"
			r.runRepo.Update(ctx, run)
			return run, fmt.Errorf("no executor registered for stage type %q", stage.Type)
		}

		run.CurrentStage = stage.ID
		stageResult := &upal.StageResult{
			StageID:   stage.ID,
			Status:    "running",
			StartedAt: time.Now(),
		}
		run.StageResults[stage.ID] = stageResult
		r.runRepo.Update(ctx, run)

		result, err := executor.Execute(ctx, stage, prevResult)
		if err != nil {
			now := time.Now()
			stageResult.Status = "failed"
			stageResult.Error = err.Error()
			stageResult.CompletedAt = &now
			run.Status = "failed"
			run.CompletedAt = &now
			r.runRepo.Update(ctx, run)
			return run, fmt.Errorf("stage %q failed: %w", stage.ID, err)
		}

		if result.Status == "waiting" {
			run.Status = "waiting"
			run.StageResults[stage.ID] = result
			r.runRepo.Update(ctx, run)
			return run, nil
		}

		now := time.Now()
		result.CompletedAt = &now
		run.StageResults[stage.ID] = result
		r.runRepo.Update(ctx, run)

		prevResult = result
	}

	now := time.Now()
	run.Status = "completed"
	run.CompletedAt = &now
	r.runRepo.Update(ctx, run)

	return run, nil
}
