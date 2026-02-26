package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestMemorySessionRunRepository_CRUD(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	run := &upal.Run{
		ID:          "run-1",
		SessionID:   "sess-1",
		Status:      upal.SessionRunCollecting,
		TriggerType: "manual",
		CreatedAt:   time.Now(),
	}

	// Create
	if err := repo.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, "run-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != upal.SessionRunCollecting {
		t.Errorf("expected status collecting, got %q", got.Status)
	}
	if got.SessionID != "sess-1" {
		t.Errorf("expected session_id 'sess-1', got %q", got.SessionID)
	}

	// List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 run, got %d", len(list))
	}

	// Update
	run.Status = upal.SessionRunPendingReview
	if err := repo.Update(ctx, run); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = repo.Get(ctx, "run-1")
	if got.Status != upal.SessionRunPendingReview {
		t.Errorf("expected pending_review, got %q", got.Status)
	}

	// Delete
	if err := repo.Delete(ctx, "run-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.Get(ctx, "run-1")
	if err == nil {
		t.Error("expected error after Delete, got nil")
	}
}

func TestMemorySessionRunRepository_GetNotFound(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing run, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemorySessionRunRepository_DuplicateCreate(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	run := &upal.Run{ID: "run-dup", SessionID: "sess-1", Status: upal.SessionRunCollecting, CreatedAt: time.Now()}
	if err := repo.Create(ctx, run); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := repo.Create(ctx, run); err == nil {
		t.Error("expected error on duplicate Create, got nil")
	}
}

func TestMemorySessionRunRepository_ListBySession(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	runs := []*upal.Run{
		{ID: "run-a1", SessionID: "sess-a", Status: upal.SessionRunCollecting, CreatedAt: time.Now()},
		{ID: "run-a2", SessionID: "sess-a", Status: upal.SessionRunAnalyzing, CreatedAt: time.Now()},
		{ID: "run-b1", SessionID: "sess-b", Status: upal.SessionRunCollecting, CreatedAt: time.Now()},
	}
	for _, r := range runs {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.ID, err)
		}
	}

	sessA, err := repo.ListBySession(ctx, "sess-a")
	if err != nil {
		t.Fatalf("ListBySession: %v", err)
	}
	if len(sessA) != 2 {
		t.Errorf("expected 2 runs for sess-a, got %d", len(sessA))
	}

	sessB, err := repo.ListBySession(ctx, "sess-b")
	if err != nil {
		t.Fatalf("ListBySession: %v", err)
	}
	if len(sessB) != 1 {
		t.Errorf("expected 1 run for sess-b, got %d", len(sessB))
	}

	sessC, err := repo.ListBySession(ctx, "sess-c")
	if err != nil {
		t.Fatalf("ListBySession: %v", err)
	}
	if len(sessC) != 0 {
		t.Errorf("expected 0 runs for sess-c, got %d", len(sessC))
	}
}

func TestMemorySessionRunRepository_ListByStatus(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	runs := []*upal.Run{
		{ID: "run-s1", SessionID: "sess-1", Status: upal.SessionRunCollecting, CreatedAt: time.Now()},
		{ID: "run-s2", SessionID: "sess-1", Status: upal.SessionRunPendingReview, CreatedAt: time.Now()},
		{ID: "run-s3", SessionID: "sess-2", Status: upal.SessionRunPendingReview, CreatedAt: time.Now()},
		{ID: "run-s4", SessionID: "sess-2", Status: upal.SessionRunApproved, CreatedAt: time.Now()},
	}
	for _, r := range runs {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.ID, err)
		}
	}

	collecting, err := repo.ListByStatus(ctx, upal.SessionRunCollecting)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(collecting) != 1 {
		t.Errorf("expected 1 collecting, got %d", len(collecting))
	}

	pending, err := repo.ListByStatus(ctx, upal.SessionRunPendingReview)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending_review, got %d", len(pending))
	}

	approved, err := repo.ListByStatus(ctx, upal.SessionRunApproved)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(approved) != 1 {
		t.Errorf("expected 1 approved, got %d", len(approved))
	}

	published, err := repo.ListByStatus(ctx, upal.SessionRunPublished)
	if err != nil {
		t.Fatalf("ListByStatus: %v", err)
	}
	if len(published) != 0 {
		t.Errorf("expected 0 published, got %d", len(published))
	}
}

