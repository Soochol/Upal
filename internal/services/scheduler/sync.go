package scheduler

// sync.go — PipelineScheduleSync responsibility layer.
// Synchronises cron entries with pipeline schedule stages, implemented as
// methods on SchedulerService (same package, no circular import).

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// SyncPipelineSchedules synchronizes cron jobs with a pipeline's schedule stages.
// Call after creating or updating a pipeline.
// Stages with existing schedule_id are kept; new stages get a cron job registered.
// Orphaned pipeline schedules (no longer referenced) are removed.
// NOTE: pipeline.Stages is modified in-place; caller must re-save the pipeline.
func (s *SchedulerService) SyncPipelineSchedules(ctx context.Context, pipeline *upal.Pipeline) error {
	// Collect schedule IDs still referenced by stages.
	inUse := map[string]bool{}
	for _, stage := range pipeline.Stages {
		if stage.Config.ScheduleID != "" {
			inUse[stage.Config.ScheduleID] = true
		}
	}

	// Remove orphaned schedules for this pipeline.
	existing, _ := s.scheduleRepo.ListByPipeline(ctx, pipeline.ID)
	for _, sched := range existing {
		if !inUse[sched.ID] {
			if err := s.RemoveSchedule(ctx, sched.ID); err != nil {
				slog.Warn("scheduler: failed to remove orphaned schedule", "id", sched.ID, "err", err)
			}
		}
	}

	// Create cron jobs for new schedule stages without a schedule_id.
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

// RemovePipelineSchedules removes all cron jobs associated with a pipeline.
// Call before deleting a pipeline.
func (s *SchedulerService) RemovePipelineSchedules(ctx context.Context, pipelineID string) error {
	schedules, _ := s.scheduleRepo.ListByPipeline(ctx, pipelineID)
	for _, sched := range schedules {
		if err := s.RemoveSchedule(ctx, sched.ID); err != nil {
			slog.Warn("scheduler: failed to remove pipeline schedule", "id", sched.ID, "err", err)
		}
	}
	return nil
}
