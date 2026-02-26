package scheduler

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// SyncPipelineSchedules synchronizes cron jobs with a pipeline's schedule stages.
// Pipeline.Stages is modified in-place; caller must re-save the pipeline.
func (s *SchedulerService) SyncPipelineSchedules(ctx context.Context, pipeline *upal.Pipeline) error {
	inUse := map[string]bool{}
	for _, stage := range pipeline.Stages {
		if stage.Config.ScheduleID != "" {
			inUse[stage.Config.ScheduleID] = true
		}
	}

	existing, err := s.scheduleRepo.ListByPipeline(ctx, pipeline.ID)
	if err != nil {
		slog.Warn("scheduler: failed to list existing pipeline schedules", "pipeline", pipeline.ID, "err", err)
	}
	for _, sched := range existing {
		if !inUse[sched.ID] {
			if err := s.RemoveSchedule(ctx, sched.ID); err != nil {
				slog.Warn("scheduler: failed to remove orphaned schedule", "id", sched.ID, "err", err)
			}
		}
	}

	for i := range pipeline.Stages {
		stage := &pipeline.Stages[i]
		if stage.Type != "schedule" || stage.Config.Cron == "" || stage.Config.ScheduleID != "" {
			continue
		}
		tz := stage.Config.Timezone
		if tz == "" {
			tz = "UTC"
		}
		sched := &upal.Schedule{
			PipelineID: pipeline.ID,
			CronExpr:   stage.Config.Cron,
			Enabled:    true,
			Timezone:   tz,
		}
		if err := s.AddSchedule(ctx, sched); err != nil {
			slog.Warn("scheduler: failed to register pipeline schedule stage",
				"pipeline", pipeline.ID, "cron", stage.Config.Cron, "err", err)
			continue
		}
		stage.Config.ScheduleID = sched.ID
	}
	return nil
}

func (s *SchedulerService) RemovePipelineSchedules(ctx context.Context, pipelineID string) error {
	schedules, err := s.scheduleRepo.ListByPipeline(ctx, pipelineID)
	if err != nil {
		slog.Warn("scheduler: failed to list pipeline schedules for removal", "pipeline", pipelineID, "err", err)
	}
	for _, sched := range schedules {
		if err := s.RemoveSchedule(ctx, sched.ID); err != nil {
			slog.Warn("scheduler: failed to remove pipeline schedule", "id", sched.ID, "err", err)
		}
	}
	return nil
}
