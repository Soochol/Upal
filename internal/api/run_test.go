package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	runpub "github.com/soochol/upal/internal/services/run"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// newTestServer creates a Server with a MemoryRepository and WorkflowService for tests.
func newTestServer() *Server {
	sessionSvc := session.InMemoryService()
	repo := repository.NewMemory()
	wfSvc := services.NewWorkflowService(repo, nil, sessionSvc, nil, agents.DefaultRegistry(), "")
	srv := NewServer(nil, wfSvc, repo, nil)

	runRepo := repository.NewMemoryRunRepository()
	runHistorySvc := services.NewRunHistoryService(runRepo)
	srv.SetRunHistoryService(runHistorySvc)

	rm := services.NewRunManager(5 * time.Minute)
	srv.SetRunManager(rm)

	pub := runpub.NewRunPublisher(wfSvc, rm, runHistorySvc)
	srv.SetRunPublisher(pub)

	return srv
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

func TestRunWorkflow_Returns202(t *testing.T) {
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

	if runW.Code != http.StatusAccepted {
		t.Fatalf("run workflow: got %d, want 202, body: %s", runW.Code, runW.Body.String())
	}

	// Verify JSON response with run_id.
	contentType := runW.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", contentType)
	}

	var result map[string]string
	if err := json.Unmarshal(runW.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if result["run_id"] == "" {
		t.Error("expected non-empty run_id in response")
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

	if runW.Code != http.StatusAccepted {
		t.Fatalf("run workflow: got %d, want 202, body: %s", runW.Code, runW.Body.String())
	}

	var result map[string]string
	if err := json.Unmarshal(runW.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	// run_id should be present (may be non-empty with history service).
	if _, ok := result["run_id"]; !ok {
		t.Error("expected run_id key in response")
	}
}

func TestStreamRunEvents_NotFound(t *testing.T) {
	srv := newTestServer()

	req := httptest.NewRequest("GET", "/api/runs/nonexistent/events", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body: %s", w.Code, w.Body.String())
	}
}
