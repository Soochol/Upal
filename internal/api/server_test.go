package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestAPI_CreateWorkflow(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager(), nil, nil)
	wf := engine.WorkflowDefinition{Name: "test-wf", Version: 1, Nodes: []engine.NodeDefinition{{ID: "input1", Type: engine.NodeTypeInput}}}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "test-wf" {
		t.Errorf("name: got %v", resp["name"])
	}
}

func TestAPI_ListWorkflows(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager(), nil, nil)
	wf := engine.WorkflowDefinition{Name: "wf1", Version: 1}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	req = httptest.NewRequest("GET", "/api/workflows", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp) != 1 {
		t.Errorf("workflows: got %d, want 1", len(resp))
	}
}

func TestAPI_AggregateAgentCard(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager(), nil, nil)

	req := httptest.NewRequest("GET", "/a2a/agent-card", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["name"] != "upal" {
		t.Errorf("name: got %v, want upal", resp["name"])
	}
}

func TestAPI_GetWorkflow(t *testing.T) {
	srv := NewServer(engine.NewEventBus(), engine.NewSessionManager(), nil, nil)
	wf := engine.WorkflowDefinition{Name: "wf1", Version: 1}
	body, _ := json.Marshal(wf)
	req := httptest.NewRequest("POST", "/api/workflows", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	req = httptest.NewRequest("GET", "/api/workflows/wf1", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
}
