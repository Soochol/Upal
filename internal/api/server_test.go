package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestAPI_CreateWorkflow(t *testing.T) {
	srv := newTestServer()
	wf := upal.WorkflowDefinition{Name: "test-wf", Version: 1, Nodes: []upal.NodeDefinition{{ID: "input1", Type: upal.NodeTypeInput}}}
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
	srv := newTestServer()
	wf := upal.WorkflowDefinition{Name: "wf1", Version: 1}
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

func TestAPI_GetWorkflow(t *testing.T) {
	srv := newTestServer()
	wf := upal.WorkflowDefinition{Name: "wf1", Version: 1}
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
