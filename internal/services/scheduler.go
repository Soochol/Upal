package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// parseCronExpr tries 6-field (with seconds) then 5-field (standard) parsing.
func parseCronExpr(expr string) (cron.Schedule, error) {
	parser6 := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser6.Parse(expr)
	if err == nil {
		return sched, nil
	}
	parser5 := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return parser5.Parse(expr)
}

// SchedulerService manages cron-based workflow scheduling.
// It wraps robfig/cron and integrates retry, concurrency, and run history.
type SchedulerService struct {
	cron          *cron.Cron
	scheduleRepo  repository.ScheduleRepository
	workflowSvc   *WorkflowService
	retryExecutor *RetryExecutor
	limiter       *ConcurrencyLimiter
	runHistorySvc *RunHistoryService
	entryMap      map[string]cron.EntryID // schedule ID → cron entry
	mu            sync.RWMutex
}

// NewSchedulerService creates a SchedulerService with all dependencies.
func NewSchedulerService(
	scheduleRepo repository.ScheduleRepository,
	workflowSvc *WorkflowService,
	retryExecutor *RetryExecutor,
	limiter *ConcurrencyLimiter,
	runHistorySvc *RunHistoryService,
) *SchedulerService {
	return &SchedulerService{
		cron:          cron.New(cron.WithSeconds()),
		scheduleRepo:  scheduleRepo,
		workflowSvc:   workflowSvc,
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
	cronSched, err := parseCronExpr(schedule.CronExpr)
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

// registerCronJob registers a cron job for the given schedule.
// Uses cron.Schedule() with a pre-parsed schedule to support both 5-field and 6-field expressions.
func (s *SchedulerService) registerCronJob(schedule *upal.Schedule) error {
	cronSched, err := parseCronExpr(schedule.CronExpr)
	if err != nil {
		return err
	}

	entryID := s.cron.Schedule(cronSched, cron.FuncJob(func() {
		s.executeScheduledRun(schedule)
	}))

	s.mu.Lock()
	s.entryMap[schedule.ID] = entryID
	s.mu.Unlock()

	slog.Info("scheduler: registered cron job",
		"id", schedule.ID, "workflow", schedule.WorkflowName, "cron", schedule.CronExpr)
	return nil
}

// executeScheduledRun is called by cron when a schedule fires.
func (s *SchedulerService) executeScheduledRun(schedule *upal.Schedule) {
	ctx := context.Background()

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
	wf, err := s.workflowSvc.Lookup(ctx, schedule.WorkflowName)
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
	cronSched, parseErr := parseCronExpr(schedule.CronExpr)
	if parseErr == nil {
		schedule.NextRunAt = cronSched.Next(now)
	}
	schedule.UpdatedAt = now

	if updateErr := s.scheduleRepo.Update(ctx, schedule); updateErr != nil {
		slog.Warn("scheduler: failed to update schedule after run", "err", updateErr)
	}
}
