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
		repository.NewMemoryWorkflowResultRepository(),
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

func TestContentSessionService_ArchiveAndUnarchive(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	if err := svc.CreateSession(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Archive
	if err := svc.ArchiveSession(ctx, s.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	got, _ := svc.GetSession(ctx, s.ID)
	if got.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be set after archive")
	}
	if got.Status != upal.SessionCollecting {
		t.Errorf("expected status preserved as collecting, got %q", got.Status)
	}

	// Archived sessions excluded from ListByPipeline
	list, _ := svc.ListSessionsByPipeline(ctx, "pipe-1")
	if len(list) != 0 {
		t.Errorf("expected 0 active sessions, got %d", len(list))
	}

	// Listed in archived
	archived, _ := svc.ListArchivedByPipeline(ctx, "pipe-1")
	if len(archived) != 1 {
		t.Errorf("expected 1 archived session, got %d", len(archived))
	}

	// Unarchive
	if err := svc.UnarchiveSession(ctx, s.ID); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	got, _ = svc.GetSession(ctx, s.ID)
	if got.ArchivedAt != nil {
		t.Error("expected ArchivedAt to be nil after unarchive")
	}
	list, _ = svc.ListSessionsByPipeline(ctx, "pipe-1")
	if len(list) != 1 {
		t.Errorf("expected 1 active session after unarchive, got %d", len(list))
	}
}

func TestContentSessionService_DeleteRequiresArchived(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)

	// Delete without archiving should fail
	err := svc.DeleteSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when deleting non-archived session")
	}

	// Archive then delete should succeed
	svc.ArchiveSession(ctx, s.ID)
	if err := svc.DeleteSession(ctx, s.ID); err != nil {
		t.Fatalf("delete archived session: %v", err)
	}

	// Session should no longer exist
	_, err = svc.GetSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when getting deleted session")
	}
}

func TestContentSessionService_ArchiveAlreadyArchived(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)
	svc.ArchiveSession(ctx, s.ID)

	err := svc.ArchiveSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when archiving already archived session")
	}
}

func TestContentSessionService_UnarchiveNotArchived(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)

	err := svc.UnarchiveSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when unarchiving non-archived session")
	}
}

func TestContentSessionService_TemplateInstanceSeparation(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	// Create a template session
	tmpl := &upal.ContentSession{
		PipelineID: "pipe-1", TriggerType: "manual",
		IsTemplate: true, Status: upal.SessionDraft,
	}
	if err := svc.CreateSession(ctx, tmpl); err != nil {
		t.Fatalf("create template: %v", err)
	}

	// Create an instance session (like collectSession would)
	inst := &upal.ContentSession{
		PipelineID: "pipe-1", TriggerType: "manual",
		ParentSessionID: tmpl.ID,
	}
	if err := svc.CreateSession(ctx, inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// ListSessionsByPipeline (instances) excludes templates
	instances, err := svc.ListSessionsByPipeline(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("expected 1 instance, got %d", len(instances))
	}
	if instances[0].ID == tmpl.ID {
		t.Error("ListSessionsByPipeline should not return templates")
	}

	// ListTemplatesByPipeline returns only templates
	templates, err := svc.ListTemplatesByPipeline(ctx, "pipe-1")
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(templates) != 1 {
		t.Errorf("expected 1 template, got %d", len(templates))
	}
	if templates[0].ID != tmpl.ID {
		t.Errorf("expected template ID %q, got %q", tmpl.ID, templates[0].ID)
	}

	// Instance should have ParentSessionID set
	got, _ := svc.GetSession(ctx, inst.ID)
	if got.ParentSessionID != tmpl.ID {
		t.Errorf("expected ParentSessionID %q, got %q", tmpl.ID, got.ParentSessionID)
	}
	if got.IsTemplate {
		t.Error("instance should not be a template")
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
