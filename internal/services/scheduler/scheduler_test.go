package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// noopLimiter satisfies ports.ConcurrencyControl for tests that don't need limiting.
type noopLimiter struct{}

func (noopLimiter) Acquire(_ context.Context, _ string) error { return nil }
func (noopLimiter) Release(_ string)                          {}

func TestParseCronExpr_5Field(t *testing.T) {
	sched, err := parseCronExpr("*/5 * * * *", "")
	if err != nil {
		t.Fatalf("expected 5-field expression to parse, got error: %v", err)
	}
	next := sched.Next(time.Now())
	if next.IsZero() {
		t.Fatal("expected non-zero next time")
	}
}

func TestParseCronExpr_6Field(t *testing.T) {
	sched, err := parseCronExpr("0 */5 * * * *", "")
	if err != nil {
		t.Fatalf("expected 6-field expression to parse, got error: %v", err)
	}
	next := sched.Next(time.Now())
	if next.IsZero() {
		t.Fatal("expected non-zero next time")
	}
}

func TestParseCronExpr_Invalid(t *testing.T) {
	_, err := parseCronExpr("invalid cron", "")
	if err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

func TestSchedulerService_AddSchedule_5Field(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

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
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

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
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

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

func TestSchedulerService_RemoveSchedule(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

	ctx := context.Background()
	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
	}

	if err := svc.AddSchedule(ctx, schedule); err != nil {
		t.Fatalf("AddSchedule failed: %v", err)
	}

	id := schedule.ID

	// Remove the schedule.
	if err := svc.RemoveSchedule(ctx, id); err != nil {
		t.Fatalf("RemoveSchedule failed: %v", err)
	}

	// Verify not in repo.
	_, err := repo.Get(ctx, id)
	if err == nil {
		t.Fatal("expected Get to return error after removal, got nil")
	}

	// Verify not in entryMap.
	svc.mu.RLock()
	_, registered := svc.entryMap[id]
	svc.mu.RUnlock()
	if registered {
		t.Fatal("expected schedule to be removed from cron entryMap")
	}

	svc.Stop()
}

func TestSchedulerService_UpdateSchedule(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

	ctx := context.Background()
	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
	}

	if err := svc.AddSchedule(ctx, schedule); err != nil {
		t.Fatalf("AddSchedule failed: %v", err)
	}

	// Record old entry ID.
	svc.mu.RLock()
	oldEntryID := svc.entryMap[schedule.ID]
	svc.mu.RUnlock()

	// Update cron expression.
	schedule.CronExpr = "*/10 * * * *"
	if err := svc.UpdateSchedule(ctx, schedule); err != nil {
		t.Fatalf("UpdateSchedule failed: %v", err)
	}

	// Verify repo has the new cron expression.
	stored, err := repo.Get(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("expected schedule in repo: %v", err)
	}
	if stored.CronExpr != "*/10 * * * *" {
		t.Fatalf("expected cron expr %q, got %q", "*/10 * * * *", stored.CronExpr)
	}

	// Verify entryMap key still exists with a new entry ID (old one replaced).
	svc.mu.RLock()
	newEntryID, registered := svc.entryMap[schedule.ID]
	svc.mu.RUnlock()
	if !registered {
		t.Fatal("expected schedule to be registered in cron entryMap after update")
	}
	if newEntryID == oldEntryID {
		t.Fatal("expected entryMap to have a new entry ID after update")
	}

	svc.Stop()
}

func TestSchedulerService_AddSchedule_Disabled(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

	ctx := context.Background()
	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      false,
	}

	if err := svc.AddSchedule(ctx, schedule); err != nil {
		t.Fatalf("AddSchedule failed: %v", err)
	}

	// Verify stored in repo.
	stored, err := repo.Get(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("expected schedule in repo: %v", err)
	}
	if stored.WorkflowName != "test-workflow" {
		t.Fatalf("expected workflow name %q, got %q", "test-workflow", stored.WorkflowName)
	}

	// Verify NOT in entryMap.
	svc.mu.RLock()
	_, registered := svc.entryMap[schedule.ID]
	svc.mu.RUnlock()
	if registered {
		t.Fatal("expected disabled schedule to NOT be registered in cron entryMap")
	}

	svc.Stop()
}

func TestSchedulerService_AddSchedule_InvalidCron(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

	ctx := context.Background()
	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "invalid",
		Enabled:      true,
	}

	err := svc.AddSchedule(ctx, schedule)
	if err == nil {
		t.Fatal("expected error for invalid cron expression, got nil")
	}

	// Verify nothing stored in repo.
	list, listErr := repo.List(ctx)
	if listErr != nil {
		t.Fatalf("List failed: %v", listErr)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 schedules in repo, got %d", len(list))
	}

	svc.Stop()
}

func TestSchedulerService_AddSchedule_DefaultTimezone(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)

	ctx := context.Background()
	schedule := &upal.Schedule{
		WorkflowName: "test-workflow",
		CronExpr:     "*/5 * * * *",
		Enabled:      true,
	}
	// Timezone deliberately left empty.

	if err := svc.AddSchedule(ctx, schedule); err != nil {
		t.Fatalf("AddSchedule failed: %v", err)
	}

	if schedule.Timezone != "UTC" {
		t.Fatalf("expected Timezone %q, got %q", "UTC", schedule.Timezone)
	}

	// Also verify via repo.
	stored, err := repo.Get(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("expected schedule in repo: %v", err)
	}
	if stored.Timezone != "UTC" {
		t.Fatalf("expected stored Timezone %q, got %q", "UTC", stored.Timezone)
	}

	svc.Stop()
}

func TestSchedulerService_Start_LoadsExisting(t *testing.T) {
	repo := repository.NewMemoryScheduleRepository()
	ctx := context.Background()

	// Seed schedules directly into repo.
	sched1 := &upal.Schedule{ID: "sched-1", WorkflowName: "wf1", CronExpr: "*/5 * * * *", Enabled: true}
	if err := repo.Create(ctx, sched1); err != nil {
		t.Fatalf("failed to seed sched1: %v", err)
	}
	sched2 := &upal.Schedule{ID: "sched-2", WorkflowName: "wf2", CronExpr: "*/10 * * * *", Enabled: true}
	if err := repo.Create(ctx, sched2); err != nil {
		t.Fatalf("failed to seed sched2: %v", err)
	}
	sched3 := &upal.Schedule{ID: "sched-3", WorkflowName: "wf3", CronExpr: "*/15 * * * *", Enabled: false}
	if err := repo.Create(ctx, sched3); err != nil {
		t.Fatalf("failed to seed sched3: %v", err)
	}

	svc := NewSchedulerService(repo, nil, nil, noopLimiter{}, nil)
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer svc.Stop()

	// Check entryMap has sched-1 and sched-2 but not sched-3.
	svc.mu.RLock()
	_, has1 := svc.entryMap["sched-1"]
	_, has2 := svc.entryMap["sched-2"]
	_, has3 := svc.entryMap["sched-3"]
	svc.mu.RUnlock()

	if !has1 {
		t.Fatal("expected sched-1 to be registered in entryMap")
	}
	if !has2 {
		t.Fatal("expected sched-2 to be registered in entryMap")
	}
	if has3 {
		t.Fatal("expected sched-3 (disabled) to NOT be registered in entryMap")
	}
}
