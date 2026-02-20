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

func TestMemoryTriggerRepo_CreateAndGet(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	trigger := &upal.Trigger{
		ID:           "trig-001",
		WorkflowName: "order-processor",
		Type:         upal.TriggerWebhook,
		Config: upal.TriggerConfig{
			Secret: "super-secret-token",
			InputMapping: map[string]string{
				"$.order_id": "order_id",
				"$.amount":   "total",
			},
		},
		Enabled:   true,
		CreatedAt: now,
	}

	if err := repo.Create(ctx, trigger); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	got, err := repo.Get(ctx, "trig-001")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if got.ID != trigger.ID {
		t.Errorf("ID = %q, want %q", got.ID, trigger.ID)
	}
	if got.WorkflowName != trigger.WorkflowName {
		t.Errorf("WorkflowName = %q, want %q", got.WorkflowName, trigger.WorkflowName)
	}
	if got.Type != upal.TriggerWebhook {
		t.Errorf("Type = %q, want %q", got.Type, upal.TriggerWebhook)
	}
	if got.Enabled != true {
		t.Error("Enabled = false, want true")
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
	if got.Config.Secret != "super-secret-token" {
		t.Errorf("Config.Secret = %q, want %q", got.Config.Secret, "super-secret-token")
	}
	if len(got.Config.InputMapping) != 2 {
		t.Fatalf("Config.InputMapping has %d entries, want 2", len(got.Config.InputMapping))
	}
	if got.Config.InputMapping["$.order_id"] != "order_id" {
		t.Errorf("Config.InputMapping[$.order_id] = %q, want %q", got.Config.InputMapping["$.order_id"], "order_id")
	}
	if got.Config.InputMapping["$.amount"] != "total" {
		t.Errorf("Config.InputMapping[$.amount] = %q, want %q", got.Config.InputMapping["$.amount"], "total")
	}
}

func TestMemoryTriggerRepo_Get_NotFound(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Get returned nil error for nonexistent trigger")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestMemoryTriggerRepo_Delete(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	trigger := &upal.Trigger{
		ID:           "trig-del",
		WorkflowName: "cleanup",
		Type:         upal.TriggerWebhook,
		Enabled:      true,
		CreatedAt:    time.Now(),
	}

	if err := repo.Create(ctx, trigger); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	if err := repo.Delete(ctx, "trig-del"); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	_, err := repo.Get(ctx, "trig-del")
	if err == nil {
		t.Fatal("Get returned nil error after delete")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}
}

func TestMemoryTriggerRepo_Delete_NotFound(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Delete returned nil error for nonexistent trigger")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete error = %v, want ErrNotFound", err)
	}
}

func TestMemoryTriggerRepo_ListByWorkflow(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	now := time.Now()

	// Two triggers for "wf-a"
	for i := 0; i < 2; i++ {
		tr := &upal.Trigger{
			ID:           fmt.Sprintf("trig-a-%d", i),
			WorkflowName: "wf-a",
			Type:         upal.TriggerWebhook,
			Enabled:      true,
			CreatedAt:    now,
		}
		if err := repo.Create(ctx, tr); err != nil {
			t.Fatalf("Create(trig-a-%d) returned unexpected error: %v", i, err)
		}
	}

	// One trigger for "wf-b"
	trB := &upal.Trigger{
		ID:           "trig-b-0",
		WorkflowName: "wf-b",
		Type:         upal.TriggerWebhook,
		Enabled:      true,
		CreatedAt:    now,
	}
	if err := repo.Create(ctx, trB); err != nil {
		t.Fatalf("Create(trig-b-0) returned unexpected error: %v", err)
	}

	// List for "wf-a" should return 2
	listA, err := repo.ListByWorkflow(ctx, "wf-a")
	if err != nil {
		t.Fatalf("ListByWorkflow(wf-a) returned unexpected error: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("ListByWorkflow(wf-a) returned %d triggers, want 2", len(listA))
	}
	for _, tr := range listA {
		if tr.WorkflowName != "wf-a" {
			t.Errorf("ListByWorkflow(wf-a) returned trigger with WorkflowName %q", tr.WorkflowName)
		}
	}

	// List for "wf-b" should return 1
	listB, err := repo.ListByWorkflow(ctx, "wf-b")
	if err != nil {
		t.Fatalf("ListByWorkflow(wf-b) returned unexpected error: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("ListByWorkflow(wf-b) returned %d triggers, want 1", len(listB))
	}
}

func TestMemoryTriggerRepo_ListByWorkflow_Empty(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	list, err := repo.ListByWorkflow(ctx, "unknown-workflow")
	if err != nil {
		t.Fatalf("ListByWorkflow returned unexpected error: %v", err)
	}
	if list != nil && len(list) != 0 {
		t.Errorf("ListByWorkflow returned %d triggers, want 0", len(list))
	}
}

func TestMemoryTriggerRepo_ConcurrentAccess(t *testing.T) {
	repo := NewMemoryTriggerRepository()
	ctx := context.Background()

	var wg sync.WaitGroup
	goroutines := 10

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()

			id := fmt.Sprintf("trig-concurrent-%d", n)
			workflowName := fmt.Sprintf("workflow-%d", n%3) // spread across 3 workflows

			tr := &upal.Trigger{
				ID:           id,
				WorkflowName: workflowName,
				Type:         upal.TriggerWebhook,
				Config: upal.TriggerConfig{
					Secret: fmt.Sprintf("secret-%d", n),
				},
				Enabled:   true,
				CreatedAt: time.Now(),
			}

			if err := repo.Create(ctx, tr); err != nil {
				t.Errorf("goroutine %d: Create error: %v", n, err)
				return
			}

			if _, err := repo.Get(ctx, id); err != nil {
				t.Errorf("goroutine %d: Get error: %v", n, err)
				return
			}

			if _, err := repo.ListByWorkflow(ctx, workflowName); err != nil {
				t.Errorf("goroutine %d: ListByWorkflow error: %v", n, err)
				return
			}
		}(i)
	}

	wg.Wait()

	// Verify all triggers exist by checking total count across all workflows
	total := 0
	for i := 0; i < 3; i++ {
		list, err := repo.ListByWorkflow(ctx, fmt.Sprintf("workflow-%d", i))
		if err != nil {
			t.Fatalf("ListByWorkflow(workflow-%d) returned error: %v", i, err)
		}
		total += len(list)
	}
	if total != goroutines {
		t.Errorf("total triggers across workflows = %d, want %d", total, goroutines)
	}
}
