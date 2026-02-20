package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestMemoryScheduleRepo_CreateAndGet(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	retry := &upal.RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      5 * time.Minute,
		BackoffFactor: 2.0,
	}

	schedule := &upal.Schedule{
		ID:           "sched-001",
		WorkflowName: "daily-report",
		CronExpr:     "0 9 * * *",
		Inputs:       map[string]any{"format": "pdf", "pages": 10},
		Enabled:      true,
		Timezone:     "America/New_York",
		RetryPolicy:  retry,
		NextRunAt:    now.Add(24 * time.Hour),
		LastRunAt:    nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(ctx, schedule); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	got, err := repo.Get(ctx, "sched-001")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if got.ID != schedule.ID {
		t.Errorf("ID = %q, want %q", got.ID, schedule.ID)
	}
	if got.WorkflowName != schedule.WorkflowName {
		t.Errorf("WorkflowName = %q, want %q", got.WorkflowName, schedule.WorkflowName)
	}
	if got.CronExpr != schedule.CronExpr {
		t.Errorf("CronExpr = %q, want %q", got.CronExpr, schedule.CronExpr)
	}
	if got.Enabled != schedule.Enabled {
		t.Errorf("Enabled = %v, want %v", got.Enabled, schedule.Enabled)
	}
	if got.Timezone != schedule.Timezone {
		t.Errorf("Timezone = %q, want %q", got.Timezone, schedule.Timezone)
	}
	if got.RetryPolicy == nil {
		t.Fatal("RetryPolicy is nil, want non-nil")
	}
	if got.RetryPolicy.MaxRetries != retry.MaxRetries {
		t.Errorf("RetryPolicy.MaxRetries = %d, want %d", got.RetryPolicy.MaxRetries, retry.MaxRetries)
	}
	if !got.NextRunAt.Equal(schedule.NextRunAt) {
		t.Errorf("NextRunAt = %v, want %v", got.NextRunAt, schedule.NextRunAt)
	}
	if got.LastRunAt != nil {
		t.Errorf("LastRunAt = %v, want nil", got.LastRunAt)
	}
	if !got.CreatedAt.Equal(schedule.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, schedule.CreatedAt)
	}
	if !got.UpdatedAt.Equal(schedule.UpdatedAt) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, schedule.UpdatedAt)
	}
	if fmt.Sprintf("%v", got.Inputs) != fmt.Sprintf("%v", schedule.Inputs) {
		t.Errorf("Inputs = %v, want %v", got.Inputs, schedule.Inputs)
	}
}

func TestMemoryScheduleRepo_Get_NotFound(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Get returned nil error for nonexistent schedule")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestMemoryScheduleRepo_Update(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	schedule := &upal.Schedule{
		ID:           "sched-upd",
		WorkflowName: "etl-pipeline",
		CronExpr:     "0 0 * * *",
		Inputs:       map[string]any{"source": "s3"},
		Enabled:      true,
		Timezone:     "UTC",
		NextRunAt:    now.Add(time.Hour),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(ctx, schedule); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Modify fields
	lastRun := now.Add(30 * time.Minute)
	updated := &upal.Schedule{
		ID:           "sched-upd",
		WorkflowName: "etl-pipeline-v2",
		CronExpr:     "*/30 * * * *",
		Inputs:       map[string]any{"source": "gcs", "bucket": "data"},
		Enabled:      false,
		Timezone:     "Europe/London",
		NextRunAt:    now.Add(2 * time.Hour),
		LastRunAt:    &lastRun,
		CreatedAt:    now,
		UpdatedAt:    now.Add(time.Minute),
	}

	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	got, err := repo.Get(ctx, "sched-upd")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if got.WorkflowName != "etl-pipeline-v2" {
		t.Errorf("WorkflowName = %q, want %q", got.WorkflowName, "etl-pipeline-v2")
	}
	if got.CronExpr != "*/30 * * * *" {
		t.Errorf("CronExpr = %q, want %q", got.CronExpr, "*/30 * * * *")
	}
	if got.Enabled != false {
		t.Error("Enabled = true, want false")
	}
	if got.Timezone != "Europe/London" {
		t.Errorf("Timezone = %q, want %q", got.Timezone, "Europe/London")
	}
	if got.LastRunAt == nil {
		t.Fatal("LastRunAt is nil after update, want non-nil")
	}
	if !got.LastRunAt.Equal(lastRun) {
		t.Errorf("LastRunAt = %v, want %v", *got.LastRunAt, lastRun)
	}
	if !got.UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, now.Add(time.Minute))
	}
}

