package repository

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestMemoryConnectionRepository_CRUD(t *testing.T) {
	repo := NewMemoryConnectionRepository()
	ctx := context.Background()

	conn := &upal.Connection{
		ID:    "conn-1",
		Name:  "Test Telegram",
		Type:  upal.ConnTypeTelegram,
		Token: "bot-token-123",
	}

	// Create.
	if err := repo.Create(ctx, conn); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Duplicate create fails.
	if err := repo.Create(ctx, conn); err == nil {
		t.Fatal("duplicate create should fail")
	}

	// Get.
	got, err := repo.Get(ctx, "conn-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Token != "bot-token-123" {
		t.Fatalf("expected token 'bot-token-123', got %q", got.Token)
	}

	// List.
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(list))
	}

	// Update.
	conn.Token = "updated-token"
	if err := repo.Update(ctx, conn); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = repo.Get(ctx, "conn-1")
	if got.Token != "updated-token" {
		t.Fatalf("expected updated token, got %q", got.Token)
	}

	// Update nonexistent.
	if err := repo.Update(ctx, &upal.Connection{ID: "nope"}); err == nil {
		t.Fatal("update nonexistent should fail")
	}

	// Delete.
	if err := repo.Delete(ctx, "conn-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, "conn-1"); err == nil {
		t.Fatal("get after delete should fail")
	}

	// Delete nonexistent.
	if err := repo.Delete(ctx, "conn-1"); err == nil {
		t.Fatal("delete nonexistent should fail")
	}
}
