package services

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

func newNodeRun(nodeID, status string) upal.NodeRunRecord {
	return upal.NodeRunRecord{
		NodeID:    nodeID,
		Status:    status,
		StartedAt: time.Now(),
	}
}

func TestRunHistoryService_StartAndComplete(t *testing.T) {
	repo := repository.NewMemoryRunRepository()
	svc := NewRunHistoryService(repo)
	ctx := context.Background()

	// Start a run.
	record, err := svc.StartRun(ctx, "test-workflow", "manual", "", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if record.ID == "" {
		t.Fatal("expected non-empty run ID")
	}
	if record.Status != "running" {
		t.Fatalf("expected status running, got %s", record.Status)
	}

	// Complete the run.
	outputs := map[string]any{"result": "ok"}
	if err := svc.CompleteRun(ctx, record.ID, outputs); err != nil {
		t.Fatalf("CompleteRun: %v", err)
	}

	// Verify.
	got, err := svc.GetRun(ctx, record.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != "success" {
		t.Fatalf("expected status success, got %s", got.Status)
	}
	if got.Outputs["result"] != "ok" {
		t.Fatalf("expected output result=ok, got %v", got.Outputs)
	}
	if got.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestRunHistoryService_StartAndFail(t *testing.T) {
	repo := repository.NewMemoryRunRepository()
	svc := NewRunHistoryService(repo)
	ctx := context.Background()

	record, err := svc.StartRun(ctx, "test-workflow", "cron", "sched-123", nil)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}

	if err := svc.FailRun(ctx, record.ID, "timeout"); err != nil {
		t.Fatalf("FailRun: %v", err)
	}

	got, err := svc.GetRun(ctx, record.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Status != "failed" {
		t.Fatalf("expected status failed, got %s", got.Status)
	}
	if got.Error == nil || *got.Error != "timeout" {
		t.Fatalf("expected error=timeout, got %v", got.Error)
	}
}

func TestRunHistoryService_ListRuns(t *testing.T) {
	repo := repository.NewMemoryRunRepository()
	svc := NewRunHistoryService(repo)
	ctx := context.Background()

	// Create 3 runs for wf-a, 2 for wf-b.
	for i := 0; i < 3; i++ {
		svc.StartRun(ctx, "wf-a", "manual", "", nil)
	}
	for i := 0; i < 2; i++ {
		svc.StartRun(ctx, "wf-b", "manual", "", nil)
	}

	// List all.
	all, total, err := svc.ListAllRuns(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListAllRuns: %v", err)
	}
	if total != 5 {
		t.Fatalf("expected total=5, got %d", total)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5 runs, got %d", len(all))
	}

	// List by workflow.
	wfaRuns, wfaTotal, err := svc.ListRuns(ctx, "wf-a", 10, 0)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if wfaTotal != 3 {
		t.Fatalf("expected total=3 for wf-a, got %d", wfaTotal)
	}
	if len(wfaRuns) != 3 {
		t.Fatalf("expected 3 runs for wf-a, got %d", len(wfaRuns))
	}

	// Pagination.
	page, pageTotal, err := svc.ListAllRuns(ctx, 2, 0)
	if err != nil {
		t.Fatalf("ListAllRuns paginated: %v", err)
	}
	if pageTotal != 5 {
		t.Fatalf("expected total=5, got %d", pageTotal)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 runs on page, got %d", len(page))
	}
}

func TestRunHistoryService_UpdateNodeRun(t *testing.T) {
	repo := repository.NewMemoryRunRepository()
	svc := NewRunHistoryService(repo)
	ctx := context.Background()

	record, _ := svc.StartRun(ctx, "test-wf", "manual", "", nil)

	// Add a node run.
	svc.UpdateNodeRun(ctx, record.ID, newNodeRun("node-1", "running"))

	got, _ := svc.GetRun(ctx, record.ID)
	if len(got.NodeRuns) != 1 {
		t.Fatalf("expected 1 node run, got %d", len(got.NodeRuns))
	}
	if got.NodeRuns[0].NodeID != "node-1" {
		t.Fatalf("expected node-1, got %s", got.NodeRuns[0].NodeID)
	}

	// Update existing node run.
	svc.UpdateNodeRun(ctx, record.ID, newNodeRun("node-1", "completed"))

	got, _ = svc.GetRun(ctx, record.ID)
	if len(got.NodeRuns) != 1 {
		t.Fatalf("expected 1 node run after update, got %d", len(got.NodeRuns))
	}
	if got.NodeRuns[0].Status != "completed" {
		t.Fatalf("expected completed, got %s", got.NodeRuns[0].Status)
	}
}
