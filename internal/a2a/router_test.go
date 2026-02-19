package a2a

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/engine"
)

func TestRouter_NodeEndpoint(t *testing.T) {
	r := chi.NewRouter()
	nodeDef := &engine.NodeDefinition{
		ID:     "agent1",
		Type:   engine.NodeTypeAgent,
		Config: map[string]any{},
	}
	exec := &mockExecutor{result: "routed response"}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeAgent: exec,
	}

	MountA2ARoutes(r, []*engine.NodeDefinition{nodeDef}, executors, "http://localhost:8080")

	params := SendMessageParams{
		Message: Message{
			Role:  "user",
			Parts: []Part{TextPart("hello via router")},
		},
	}
	body := makeJSONRPCBody("a2a.sendMessage", params)

	req := httptest.NewRequest(http.MethodPost, "/a2a/nodes/agent1", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp.Error)
	}

	taskData, _ := json.Marshal(resp.Result)
	var task Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		t.Fatalf("failed to decode task: %v", err)
	}
	if task.Status != TaskCompleted {
		t.Errorf("expected status %q, got %q", TaskCompleted, task.Status)
	}
	if len(task.Artifacts) != 1 || task.Artifacts[0].FirstText() != "routed response" {
		t.Errorf("expected artifact text %q, got %v", "routed response", task.Artifacts)
	}
}

func TestRouter_AgentCardEndpoint(t *testing.T) {
	r := chi.NewRouter()
	nodeDef := &engine.NodeDefinition{
		ID:     "agent1",
		Type:   engine.NodeTypeAgent,
		Config: map[string]any{"system_prompt": "I summarize things."},
	}
	exec := &mockExecutor{result: "unused"}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeAgent: exec,
	}

	MountA2ARoutes(r, []*engine.NodeDefinition{nodeDef}, executors, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/a2a/nodes/agent1/agent-card", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var card AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("failed to decode agent card: %v", err)
	}
	if card.Name != "agent1" {
		t.Errorf("expected name %q, got %q", "agent1", card.Name)
	}
	if card.URL != "http://localhost:8080/a2a/nodes/agent1" {
		t.Errorf("expected URL %q, got %q", "http://localhost:8080/a2a/nodes/agent1", card.URL)
	}
	if card.Description != "I summarize things." {
		t.Errorf("expected description from system_prompt, got %q", card.Description)
	}
}

func TestRouter_AggregateAgentCard(t *testing.T) {
	r := chi.NewRouter()
	nodes := []*engine.NodeDefinition{
		{ID: "node1", Type: engine.NodeTypeAgent, Config: map[string]any{}},
		{ID: "node2", Type: engine.NodeTypeAgent, Config: map[string]any{}},
	}
	exec := &mockExecutor{result: "unused"}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeAgent: exec,
	}

	MountA2ARoutes(r, nodes, executors, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/a2a/agent-card", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var card AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("failed to decode aggregate card: %v", err)
	}
	if card.Name != "upal" {
		t.Errorf("expected name %q, got %q", "upal", card.Name)
	}
	if len(card.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(card.Skills))
	}
	if card.URL != "http://localhost:8080/a2a" {
		t.Errorf("expected URL %q, got %q", "http://localhost:8080/a2a", card.URL)
	}
}

func TestRouter_StaticAgentCard(t *testing.T) {
	r := chi.NewRouter()
	executors := map[engine.NodeType]engine.NodeExecutorInterface{}

	MountStaticA2ARoutes(r, executors, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/a2a/agent-card", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var card AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("failed to decode static card: %v", err)
	}
	if card.Name != "upal" {
		t.Errorf("expected name %q, got %q", "upal", card.Name)
	}
	if card.Description != "Upal visual AI workflow platform" {
		t.Errorf("expected description %q, got %q", "Upal visual AI workflow platform", card.Description)
	}
	if len(card.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(card.Skills))
	}
	if card.URL != "http://localhost:8080/a2a" {
		t.Errorf("expected URL %q, got %q", "http://localhost:8080/a2a", card.URL)
	}
}

func TestRouter_StaticNodeNotRegistered(t *testing.T) {
	r := chi.NewRouter()
	executors := map[engine.NodeType]engine.NodeExecutorInterface{}

	MountStaticA2ARoutes(r, executors, "http://localhost:8080")

	body := makeJSONRPCBody("a2a.sendMessage", SendMessageParams{
		Message: Message{
			Role:  "user",
			Parts: []Part{TextPart("hello")},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/a2a/nodes/foo", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (JSON-RPC error in body), got %d", rec.Code)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Node not registered. Start a workflow run first." {
		t.Errorf("expected error message about node not registered, got %q", resp.Error.Message)
	}
}
