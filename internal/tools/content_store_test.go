// internal/tools/content_store_test.go
package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestContentStoreTool_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	tool := NewContentStoreTool(filepath.Join(dir, "store.json"))

	// Set a value
	_, err := tool.Execute(context.Background(), map[string]any{
		"action": "set",
		"key":    "seen:https://example.com/1",
		"value":  "2026-02-21T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Get the value
	result, err := tool.Execute(context.Background(), map[string]any{
		"action": "get",
		"key":    "seen:https://example.com/1",
	})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	res := result.(map[string]any)
	if res["value"] != "2026-02-21T09:00:00Z" {
		t.Errorf("expected stored value, got %v", res["value"])
	}
}

func TestContentStoreTool_ListByPrefix(t *testing.T) {
	dir := t.TempDir()
	tool := NewContentStoreTool(filepath.Join(dir, "store.json"))

	ctx := context.Background()
	tool.Execute(ctx, map[string]any{"action": "set", "key": "seen:url1", "value": "1"})
	tool.Execute(ctx, map[string]any{"action": "set", "key": "seen:url2", "value": "2"})
	tool.Execute(ctx, map[string]any{"action": "set", "key": "other:key", "value": "3"})

	result, err := tool.Execute(ctx, map[string]any{
		"action": "list",
		"prefix": "seen:",
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	res := result.(map[string]any)
	keys := res["keys"].([]string)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys with prefix 'seen:', got %d", len(keys))
	}
}

func TestContentStoreTool_Delete(t *testing.T) {
	dir := t.TempDir()
	tool := NewContentStoreTool(filepath.Join(dir, "store.json"))

	ctx := context.Background()
	tool.Execute(ctx, map[string]any{"action": "set", "key": "k1", "value": "v1"})
	tool.Execute(ctx, map[string]any{"action": "delete", "key": "k1"})

	result, err := tool.Execute(ctx, map[string]any{"action": "get", "key": "k1"})
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	res := result.(map[string]any)
	if res["value"] != nil {
		t.Errorf("expected nil after delete, got %v", res["value"])
	}
}

func TestContentStoreTool_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	// Write with first instance
	tool1 := NewContentStoreTool(path)
	tool1.Execute(context.Background(), map[string]any{
		"action": "set", "key": "persist", "value": "yes",
	})

	// Read with new instance (same file)
	tool2 := NewContentStoreTool(path)
	result, _ := tool2.Execute(context.Background(), map[string]any{
		"action": "get", "key": "persist",
	})
	res := result.(map[string]any)
	if res["value"] != "yes" {
		t.Errorf("expected persisted value 'yes', got %v", res["value"])
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("store file should exist on disk")
	}
}
