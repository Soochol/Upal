package services

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

func TestParseCronExpr_5Field(t *testing.T) {
	sched, err := parseCronExpr("*/5 * * * *")
	if err != nil {
		t.Fatalf("expected 5-field expression to parse, got error: %v", err)
	}
	next := sched.Next(time.Now())
	if next.IsZero() {
		t.Fatal("expected non-zero next time")
	}
}

func TestParseCronExpr_6Field(t *testing.T) {
	sched, err := parseCronExpr("0 */5 * * * *")
	if err != nil {
		t.Fatalf("expected 6-field expression to parse, got error: %v", err)
	}
	next := sched.Next(time.Now())
	if next.IsZero() {
		t.Fatal("expected non-zero next time")
	}
}

func TestParseCronExpr_Invalid(t *testing.T) {
	_, err := parseCronExpr("invalid cron")
	if err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

func TestSchedulerService_AddSchedule_5Field(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, NewConcurrencyLimiter(upal.ConcurrencyLimits{}), nil)

	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
	}

	if err := svc.AddSchedule(context.Background(), schedule); err != nil {
		t.Fatalf("AddSchedule with 5-field expression failed: %v", err)
	}

	if schedule.ID == "" {
		t.Fatal("expected schedule ID to be set")
	}
	if schedule.NextRunAt.IsZero() {
		t.Fatal("expected NextRunAt to be set")
	}

	// Verify it was stored in the repository.
	stored, err := repo.Get(context.Background(), schedule.ID)
	if err != nil {
		t.Fatalf("expected schedule to be in repository: %v", err)
	}
	if stored.WorkflowName != "test-workflow" {
		t.Fatalf("expected workflow name %q, got %q", "test-workflow", stored.WorkflowName)
	}

	// Verify it was registered in cron (check entryMap).
	svc.mu.RLock()
	_, registered := svc.entryMap[schedule.ID]
	svc.mu.RUnlock()
	if !registered {
		t.Fatal("expected schedule to be registered in cron entryMap")
	}

	svc.Stop()
}

func TestSchedulerService_AddSchedule_6Field(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, NewConcurrencyLimiter(upal.ConcurrencyLimits{}), nil)

	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "0 */5 * * * *",
		Enabled:      true,
	}

	if err := svc.AddSchedule(context.Background(), schedule); err != nil {
		t.Fatalf("AddSchedule with 6-field expression failed: %v", err)
	}

	svc.mu.RLock()
	_, registered := svc.entryMap[schedule.ID]
	svc.mu.RUnlock()
	if !registered {
		t.Fatal("expected schedule to be registered in cron entryMap")
	}

	svc.Stop()
}

func TestSchedulerService_PauseResume(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, NewConcurrencyLimiter(upal.ConcurrencyLimits{}), nil)

	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
	}

	ctx := context.Background()
	if err := svc.AddSchedule(ctx, schedule); err != nil {
		t.Fatalf("AddSchedule failed: %v", err)
	}

	// Pause.
	if err := svc.PauseSchedule(ctx, schedule.ID); err != nil {
		t.Fatalf("PauseSchedule failed: %v", err)
	}

	stored, _ := repo.Get(ctx, schedule.ID)
	if stored.Enabled {
		t.Fatal("expected schedule to be disabled after pause")
	}

	svc.mu.RLock()
	_, registered := svc.entryMap[schedule.ID]
	svc.mu.RUnlock()
	if registered {
		t.Fatal("expected schedule to be removed from cron after pause")
	}

	// Resume.
	if err := svc.ResumeSchedule(ctx, schedule.ID); err != nil {
		t.Fatalf("ResumeSchedule failed: %v", err)
	}

	stored, _ = repo.Get(ctx, schedule.ID)
	if !stored.Enabled {
		t.Fatal("expected schedule to be enabled after resume")
	}

	svc.mu.RLock()
	_, registered = svc.entryMap[schedule.ID]
	svc.mu.RUnlock()
	if !registered {
		t.Fatal("expected schedule to be registered in cron after resume")
	}

	svc.Stop()
}
