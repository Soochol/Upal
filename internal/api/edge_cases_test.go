package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// --- Helpers ---

// newEdgeCaseServer creates a server with a trigger repo suitable for webhook edge-case tests.
func newEdgeCaseServer() (*Server, repository.TriggerRepository) {
	sessionSvc := session.InMemoryService()
	wfRepo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(wfRepo, nil, sessionSvc, nil, agents.DefaultRegistry())
	trigRepo := repository.NewMemoryTriggerRepository()
	srv := NewServer(nil, wfSvc, wfRepo, nil)
	srv.SetTriggerRepository(trigRepo)
	return srv, trigRepo
}

// seedEdgeCaseWorkflow creates a minimal workflow so the webhook handler's Lookup succeeds.
func seedEdgeCaseWorkflow(t *testing.T, srv *Server, name string) {
	t.Helper()
	wf := upal.WorkflowDefinition{
		Name:    name,
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "out1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
	}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("seedEdgeCaseWorkflow: got status %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

// ============================================================
// Webhook Edge Cases
// ============================================================

// TestWebhook_MalformedJSON verifies that sending invalid JSON to the webhook
// handler does not cause a panic. The handler silently ignores the unmarshal
// failure and proceeds with a nil payload, ultimately returning 202.
func TestWebhook_MalformedJSON(t *testing.T) {
	srv, trigRepo := newEdgeCaseServer()
	seedEdgeCaseWorkflow(t, srv, "malformed-wf")

	trigger := &upal.Trigger{
		ID:           "trig_malformed",
		WorkflowName: "malformed-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""}, // no secret — skip HMAC
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	// Send body that is not valid JSON.
	malformed := []byte(`{not valid json}`)
	req := httptest.NewRequest("POST", "/api/hooks/trig_malformed", bytes.NewReader(malformed))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// The primary assertion: no panic occurs.
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}
}

// TestWebhook_LargePayload verifies that the webhook handler can process a
// large (~100KB) JSON payload without error.
func TestWebhook_LargePayload(t *testing.T) {
	srv, trigRepo := newEdgeCaseServer()
	seedEdgeCaseWorkflow(t, srv, "large-wf")

	trigger := &upal.Trigger{
		ID:           "trig_large",
		WorkflowName: "large-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	// Build a ~100KB JSON payload.
	largeData := strings.Repeat("x", 100_000)
	payload, _ := json.Marshal(map[string]string{"data": largeData})

	req := httptest.NewRequest("POST", "/api/hooks/trig_large", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
	}
}

// TestWebhook_EmptyTriggerID verifies that POSTing to /api/hooks/ with no
// trigger ID returns a 404 or 405 (Chi won't route to the handler).
func TestWebhook_EmptyTriggerID(t *testing.T) {
	srv, _ := newEdgeCaseServer()

	req := httptest.NewRequest("POST", "/api/hooks/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	// Chi routing: /api/hooks/{id} won't match an empty id, so we expect
	// either 404 (no route matched) or 405 (method not allowed).
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d, want 404 or 405", w.Code)
	}
}

// TestWebhook_ContentTypeVariations verifies that the webhook handler works
// regardless of Content-Type header since it reads raw body bytes.
func TestWebhook_ContentTypeVariations(t *testing.T) {
	srv, trigRepo := newEdgeCaseServer()
	seedEdgeCaseWorkflow(t, srv, "ctype-wf")

	trigger := &upal.Trigger{
		ID:           "trig_ctype",
		WorkflowName: "ctype-wf",
		Type:         upal.TriggerWebhook,
		Config:       upal.TriggerConfig{Secret: ""},
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
	if err := trigRepo.Create(context.Background(), trigger); err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	payload := []byte(`{"msg":"hello"}`)

	tests := []struct {
		name        string
		contentType string
	}{
		{"text/plain", "text/plain"},
		{"no content-type", ""},
		{"application/octet-stream", "application/octet-stream"},
		{"application/xml", "application/xml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/hooks/trig_ctype", bytes.NewReader(payload))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)

			if w.Code != http.StatusAccepted {
				t.Fatalf("status: got %d, want 202; body: %s", w.Code, w.Body.String())
			}
		})
	}
}

// ============================================================
// Input Mapping Edge Cases
// ============================================================

// TestMapInputs_MissingKeys verifies that when the mapping references keys not
// present in the payload, those inputs are simply absent from the result (no error).
func TestMapInputs_MissingKeys(t *testing.T) {
	payload := map[string]any{
		"name": "alice",
	}
	mapping := map[string]string{
		"user":  "name",            // exists in payload
		"email": "email_address",   // does NOT exist in payload
		"age":   "user_age",        // does NOT exist in payload
	}

	result := mapInputs(payload, mapping)

	// "user" should be mapped.
	if result["user"] != "alice" {
		t.Errorf("expected user=alice, got %v", result["user"])
	}

	// "email" and "age" should be absent (not error, not nil-valued).
	if _, ok := result["email"]; ok {
		t.Error("expected 'email' key to be absent from result, but it was present")
	}
	if _, ok := result["age"]; ok {
		t.Error("expected 'age' key to be absent from result, but it was present")
	}

	// Result should have exactly 1 key.
	if len(result) != 1 {
		t.Errorf("expected 1 key in result, got %d: %v", len(result), result)
	}
}

// TestMapInputs_EmptyPayload verifies that a nil payload with a non-empty
// mapping returns an empty (non-nil) map.
func TestMapInputs_EmptyPayload(t *testing.T) {
	mapping := map[string]string{
		"query": "text",
		"user":  "sender",
	}

	result := mapInputs(nil, mapping)

	if result == nil {
		t.Fatal("expected non-nil empty map, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys: %v", len(result), result)
	}
}

// TestMapInputs_EmptyMapping verifies that an empty (non-nil) mapping is
// treated the same as a nil mapping: the full payload is returned as-is.
func TestMapInputs_EmptyMapping(t *testing.T) {
	payload := map[string]any{
		"message": "hello",
		"count":   42,
	}

	// Empty map (not nil).
	result := mapInputs(payload, map[string]string{})

	// Should return the full payload.
	if result["message"] != "hello" {
		t.Errorf("expected message=hello, got %v", result["message"])
	}
	if result["count"] != 42 {
		t.Errorf("expected count=42, got %v", result["count"])
	}
	if len(result) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(result), result)
	}
}
