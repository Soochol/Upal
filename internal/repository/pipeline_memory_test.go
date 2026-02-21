// internal/repository/pipeline_memory_test.go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestMemoryPipelineRepo_CRUD(t *testing.T) {
	repo := NewMemoryPipelineRepository()
	ctx := context.Background()

	p := &upal.Pipeline{
		ID:   "pipe-test1",
		Name: "Test Pipeline",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Collect", Type: "workflow"},
		},
		CreatedAt: time.Now(),
	}

	// Create
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, "pipe-test1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "Test Pipeline" {
		t.Errorf("expected name 'Test Pipeline', got %q", got.Name)
	}

	// List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(list))
	}

	// Update
	p.Name = "Updated"
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	got, _ = repo.Get(ctx, "pipe-test1")
	if got.Name != "Updated" {
		t.Errorf("expected updated name, got %q", got.Name)
	}

	// Delete
	if err := repo.Delete(ctx, "pipe-test1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	_, err = repo.Get(ctx, "pipe-test1")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestMemoryPipelineRepo_DuplicateCreate(t *testing.T) {
	repo := NewMemoryPipelineRepository()
	ctx := context.Background()
	p := &upal.Pipeline{ID: "pipe-dup", Name: "Dup"}
	repo.Create(ctx, p)
	if err := repo.Create(ctx, p); err == nil {
		t.Error("expected error on duplicate create")
	}
}

func TestMemoryPipelineRunRepo_CRUD(t *testing.T) {
	repo := NewMemoryPipelineRunRepository()
	ctx := context.Background()

	run := &upal.PipelineRun{
		ID:         "prun-1",
		PipelineID: "pipe-1",
		Status:     "running",
		StartedAt:  time.Now(),
	}

	if err := repo.Create(ctx, run); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	got, err := repo.Get(ctx, "prun-1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("expected status 'running', got %q", got.Status)
	}

	runs, err := repo.ListByPipeline(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}

	run.Status = "completed"
	repo.Update(ctx, run)
	got, _ = repo.Get(ctx, "prun-1")
	if got.Status != "completed" {
		t.Errorf("expected updated status, got %q", got.Status)
	}
}