func TestMemorySessionRunRepository_DeleteBySession(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	runs := []*upal.Run{
		{ID: "run-d1", SessionID: "sess-del", Status: upal.SessionRunCollecting, CreatedAt: time.Now()},
		{ID: "run-d2", SessionID: "sess-del", Status: upal.SessionRunAnalyzing, CreatedAt: time.Now()},
		{ID: "run-d3", SessionID: "sess-keep", Status: upal.SessionRunCollecting, CreatedAt: time.Now()},
	}
	for _, r := range runs {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.ID, err)
		}
	}

	if err := repo.DeleteBySession(ctx, "sess-del"); err != nil {
		t.Fatalf("DeleteBySession: %v", err)
	}

	// Runs for deleted session should be gone
	deleted, err := repo.ListBySession(ctx, "sess-del")
	if err != nil {
		t.Fatalf("ListBySession after delete: %v", err)
	}
	if len(deleted) != 0 {
		t.Errorf("expected 0 runs for sess-del, got %d", len(deleted))
	}

	// Runs for other session should remain
	kept, err := repo.ListBySession(ctx, "sess-keep")
	if err != nil {
		t.Fatalf("ListBySession for kept: %v", err)
	}
	if len(kept) != 1 {
		t.Errorf("expected 1 run for sess-keep, got %d", len(kept))
	}
}

func TestMemorySessionRunRepository_DeleteNotFound(t *testing.T) {
	repo := NewMemorySessionRunRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for Delete on missing run, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryWorkflowRunRepository(t *testing.T) {
	repo := NewMemoryWorkflowRunRepository()
	ctx := context.Background()

	results := []upal.WorkflowRun{
		{WorkflowName: "blog-post", RunID: "wfrun-1", Status: upal.WFRunPending},
		{WorkflowName: "newsletter", RunID: "wfrun-2", Status: upal.WFRunRunning},
	}

	// Save
	if err := repo.Save(ctx, "run-1", results); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// GetByRun
	got, err := repo.GetByRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetByRun: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 workflow runs, got %d", len(got))
	}
	if got[0].WorkflowName != "blog-post" && got[1].WorkflowName != "blog-post" {
		t.Error("expected blog-post workflow run in results")
	}

	// GetByRun for nonexistent returns empty slice
	empty, err := repo.GetByRun(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetByRun nonexistent: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 results for nonexistent, got %d", len(empty))
	}

	// Verify Save returns a copy (mutation-safe)
	got[0].Status = upal.WFRunFailed
	original, _ := repo.GetByRun(ctx, "run-1")
	if original[0].Status == upal.WFRunFailed && original[1].Status == upal.WFRunFailed {
		t.Error("modifying returned slice should not affect stored data")
	}

	// DeleteByRun
	if err := repo.DeleteByRun(ctx, "run-1"); err != nil {
		t.Fatalf("DeleteByRun: %v", err)
	}
	after, err := repo.GetByRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetByRun after delete: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("expected 0 after delete, got %d", len(after))
	}
}

func TestMemoryWorkflowRunRepository_OverwriteOnSave(t *testing.T) {
	repo := NewMemoryWorkflowRunRepository()
	ctx := context.Background()

	first := []upal.WorkflowRun{
		{WorkflowName: "old-workflow", RunID: "wfrun-1", Status: upal.WFRunPending},
	}
	repo.Save(ctx, "run-1", first)

	second := []upal.WorkflowRun{
		{WorkflowName: "new-workflow", RunID: "wfrun-2", Status: upal.WFRunSuccess},
	}
	repo.Save(ctx, "run-1", second)

	got, _ := repo.GetByRun(ctx, "run-1")
	if len(got) != 1 {
		t.Fatalf("expected 1 result after overwrite, got %d", len(got))
	}
	if got[0].WorkflowName != "new-workflow" {
		t.Errorf("expected new-workflow, got %q", got[0].WorkflowName)
	}
}
