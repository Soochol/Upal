package repository

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestMemoryContentSessionRepo_CRUD(t *testing.T) {
	repo := NewMemoryContentSessionRepository()
	ctx := context.Background()

	s := &upal.ContentSession{
		ID:          "csess-1",
		PipelineID:  "pipe-1",
		Status:      upal.SessionCollecting,
		TriggerType: "manual",
		CreatedAt:   time.Now(),
	}

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, "csess-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != upal.SessionCollecting {
		t.Errorf("expected status collecting, got %q", got.Status)
	}
	list, _ := repo.List(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	byPipeline, _ := repo.ListByPipeline(ctx, "pipe-1")
	if len(byPipeline) != 1 {
		t.Fatalf("expected 1 by pipeline, got %d", len(byPipeline))
	}
	s.Status = upal.SessionPendingReview
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = repo.Get(ctx, "csess-1")
	if got.Status != upal.SessionPendingReview {
		t.Errorf("expected pending_review, got %q", got.Status)
	}
}

func TestMemorySourceFetchRepo(t *testing.T) {
	repo := NewMemorySourceFetchRepository()
	ctx := context.Background()

	sf := &upal.SourceFetch{
		ID:        "sf-1",
		SessionID: "csess-1",
		ToolName:  "hn_fetch",
		FetchedAt: time.Now(),
		RawItems:  []upal.SourceItem{{Title: "Test Article", Score: 100}},
	}
	if err := repo.Create(ctx, sf); err != nil {
		t.Fatalf("create: %v", err)
	}
	list, _ := repo.ListBySession(ctx, "csess-1")
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].ToolName != "hn_fetch" {
		t.Errorf("unexpected tool name: %q", list[0].ToolName)
	}
}

func TestMemorySurgeEventRepo(t *testing.T) {
	repo := NewMemorySurgeEventRepository()
	ctx := context.Background()

	se := &upal.SurgeEvent{
		ID:         "surge-1",
		Keyword:    "DeepSeek",
		Multiplier: 10.0,
		CreatedAt:  time.Now(),
	}
	if err := repo.Create(ctx, se); err != nil {
		t.Fatalf("create: %v", err)
	}
	list, _ := repo.ListActive(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 active, got %d", len(list))
	}
	se.Dismissed = true
	repo.Update(ctx, se)
	active, _ := repo.ListActive(ctx)
	if len(active) != 0 {
		t.Fatalf("expected 0 active after dismiss, got %d", len(active))
	}
}
