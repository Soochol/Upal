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
	if err := r.runRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}
	if err := r.executeFrom(ctx, pipeline, run, 0); err != nil {
		return run, err
	}
	return run, nil
}

// Resume continues a paused pipeline run from the stage after run.CurrentStage.
func (r *PipelineRunner) Resume(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun) error {
	if run.Status != "waiting" {
		return fmt.Errorf("cannot resume run %q with status %q", run.ID, run.Status)
	}
	currentIdx := -1
	for i, stage := range pipeline.Stages {
		if stage.ID == run.CurrentStage {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		return fmt.Errorf("current stage %q not found in pipeline %q", run.CurrentStage, pipeline.ID)
	}
	return r.executeFrom(ctx, pipeline, run, currentIdx+1)
}

// executeFrom runs pipeline stages sequentially starting from startIdx.
// It updates run in the repository at each transition.
func (r *PipelineRunner) executeFrom(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun, startIdx int) error {
	// Seed prevResult from the last completed stage before startIdx.
	// Waiting results (e.g., from an approval gate) are excluded because
	// they represent a paused state, not a usable data output.
	var prevResult *upal.StageResult
	for i := 0; i < startIdx; i++ {
		stage := pipeline.Stages[i]
		if result, ok := run.StageResults[stage.ID]; ok && result.Status == "completed" {
			prevResult = result
		}
	}

	for i := startIdx; i < len(pipeline.Stages); i++ {
		stage := pipeline.Stages[i]

		executor, ok := r.executors[stage.Type]
		if !ok {
			now := time.Now()
			run.Status = "failed"
			run.CompletedAt = &now
			r.runRepo.Update(ctx, run)
			return fmt.Errorf("no executor registered for stage type %q", stage.Type)
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
			return fmt.Errorf("stage %q failed: %w", stage.ID, err)
		}

		if result.Status == "waiting" {
			run.Status = "waiting"
			run.StageResults[stage.ID] = result
			r.runRepo.Update(ctx, run)
			return nil
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
	return nil
}
