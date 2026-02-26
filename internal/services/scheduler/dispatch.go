package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func (s *SchedulerService) executeScheduledRun(schedule *upal.Schedule) {
	ctx := context.Background()

	if schedule.PipelineID != "" && s.pipelineSvc != nil && s.pipelineRunner != nil {
		s.executePipelineRun(ctx, schedule)
		return
	}

	s.executeWorkflowRun(ctx, schedule)
}

func (s *SchedulerService) executePipelineRun(ctx context.Context, schedule *upal.Schedule) {
	slog.Info("scheduler: executing scheduled pipeline run",
		"schedule", schedule.ID, "pipeline", schedule.PipelineID)

	pipeline, err := s.pipelineSvc.Get(ctx, schedule.PipelineID)
	if err != nil {
		slog.Error("scheduler: pipeline not found",
			"schedule", schedule.ID, "pipeline", schedule.PipelineID, "err", err)
		return
	}

	if s.contentCollector != nil {
		if err := s.contentCollector.CollectPipeline(ctx, schedule.PipelineID); err != nil {
			slog.Error("scheduler: content pipeline collection failed",
				"schedule", schedule.ID, "pipeline", schedule.PipelineID, "err", err)
		}
	} else if _, err := s.pipelineRunner.Start(ctx, pipeline, nil); err != nil {
		slog.Error("scheduler: pipeline execution failed",
			"schedule", schedule.ID, "pipeline", schedule.PipelineID, "err", err)
	}

	s.updateScheduleTimestamps(ctx, schedule)
}

func (s *SchedulerService) executeWorkflowRun(ctx context.Context, schedule *upal.Schedule) {
	slog.Info("scheduler: executing scheduled run",
		"schedule", schedule.ID, "workflow", schedule.WorkflowName)

	if err := s.limiter.Acquire(ctx, schedule.WorkflowName); err != nil {
		slog.Warn("scheduler: concurrency limit reached, skipping",
			"schedule", schedule.ID, "err", err)
		return
	}
	defer s.limiter.Release(schedule.WorkflowName)

	wf, err := s.workflowExec.Lookup(ctx, schedule.WorkflowName)
	if err != nil {
		slog.Error("scheduler: workflow not found",
			"schedule", schedule.ID, "workflow", schedule.WorkflowName, "err", err)
		return
	}

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

	for range events {
	}

	res, ok := <-result
	if ok {
		slog.Info("scheduler: run completed",
			"schedule", schedule.ID, "session", res.SessionID)
	} else {
		slog.Warn("scheduler: run result channel closed without value",
			"schedule", schedule.ID)
	}

	s.updateScheduleTimestamps(ctx, schedule)
}

func (s *SchedulerService) updateScheduleTimestamps(ctx context.Context, schedule *upal.Schedule) {
	now := time.Now()
	schedule.LastRunAt = &now
	schedule.UpdatedAt = now

	if cronSched, err := parseCronExpr(schedule.CronExpr, schedule.Timezone); err == nil {
		schedule.NextRunAt = cronSched.Next(now)
	}

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		slog.Warn("scheduler: failed to update schedule timestamps", "err", err)
	}
}
