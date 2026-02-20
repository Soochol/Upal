package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// newTestServer creates a Server with a MemoryRepository and WorkflowService for tests.
func newTestServer() *Server {
	sessionSvc := session.InMemoryService()
	repo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(repo, nil, sessionSvc, nil)
	return NewServer(nil, wfSvc, repo, nil)
}

func TestRunWorkflow_NotFound(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("POST", "/api/workflows/nonexistent/run", strings.NewReader(`{"inputs":{}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRunWorkflow_SSE(t *testing.T) {
	srv := newTestServer()

	// Create a workflow with input and output nodes.
	wf := upal.WorkflowDefinition{
		Name:    "test-wf",
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "input1", To: "output1"},
		},
	}
	body, _ := json.Marshal(wf)
	createReq := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create workflow: got %d, want 201", createW.Code)
	}

	// Run the workflow.
	runBody := `{"inputs":{"input1":"test-value"}}`
	runReq := httptest.NewRequest("POST", "/api/workflows/test-wf/run", strings.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run workflow: got %d, want 200, body: %s", runW.Code, runW.Body.String())
	}

	// Verify SSE content type.
	contentType := runW.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "text/event-stream")
	}

	// Verify we got a done event.
	responseBody := runW.Body.String()
	if !strings.Contains(responseBody, "event: done") {
		t.Errorf("expected done event in response, got: %s", responseBody)
	}
	if !strings.Contains(responseBody, `"completed"`) {
		t.Errorf("expected completed status in response, got: %s", responseBody)
	}
}

func TestRunWorkflow_EmptyBody(t *testing.T) {
	srv := newTestServer()

	// Create a workflow with just an output node (no inputs needed).
	wf := upal.WorkflowDefinition{
		Name:    "simple-wf",
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "out1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
	}
	body, _ := json.Marshal(wf)
	createReq := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createW, createReq)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create workflow: got %d, want 201", createW.Code)
	}

	// Run with empty body.
	runReq := httptest.NewRequest("POST", "/api/workflows/simple-wf/run", nil)
	runW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run workflow: got %d, want 200, body: %s", runW.Code, runW.Body.String())
	}

	// Verify it has a done event.
	responseBody := runW.Body.String()
	if !strings.Contains(responseBody, "event: done") {
		t.Error("expected done event in response")
	}
}
