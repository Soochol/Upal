package services_test

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

func newTestContentSvc() *services.ContentSessionService {
	return services.NewContentSessionService(
		repository.NewMemoryContentSessionRepository(),
		repository.NewMemorySourceFetchRepository(),
		repository.NewMemoryLLMAnalysisRepository(),
		repository.NewMemoryPublishedContentRepository(),
		repository.NewMemorySurgeEventRepository(),
	)
}

func TestContentSessionService_CreateAndApprove(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{
		PipelineID:  "pipe-1",
		TriggerType: "manual",
	}
	if err := svc.CreateSession(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	if s.ID == "" {
		t.Error("expected ID to be generated")
	}
	if s.Status != upal.SessionCollecting {
		t.Errorf("expected collecting, got %q", s.Status)
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if err := svc.ApproveSession(ctx, s.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, err := svc.GetSession(ctx, s.ID)
	if err != nil {
		t.Fatalf("get after approve: %v", err)
	}
	if got.Status != upal.SessionApproved {
		t.Errorf("expected approved, got %q", got.Status)
	}
	if got.ReviewedAt == nil {
		t.Error("expected ReviewedAt to be set on approval")
	}
}

func TestContentSessionService_Reject(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	if err := svc.CreateSession(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.RejectSession(ctx, s.ID); err != nil {
		t.Fatalf("reject: %v", err)
	}
	got, _ := svc.GetSession(ctx, s.ID)
	if got.Status != upal.SessionRejected {
		t.Errorf("expected rejected, got %q", got.Status)
	}
	if got.ReviewedAt == nil {
		t.Error("expected ReviewedAt to be set on rejection")
	}
}

func TestContentSessionService_ListByPipeline(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	svc.CreateSession(ctx, &upal.ContentSession{PipelineID: "pipe-A", TriggerType: "manual"})
	svc.CreateSession(ctx, &upal.ContentSession{PipelineID: "pipe-A", TriggerType: "manual"})
	svc.CreateSession(ctx, &upal.ContentSession{PipelineID: "pipe-B", TriggerType: "manual"})

	list, err := svc.ListSessionsByPipeline(ctx, "pipe-A")
	if err != nil {
		t.Fatalf("list by pipeline: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 sessions for pipe-A, got %d", len(list))
	}
}

func TestContentSessionService_DismissSurge(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	surge := &upal.SurgeEvent{Keyword: "DeepSeek", Multiplier: 10.0}
	if err := svc.CreateSurge(ctx, surge); err != nil {
		t.Fatalf("create surge: %v", err)
	}
	if surge.ID == "" {
		t.Error("expected surge ID to be generated")
	}
	active, _ := svc.ListActiveSurges(ctx)
	if len(active) != 1 {
		t.Fatalf("expected 1 active surge, got %d", len(active))
	}

	if err := svc.DismissSurge(ctx, surge.ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	active, _ = svc.ListActiveSurges(ctx)
	if len(active) != 0 {
		t.Fatalf("expected 0 active surges after dismiss, got %d", len(active))
	}
}

func TestContentSessionService_CreateValidation(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	err := svc.CreateSession(ctx, &upal.ContentSession{TriggerType: "manual"}) // missing pipeline_id
	if err == nil {
		t.Error("expected error when pipeline_id is empty")
	}
}
