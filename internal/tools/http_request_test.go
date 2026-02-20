package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRequestTool_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("X-Test", "hello")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := &HTTPRequestTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"method": "GET",
		"url":    srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["status_code"] != 200 {
		t.Errorf("expected 200, got %v", m["status_code"])
	}
	if m["body"] != `{"ok":true}` {
		t.Errorf("unexpected body: %v", m["body"])
	}
	hdrs := m["headers"].(map[string]string)
	if hdrs["X-Test"] != "hello" {
		t.Errorf("expected X-Test=hello, got %v", hdrs["X-Test"])
	}
}

func TestHTTPRequestTool_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(201)
		w.Write([]byte("created"))
	}))
	defer srv.Close()

	tool := &HTTPRequestTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"method":  "POST",
		"url":     srv.URL,
		"headers": map[string]any{"Content-Type": "application/json"},
		"body":    `{"name":"test"}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["status_code"] != 201 {
		t.Errorf("expected 201, got %v", m["status_code"])
	}
}

func TestHTTPRequestTool_InvalidMethod(t *testing.T) {
	tool := &HTTPRequestTool{}
	_, err := tool.Execute(context.Background(), map[string]any{
		"method": "INVALID",
		"url":    "http://example.com",
	})
	if err == nil {
		t.Error("expected error for invalid method")
	}
}

func TestHTTPRequestTool_MissingURL(t *testing.T) {
	tool := &HTTPRequestTool{}
	_, err := tool.Execute(context.Background(), map[string]any{
		"method": "GET",
	})
	if err == nil {
		t.Error("expected error for missing URL")
	}
}
