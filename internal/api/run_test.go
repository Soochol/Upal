package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

// mockExecutor is a simple executor that returns a fixed string.
type mockExecutor struct {
	result string
}

func (m *mockExecutor) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	return m.result, nil
}

func TestRunWorkflow_NotFound(t *testing.T) {
	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()
	runner := engine.NewRunner(eventBus, sessions)
	executors := map[engine.NodeType]engine.NodeExecutorInterface{}

	srv := NewServer(eventBus, sessions, runner, nil, executors)

	req := httptest.NewRequest("POST", "/api/workflows/nonexistent/run", strings.NewReader(`{"inputs":{}}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRunWorkflow_SSE(t *testing.T) {
	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()
	runner := engine.NewRunner(eventBus, sessions)

	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:  &mockExecutor{result: "hello"},
		engine.NodeTypeOutput: &mockExecutor{result: "world"},
	}

	srv := NewServer(eventBus, sessions, runner, nil, executors)

	// Create a workflow first
	wf := engine.WorkflowDefinition{
		Name:    "test-wf",
		Version: 1,
		Nodes: []engine.NodeDefinition{
			{ID: "input1", Type: engine.NodeTypeInput},
			{ID: "output1", Type: engine.NodeTypeOutput},
		},
		Edges: []engine.EdgeDefinition{
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

	// Run the workflow
	runBody := `{"inputs":{"input1":"test-value"}}`
	runReq := httptest.NewRequest("POST", "/api/workflows/test-wf/run", strings.NewReader(runBody))
	runReq.Header.Set("Content-Type", "application/json")
	runW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run workflow: got %d, want 200", runW.Code)
	}

	// Verify SSE content type
	contentType := runW.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "text/event-stream")
	}

	// Parse SSE events from the response body
	responseBody := runW.Body.String()
	scanner := bufio.NewScanner(strings.NewReader(responseBody))

	var events []struct {
		eventType string
		data      string
	}

	var currentEvent string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			events = append(events, struct {
				eventType string
				data      string
			}{eventType: currentEvent, data: data})
		}
	}

	if len(events) == 0 {
		t.Fatal("expected at least one SSE event, got none")
	}

	// Check that the last event is "done"
	lastEvent := events[len(events)-1]
	if lastEvent.eventType != "done" {
		t.Errorf("last event type: got %q, want %q", lastEvent.eventType, "done")
	}

	// Parse the done event data
	var doneData map[string]any
	if err := json.Unmarshal([]byte(lastEvent.data), &doneData); err != nil {
		t.Fatalf("parse done data: %v", err)
	}
	if doneData["status"] != "completed" {
		t.Errorf("done status: got %v, want %q", doneData["status"], "completed")
	}
	if doneData["session_id"] == nil || doneData["session_id"] == "" {
		t.Error("done event missing session_id")
	}

	// Verify we got some node events (node.started, node.completed)
	hasNodeStarted := false
	hasNodeCompleted := false
	for _, evt := range events {
		if evt.eventType == "node.started" {
			hasNodeStarted = true
		}
		if evt.eventType == "node.completed" {
			hasNodeCompleted = true
		}
	}
	if !hasNodeStarted {
		t.Error("expected at least one node.started event")
	}
	if !hasNodeCompleted {
		t.Error("expected at least one node.completed event")
	}
}

func TestRunWorkflow_EmptyBody(t *testing.T) {
	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()
	runner := engine.NewRunner(eventBus, sessions)

	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeOutput: &mockExecutor{result: "result"},
	}

	srv := NewServer(eventBus, sessions, runner, nil, executors)

	// Create a workflow with just an output node (no inputs needed)
	wf := engine.WorkflowDefinition{
		Name:    "simple-wf",
		Version: 1,
		Nodes: []engine.NodeDefinition{
			{ID: "out1", Type: engine.NodeTypeOutput},
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

	// Run with empty body
	runReq := httptest.NewRequest("POST", "/api/workflows/simple-wf/run", nil)
	runW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(runW, runReq)

	if runW.Code != http.StatusOK {
		t.Fatalf("run workflow: got %d, want 200", runW.Code)
	}

	// Verify it has a done event
	responseBody := runW.Body.String()
	if !strings.Contains(responseBody, "event: done") {
		t.Error("expected done event in response")
	}
}
