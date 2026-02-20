package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetWebpageTool_BasicHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Test Page</title></head><body><h1>Hello</h1><p>World</p></body></html>`))
	}))
	defer srv.Close()

	tool := &GetWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["title"] != "Test Page" {
		t.Errorf("expected title 'Test Page', got %v", m["title"])
	}
	text := m["text"].(string)
	if !strings.Contains(text, "Hello") || !strings.Contains(text, "World") {
		t.Errorf("expected text to contain 'Hello' and 'World', got %q", text)
	}
}

func TestGetWebpageTool_StripsScriptAndStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body><script>alert('x')</script><style>body{}</style><p>Content</p></body></html>`))
	}))
	defer srv.Close()

	tool := &GetWebpageTool{}
	result, err := tool.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	text := m["text"].(string)
	if strings.Contains(text, "alert") || strings.Contains(text, "body{}") {
		t.Errorf("expected script/style content to be stripped, got %q", text)
	}
	if !strings.Contains(text, "Content") {
		t.Errorf("expected 'Content' in text, got %q", text)
	}
}

func TestGetWebpageTool_MissingURL(t *testing.T) {
	tool := &GetWebpageTool{}
	_, err := tool.Execute(context.Background(), map[string]any{})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestGetWebpageTool_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	tool := &GetWebpageTool{}
	_, err := tool.Execute(context.Background(), map[string]any{"url": srv.URL})
	if err == nil {
		t.Error("expected error for 404 response")
	}
}
