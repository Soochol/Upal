package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocalStorage_SaveAndGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	content := "hello world"
	info, err := store.Save(ctx, "test.txt", "text/plain", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if info.Filename != "test.txt" {
		t.Errorf("filename: got %q", info.Filename)
	}
	if info.ContentType != "text/plain" {
		t.Errorf("content_type: got %q", info.ContentType)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("size: got %d, want %d", info.Size, len(content))
	}
	if info.ID == "" {
		t.Error("ID should not be empty")
	}

	// Get
	gotInfo, reader, err := store.Get(ctx, info.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer reader.Close()

	if gotInfo.Filename != "test.txt" {
		t.Errorf("get filename: got %q", gotInfo.Filename)
	}

	buf := make([]byte, 1024)
	n, _ := reader.Read(buf)
	if string(buf[:n]) != content {
		t.Errorf("content: got %q", string(buf[:n]))
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	info, err := store.Save(ctx, "to-delete.txt", "text/plain", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, info.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should not be found after delete
	_, _, err = store.Get(ctx, info.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestLocalStorage_DeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	err = store.Delete(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent file")
	}
}

func TestLocalStorage_List(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()

	// Empty list
	files, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("list empty: got %d", len(files))
	}

	// Add two files
	store.Save(ctx, "a.txt", "text/plain", strings.NewReader("aaa"))
	store.Save(ctx, "b.txt", "text/plain", strings.NewReader("bbb"))

	files, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("list: got %d, want 2", len(files))
	}
}

func TestLocalStorage_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	_, _, err = store.Get(context.Background(), "no-such-id")
	if err == nil {
		t.Fatal("expected error for get not found")
	}
}

func TestLocalStorage_SavePreservesExtension(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	info, err := store.Save(context.Background(), "image.png", "image/png", strings.NewReader("png data"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(info.Path, ".png") {
		t.Errorf("path should preserve .png extension: got %q", info.Path)
	}
}
