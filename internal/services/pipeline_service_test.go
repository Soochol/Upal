// internal/services/pipeline_service_test.go
package services

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

func TestPipelineService_CreateAndGet(t *testing.T) {
	svc := NewPipelineService(
		repository.NewMemoryPipelineRepository(),
		repository.NewMemoryPipelineRunRepository(),
	)
	ctx := context.Background()

	p := &upal.Pipeline{
		Name: "Test Pipeline",
		Stages: []upal.Stage{
			{Name: "Collect", Type: "workflow", Config: upal.StageConfig{WorkflowName: "rss-collect"}},
			{Name: "Approve", Type: "approval", Config: upal.StageConfig{Message: "Pick a topic"}},
		},
	}

	if err := svc.Create(ctx, p); err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if p.ID == "" {
		t.Error("expected ID to be generated")
	}
	if len(p.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(p.Stages))
	}
	if p.Stages[0].ID == "" {
		t.Error("expected stage IDs to be generated")
	}

	got, err := svc.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "Test Pipeline" {
		t.Errorf("expected name 'Test Pipeline', got %q", got.Name)
	}
}

func TestPipelineService_List(t *testing.T) {
	svc := NewPipelineService(
		repository.NewMemoryPipelineRepository(),
		repository.NewMemoryPipelineRunRepository(),
	)
	ctx := context.Background()

	svc.Create(ctx, &upal.Pipeline{Name: "A"})
	svc.Create(ctx, &upal.Pipeline{Name: "B"})

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(list))
	}
}

func TestPipelineService_Delete(t *testing.T) {
	svc := NewPipelineService(
		repository.NewMemoryPipelineRepository(),
		repository.NewMemoryPipelineRunRepository(),
	)
	ctx := context.Background()

	p := &upal.Pipeline{Name: "ToDelete"}
	svc.Create(ctx, p)

	if err := svc.Delete(ctx, p.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, err := svc.Get(ctx, p.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}