func TestMemoryScheduleRepo_Update_NotFound(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	schedule := &upal.Schedule{
		ID:           "nonexistent",
		WorkflowName: "ghost",
	}

	err := repo.Update(ctx, schedule)
	if err == nil {
		t.Fatal("Update returned nil error for nonexistent schedule")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Update error = %v, want ErrNotFound", err)
	}
}

func TestMemoryScheduleRepo_Delete(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	schedule := &upal.Schedule{
		ID:           "sched-del",
		WorkflowName: "cleanup",
		CronExpr:     "0 0 * * 0",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := repo.Create(ctx, schedule); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, "sched-del"); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	_, err := repo.Get(ctx, "sched-del")
	if err == nil {
		t.Fatal("Get returned nil error after delete")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}
}

func TestMemoryScheduleRepo_Delete_NotFound(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Delete returned nil error for nonexistent schedule")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete error = %v, want ErrNotFound", err)
	}
}

func TestMemoryScheduleRepo_List(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 3; i++ {
		s := &upal.Schedule{
			ID:           fmt.Sprintf("sched-%d", i),
			WorkflowName: fmt.Sprintf("workflow-%d", i),
			CronExpr:     "0 * * * *",
			Enabled:      true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create(%d) returned unexpected error: %v", i, err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List returned %d schedules, want 3", len(list))
	}

	// Verify all IDs are present
	ids := make(map[string]bool)
	for _, s := range list {
		ids[s.ID] = true
	}
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("sched-%d", i)
		if !ids[id] {
			t.Errorf("List missing schedule with ID %q", id)
		}
	}
}

func TestMemoryScheduleRepo_List_Empty(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List returned %d schedules, want 0", len(list))
	}
}

func TestMemoryScheduleRepo_ListDue(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)

	// Enabled schedule with NextRunAt in the past — should be returned
	pastEnabled := &upal.Schedule{
		ID:        "sched-past-enabled",
		Enabled:   true,
		NextRunAt: now.Add(-time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Enabled schedule with NextRunAt in the future — should NOT be returned
	futureEnabled := &upal.Schedule{
		ID:        "sched-future-enabled",
		Enabled:   true,
		NextRunAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Disabled schedule with NextRunAt in the past — should NOT be returned
	pastDisabled := &upal.Schedule{
		ID:        "sched-past-disabled",
		Enabled:   false,
		NextRunAt: now.Add(-time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, s := range []*upal.Schedule{pastEnabled, futureEnabled, pastDisabled} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create(%s) returned unexpected error: %v", s.ID, err)
		}
	}

	due, err := repo.ListDue(ctx, now)
	if err != nil {
		t.Fatalf("ListDue returned unexpected error: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("ListDue returned %d schedules, want 1", len(due))
	}
	if due[0].ID != "sched-past-enabled" {
		t.Errorf("ListDue[0].ID = %q, want %q", due[0].ID, "sched-past-enabled")
	}
}

func TestMemoryScheduleRepo_ConcurrentAccess(t *testing.T) {
	repo := NewMemoryScheduleRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	goroutines := 10

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()

			id := fmt.Sprintf("sched-concurrent-%d", n)
			now := time.Now()

			s := &upal.Schedule{
				ID:           id,
				WorkflowName: fmt.Sprintf("workflow-%d", n),
				CronExpr:     "* * * * *",
				Enabled:      true,
				NextRunAt:    now.Add(time.Hour),
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			if err := repo.Create(ctx, s); err != nil {
				t.Errorf("goroutine %d: Create error: %v", n, err)
				return
			}

			if _, err := repo.Get(ctx, id); err != nil {
				t.Errorf("goroutine %d: Get error: %v", n, err)
				return
			}

			if _, err := repo.List(ctx); err != nil {
				t.Errorf("goroutine %d: List error: %v", n, err)
				return
			}

			if _, err := repo.ListDue(ctx, time.Now().Add(2*time.Hour)); err != nil {
				t.Errorf("goroutine %d: ListDue error: %v", n, err)
				return
			}
		}(i)
	}

	wg.Wait()

	// Verify all schedules were created
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List after concurrent access returned error: %v", err)
	}
	if len(list) != goroutines {
		t.Errorf("List returned %d schedules after concurrent access, want %d", len(list), goroutines)
	}
}
