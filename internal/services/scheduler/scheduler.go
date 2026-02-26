package scheduler

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

var _ ports.SchedulerPort = (*SchedulerService)(nil)

// SchedulerService manages cron-based workflow and pipeline scheduling.
type SchedulerService struct {
	cron           *cron.Cron
	scheduleRepo   repository.ScheduleRepository
	workflowExec   ports.WorkflowExecutor
	retryExecutor  ports.RetryExecutor
	limiter        ports.ConcurrencyControl
	runHistorySvc  ports.RunHistoryPort
	entryMap       map[string]cron.EntryID // schedule ID → cron entry
	mu             sync.RWMutex
	pipelineRunner     ports.PipelineRunner
	pipelineSvc        ports.PipelineRegistry
	contentCollector   ContentCollector
}

type ContentCollector interface {
	CollectPipeline(ctx context.Context, pipelineID string) error
}

func (s *SchedulerService) SetPipelineRunner(runner ports.PipelineRunner) {
	s.pipelineRunner = runner
}

func (s *SchedulerService) SetPipelineService(svc ports.PipelineRegistry) {
	s.pipelineSvc = svc
}

func (s *SchedulerService) SetContentCollector(c ContentCollector) {
	s.contentCollector = c
}

func NewSchedulerService(
	scheduleRepo repository.ScheduleRepository,
	workflowExec ports.WorkflowExecutor,
	retryExecutor ports.RetryExecutor,
	limiter ports.ConcurrencyControl,
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

func (s *SchedulerService) Start(ctx context.Context) error {
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

func (s *SchedulerService) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("scheduler: stopped")
}

func (s *SchedulerService) AddSchedule(ctx context.Context, schedule *upal.Schedule) error {
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

func (s *SchedulerService) RemoveSchedule(ctx context.Context, id string) error {
	s.mu.Lock()
	if entryID, ok := s.entryMap[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entryMap, id)
	}
	s.mu.Unlock()

	return s.scheduleRepo.Delete(ctx, id)
}

func (s *SchedulerService) UpdateSchedule(ctx context.Context, schedule *upal.Schedule) error {
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

func (s *SchedulerService) GetSchedule(ctx context.Context, id string) (*upal.Schedule, error) {
	return s.scheduleRepo.Get(ctx, id)
}

func (s *SchedulerService) ListSchedules(ctx context.Context) ([]*upal.Schedule, error) {
	return s.scheduleRepo.List(ctx)
}

func (s *SchedulerService) TriggerNow(ctx context.Context, id string) error {
	schedule, err := s.scheduleRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	s.executeScheduledRun(schedule)
	return nil
}
