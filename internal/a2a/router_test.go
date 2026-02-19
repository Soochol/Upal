package a2a

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/a2atypes"
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

	params := a2atypes.SendMessageParams{
		Message: a2atypes.Message{
			Role:  "user",
			Parts: []a2atypes.Part{a2atypes.TextPart("hello via router")},
		},
	}
	body := makeJSONRPCBody("a2a.sendMessage", params)

	req := httptest.NewRequest(http.MethodPost, "/a2a/nodes/agent1", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp a2atypes.JSONRPCResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp.Error)
	}

	taskData, _ := json.Marshal(resp.Result)
	var task a2atypes.Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		t.Fatalf("failed to decode task: %v", err)
	}
	if task.Status != a2atypes.TaskCompleted {
		t.Errorf("expected status %q, got %q", a2atypes.TaskCompleted, task.Status)
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

	var card a2atypes.AgentCard
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

	var card a2atypes.AgentCard
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

