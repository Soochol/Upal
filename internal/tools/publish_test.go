// internal/tools/publish_test.go
package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishTool_MarkdownFile(t *testing.T) {
	dir := t.TempDir()
	tool := NewPublishTool(dir)

	result, err := tool.Execute(context.Background(), map[string]any{
		"channel":  "markdown_file",
		"title":    "Test Article",
		"content":  "# Hello World\n\nThis is a test.",
		"metadata": map[string]any{"tags": "test,demo"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := result.(map[string]any)
	if res["status"] != "published" {
		t.Errorf("expected status 'published', got %v", res["status"])
	}

	path := res["path"].(string)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "# Hello World") {
		t.Error("output file should contain the content")
	}
}

func TestPublishTool_Webhook(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tool := NewPublishTool(t.TempDir())
	result, err := tool.Execute(context.Background(), map[string]any{
		"channel": "webhook",
		"content": "Hello from pipeline",
		"title":   "Test",
		"metadata": map[string]any{
			"webhook_url": srv.URL,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := result.(map[string]any)
	if res["status"] != "published" {
		t.Errorf("expected status 'published', got %v", res["status"])
	}
	if received["content"] != "Hello from pipeline" {
		t.Errorf("webhook should receive content, got %v", received["content"])
	}
}

func TestPublishTool_UnknownChannel(t *testing.T) {
	tool := NewPublishTool(t.TempDir())
	_, err := tool.Execute(context.Background(), map[string]any{
		"channel": "fax_machine",
		"content": "test",
	})
	if err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

// Verify the output directory is created correctly
func TestPublishTool_OutputDirCreation(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "a", "b", "c")
	tool := NewPublishTool(nestedDir)

	_, err := tool.Execute(context.Background(), map[string]any{
		"channel": "markdown_file",
		"title":   "Nested",
		"content": "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("nested output directory should be created")
	}
}
