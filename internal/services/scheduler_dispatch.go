package services

// scheduler_dispatch.go — ScheduleDispatcher responsibility layer.
// executeScheduledRun contains the workflow/pipeline branching logic that fires
// when a cron entry triggers, implemented as a method on SchedulerService.

import (
	"context"
	"log/slog"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// executeScheduledRun is called by cron when a schedule fires.
// It dispatches to pipeline execution or workflow execution depending on
// whether the schedule carries a PipelineID.
func (s *SchedulerService) executeScheduledRun(schedule *upal.Schedule) {
	ctx := context.Background()

	// Pipeline-triggered execution.
	if schedule.PipelineID != "" && s.pipelineSvc != nil && s.pipelineRunner != nil {
		slog.Info("scheduler: executing scheduled pipeline run",
			"schedule", schedule.ID, "pipeline", schedule.PipelineID)

		pipeline, err := s.pipelineSvc.Get(ctx, schedule.PipelineID)
		if err != nil {
			slog.Error("scheduler: pipeline not found",
				"schedule", schedule.ID, "pipeline", schedule.PipelineID, "err", err)
			return
		}

		if _, err := s.pipelineRunner.Start(ctx, pipeline); err != nil {
			slog.Error("scheduler: pipeline execution failed",
				"schedule", schedule.ID, "pipeline", schedule.PipelineID, "err", err)
		}

		now := time.Now()
		schedule.LastRunAt = &now
		if cronSched, parseErr := parseCronExpr(schedule.CronExpr, schedule.Timezone); parseErr == nil {
			schedule.NextRunAt = cronSched.Next(now)
		}
		schedule.UpdatedAt = now
		if updateErr := s.scheduleRepo.Update(ctx, schedule); updateErr != nil {
			slog.Warn("scheduler: failed to update schedule after pipeline run", "err", updateErr)
		}
		return
	}

	slog.Info("scheduler: executing scheduled run",
		"schedule", schedule.ID, "workflow", schedule.WorkflowName)

	// Acquire concurrency slot.
	if err := s.limiter.Acquire(ctx, schedule.WorkflowName); err != nil {
		slog.Warn("scheduler: concurrency limit reached, skipping",
			"schedule", schedule.ID, "err", err)
		return
	}
	defer s.limiter.Release(schedule.WorkflowName)

	// Lookup workflow.
	wf, err := s.workflowExec.Lookup(ctx, schedule.WorkflowName)
	if err != nil {
		slog.Error("scheduler: workflow not found",
			"schedule", schedule.ID, "workflow", schedule.WorkflowName, "err", err)
		return
	}

	// Execute with retry if policy is set.
	policy := upal.DefaultRetryPolicy()
	if schedule.RetryPolicy != nil {
		policy = *schedule.RetryPolicy
	}

	events, result, err := s.retryExecutor.ExecuteWithRetry(
		ctx, wf, schedule.Inputs, policy,
		string(upal.TriggerCron), schedule.ID,
	)
	if err != nil {
		slog.Error("scheduler: execution failed",
			"schedule", schedule.ID, "err", err)
		return
	}

	// Drain events (no SSE consumer for scheduled runs).
	for range events {
	}

	// Wait for result.
	res, ok := <-result
	if ok {
		slog.Info("scheduler: run completed",
			"schedule", schedule.ID, "session", res.SessionID)
	}

	// Update schedule timestamps.
	now := time.Now()
	schedule.LastRunAt = &now

	// Compute next run time.
	cronSched, parseErr := parseCronExpr(schedule.CronExpr, schedule.Timezone)
	if parseErr == nil {
		schedule.NextRunAt = cronSched.Next(now)
	}
	schedule.UpdatedAt = now

	if updateErr := s.scheduleRepo.Update(ctx, schedule); updateErr != nil {
		slog.Warn("scheduler: failed to update schedule after run", "err", updateErr)
	}
}
