package services

// scheduler.go — SchedulerService public facade.
// Struct definition, constructor, and all public API methods live here.
// Cron-specific helpers: scheduler_cron.go
// Execution dispatch:    scheduler_dispatch.go
// Pipeline sync:         scheduler_sync.go

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

// SchedulerService manages cron-based workflow scheduling.
// It wraps robfig/cron and integrates retry, concurrency, and run history.
type SchedulerService struct {
	cron           *cron.Cron
	scheduleRepo   repository.ScheduleRepository
	workflowExec   ports.WorkflowExecutor
	retryExecutor  *RetryExecutor
	limiter        *ConcurrencyLimiter
	runHistorySvc  ports.RunHistoryPort
	entryMap       map[string]cron.EntryID // schedule ID → cron entry
	mu             sync.RWMutex
	pipelineRunner *PipelineRunner
	pipelineSvc    *PipelineService
}

// SetPipelineRunner configures the pipeline runner for scheduled pipeline execution.
func (s *SchedulerService) SetPipelineRunner(runner *PipelineRunner) {
	s.pipelineRunner = runner
}

// SetPipelineService configures the pipeline service for looking up pipelines.
func (s *SchedulerService) SetPipelineService(svc *PipelineService) {
	s.pipelineSvc = svc
}

// NewSchedulerService creates a SchedulerService with all dependencies.
func NewSchedulerService(
	scheduleRepo repository.ScheduleRepository,
	workflowExec ports.WorkflowExecutor,
	retryExecutor *RetryExecutor,
	limiter *ConcurrencyLimiter,
	runHistorySvc ports.RunHistoryPort,
) *SchedulerService {
	return &SchedulerService{
		cron:          cron.New(cron.WithSeconds()),
		scheduleRepo:  scheduleRepo,
		workflowExec:  workflowExec,
		retryExecutor: retryExecutor,
		limiter:       limiter,
		runHistorySvc: runHistorySvc,
		entryMap:      make(map[string]cron.EntryID),
	}
}

// Start begins the cron scheduler and loads existing schedules from the repository.
func (s *SchedulerService) Start(ctx context.Context) error {
	// Load existing enabled schedules.
	schedules, err := s.scheduleRepo.List(ctx)
	if err != nil {
		slog.Warn("scheduler: failed to load schedules", "err", err)
	} else {
		for _, sched := range schedules {
			if sched.Enabled {
				if err := s.registerCronJob(sched); err != nil {
					slog.Warn("scheduler: failed to register schedule",
						"id", sched.ID, "err", err)
				}
			}
		}
		slog.Info("scheduler: loaded schedules", "count", len(schedules))
	}

	s.cron.Start()
	slog.Info("scheduler: started")
	return nil
}

// Stop gracefully stops the cron scheduler.
func (s *SchedulerService) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("scheduler: stopped")
}

// AddSchedule creates a new schedule and registers its cron job.
func (s *SchedulerService) AddSchedule(ctx context.Context, schedule *upal.Schedule) error {
	// Compute next run time from cron expression.
	cronSched, err := parseCronExpr(schedule.CronExpr, schedule.Timezone)
	if err != nil {
		return err
	}

	now := time.Now()
	schedule.ID = upal.GenerateID("sched")
	schedule.NextRunAt = cronSched.Next(now)
	schedule.CreatedAt = now
	schedule.UpdatedAt = now
	if schedule.Timezone == "" {
		schedule.Timezone = "UTC"
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return err
	}

	if schedule.Enabled {
		return s.registerCronJob(schedule)
	}
	return nil
}

// RemoveSchedule removes a schedule and its cron job.
func (s *SchedulerService) RemoveSchedule(ctx context.Context, id string) error {
	s.mu.Lock()
	if entryID, ok := s.entryMap[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, id)
	}
	s.mu.Unlock()

	return s.scheduleRepo.Delete(ctx, id)
}

// UpdateSchedule updates a schedule and re-registers its cron job.
func (s *SchedulerService) UpdateSchedule(ctx context.Context, schedule *upal.Schedule) error {
	// Remove old cron job.
	s.mu.Lock()
	if entryID, ok := s.entryMap[schedule.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, schedule.ID)
	}
	s.mu.Unlock()

	schedule.UpdatedAt = time.Now()
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return err
	}

	if schedule.Enabled {
		return s.registerCronJob(schedule)
	}
	return nil
}

// PauseSchedule disables a schedule without deleting it.
func (s *SchedulerService) PauseSchedule(ctx context.Context, id string) error {
	schedule, err := s.scheduleRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if entryID, ok := s.entryMap[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, id)
	}
	s.mu.Unlock()

	schedule.Enabled = false
	schedule.UpdatedAt = time.Now()
	return s.scheduleRepo.Update(ctx, schedule)
}

// ResumeSchedule re-enables a paused schedule.
func (s *SchedulerService) ResumeSchedule(ctx context.Context, id string) error {
	schedule, err := s.scheduleRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	schedule.Enabled = true
	schedule.UpdatedAt = time.Now()

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return err
	}

	return s.registerCronJob(schedule)
}

// GetSchedule retrieves a schedule by ID.
func (s *SchedulerService) GetSchedule(ctx context.Context, id string) (*upal.Schedule, error) {
	return s.scheduleRepo.Get(ctx, id)
}

// ListSchedules returns all schedules.
func (s *SchedulerService) ListSchedules(ctx context.Context) ([]*upal.Schedule, error) {
	return s.scheduleRepo.List(ctx)
}

// TriggerNow immediately executes a scheduled workflow run, bypassing the cron
// timer. It looks up the schedule, runs the workflow synchronously, and updates
// the schedule's LastRunAt / NextRunAt timestamps — exactly the same path as a
// cron-triggered run.
func (s *SchedulerService) TriggerNow(ctx context.Context, id string) error {
	schedule, err := s.scheduleRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	s.executeScheduledRun(schedule)
	return nil
}
