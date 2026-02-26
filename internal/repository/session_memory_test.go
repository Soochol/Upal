package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestMemorySessionRepository_CRUD(t *testing.T) {
	repo := NewMemorySessionRepository()
	ctx := context.Background()

	s := &upal.Session{
		ID:        "sess-1",
		Name:      "Test Session",
		Status:    upal.SessionStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, "sess-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "Test Session" {
		t.Errorf("expected name 'Test Session', got %q", got.Name)
	}
	if got.Status != upal.SessionStatusDraft {
		t.Errorf("expected status draft, got %q", got.Status)
	}

	// List
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}

	// Update
	s.Name = "Updated Session"
	s.Status = upal.SessionStatusActive
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = repo.Get(ctx, "sess-1")
	if got.Name != "Updated Session" {
		t.Errorf("expected updated name, got %q", got.Name)
	}
	if got.Status != upal.SessionStatusActive {
		t.Errorf("expected active status, got %q", got.Status)
	}

	// Delete
	if err := repo.Delete(ctx, "sess-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.Get(ctx, "sess-1")
	if err == nil {
		t.Error("expected error after Delete, got nil")
	}
}

func TestMemorySessionRepository_GetNotFound(t *testing.T) {
	repo := NewMemorySessionRepository()
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing session, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemorySessionRepository_DuplicateCreate(t *testing.T) {
	repo := NewMemorySessionRepository()
	ctx := context.Background()

	s := &upal.Session{ID: "sess-dup", Name: "Dup", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := repo.Create(ctx, s); err == nil {
		t.Error("expected error on duplicate Create, got nil")
	}
}

func TestMemorySessionRepository_UpdateNotFound(t *testing.T) {
	repo := NewMemorySessionRepository()
	ctx := context.Background()

	s := &upal.Session{ID: "nonexistent", Name: "Ghost"}
	err := repo.Update(ctx, s)
	if err == nil {
		t.Fatal("expected error for Update on missing session, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemorySessionRepository_DeleteNotFound(t *testing.T) {
	repo := NewMemorySessionRepository()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for Delete on missing session, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
