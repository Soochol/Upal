package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

// mockExecutor implements engine.NodeExecutorInterface for testing.
type mockExecutor struct {
	result any
	err    error
}

func (m *mockExecutor) Execute(_ context.Context, _ *engine.NodeDefinition, _ map[string]any) (any, error) {
	return m.result, m.err
}

func makeJSONRPCBody(method string, params any) []byte {
	body, _ := json.Marshal(a2atypes.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	return body
}

func TestNodeHandler_SendMessage(t *testing.T) {
	exec := &mockExecutor{result: "hello from agent"}
	def := &engine.NodeDefinition{ID: "greeter", Type: engine.NodeTypeAgent, Config: map[string]any{}}
	handler := NewNodeHandler(exec, def)

	params := a2atypes.SendMessageParams{
		Message: a2atypes.Message{
			Role:  "user",
			Parts: []a2atypes.Part{a2atypes.TextPart("hi there")},
		},
	}
	body := makeJSONRPCBody("a2a.sendMessage", params)

	req := httptest.NewRequest(http.MethodPost, "/a2a/nodes/greeter", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var resp a2atypes.JSONRPCResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Decode the Task from result.
	taskData, _ := json.Marshal(resp.Result)
	var task a2atypes.Task
	if err := json.Unmarshal(taskData, &task); err != nil {
		t.Fatalf("failed to decode task: %v", err)
	}

	if task.Status != a2atypes.TaskCompleted {
		t.Errorf("expected status %q, got %q", a2atypes.TaskCompleted, task.Status)
	}
	if len(task.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(task.Artifacts))
	}
	if task.Artifacts[0].FirstText() != "hello from agent" {
		t.Errorf("expected artifact text %q, got %q", "hello from agent", task.Artifacts[0].FirstText())
	}
	// Should have 2 messages: user + agent.
	if len(task.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(task.Messages))
	}
	if task.Messages[0].Role != "user" {
		t.Errorf("expected first message role %q, got %q", "user", task.Messages[0].Role)
	}
	if task.Messages[1].Role != "agent" {
		t.Errorf("expected second message role %q, got %q", "agent", task.Messages[1].Role)
	}
}

func TestNodeHandler_SendMessage_ExecutorError(t *testing.T) {
	exec := &mockExecutor{err: errors.New("boom")}
	def := &engine.NodeDefinition{ID: "fail-node", Type: engine.NodeTypeAgent, Config: map[string]any{}}
	handler := NewNodeHandler(exec, def)

	params := a2atypes.SendMessageParams{
		Message: a2atypes.Message{
			Role:  "user",
			Parts: []a2atypes.Part{a2atypes.TextPart("trigger error")},
		},
	}
	body := makeJSONRPCBody("a2a.sendMessage", params)

	req := httptest.NewRequest(http.MethodPost, "/a2a/nodes/fail-node", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

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

	if task.Status != a2atypes.TaskFailed {
		t.Errorf("expected status %q, got %q", a2atypes.TaskFailed, task.Status)
	}
	// Should have 2 messages: user + error agent message.
	if len(task.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(task.Messages))
	}
	if task.Messages[1].Parts[0].Text != "Error: boom" {
		t.Errorf("expected error text %q, got %q", "Error: boom", task.Messages[1].Parts[0].Text)
	}
}

func TestNodeHandler_UnknownMethod(t *testing.T) {
	exec := &mockExecutor{result: "unused"}
	def := &engine.NodeDefinition{ID: "n1", Type: engine.NodeTypeAgent, Config: map[string]any{}}
	handler := NewNodeHandler(exec, def)

	body := makeJSONRPCBody("a2a.unknownMethod", nil)

	req := httptest.NewRequest(http.MethodPost, "/a2a/nodes/n1", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var resp a2atypes.JSONRPCResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected JSON-RPC error, got nil")
	}
	// Code should be a float64 after JSON round-trip.
	code := int(resp.Error.Code)
	if code != -32601 {
		t.Errorf("expected error code -32601, got %d", code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("expected error message %q, got %q", "Method not found", resp.Error.Message)
	}
}

func TestResultToArtifact_String(t *testing.T) {
	a := resultToArtifact("hello world")
	if len(a.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(a.Parts))
	}
	if a.Parts[0].Type != "text" {
		t.Errorf("expected part type %q, got %q", "text", a.Parts[0].Type)
	}
	if a.Parts[0].Text != "hello world" {
		t.Errorf("expected text %q, got %q", "hello world", a.Parts[0].Text)
	}
	if a.Index != 0 {
		t.Errorf("expected index 0, got %d", a.Index)
	}
}

func TestResultToArtifact_Map(t *testing.T) {
	m := map[string]any{"key": "value", "count": float64(42)}
	a := resultToArtifact(m)
	if len(a.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(a.Parts))
	}
	if a.Parts[0].Type != "data" {
		t.Errorf("expected part type %q, got %q", "data", a.Parts[0].Type)
	}
	if a.Parts[0].MimeType != "application/json" {
		t.Errorf("expected mime type %q, got %q", "application/json", a.Parts[0].MimeType)
	}
	// Verify data survived.
	dataMap, ok := a.Parts[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be map[string]any, got %T", a.Parts[0].Data)
	}
	if dataMap["key"] != "value" {
		t.Errorf("expected key %q, got %q", "value", dataMap["key"])
	}
}

func TestResultToArtifact_Slice(t *testing.T) {
	s := []any{"a", "b", "c"}
	a := resultToArtifact(s)
	if len(a.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(a.Parts))
	}
	if a.Parts[0].Type != "data" {
		t.Errorf("expected part type %q, got %q", "data", a.Parts[0].Type)
	}
	if a.Parts[0].MimeType != "application/json" {
		t.Errorf("expected mime type %q, got %q", "application/json", a.Parts[0].MimeType)
	}
}

func TestAgentCardFromNodeDef(t *testing.T) {
	def := &engine.NodeDefinition{
		ID:   "summarizer",
		Type: engine.NodeTypeAgent,
		Config: map[string]any{
			"system_prompt": "You are a summarization agent.",
			"model":         "anthropic/claude-sonnet-4-20250514",
		},
	}
	card := AgentCardFromNodeDef(def, "http://localhost:8080")

	if card.Name != "summarizer" {
		t.Errorf("expected name %q, got %q", "summarizer", card.Name)
	}
	if card.Description != "You are a summarization agent." {
		t.Errorf("expected description from system_prompt, got %q", card.Description)
	}
	if card.URL != "http://localhost:8080/a2a/nodes/summarizer" {
		t.Errorf("expected URL %q, got %q", "http://localhost:8080/a2a/nodes/summarizer", card.URL)
	}
	if card.Capabilities.Streaming != false {
		t.Error("expected streaming false")
	}
	if len(card.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(card.Skills))
	}
	if card.Skills[0].ID != "summarizer" {
		t.Errorf("expected skill ID %q, got %q", "summarizer", card.Skills[0].ID)
	}
	if card.Skills[0].Name != "agent: summarizer" {
		t.Errorf("expected skill name %q, got %q", "agent: summarizer", card.Skills[0].Name)
	}

	// Test fallback description when no system_prompt.
	defNoPrompt := &engine.NodeDefinition{
		ID:     "basic",
		Type:   engine.NodeTypeInput,
		Config: map[string]any{},
	}
	card2 := AgentCardFromNodeDef(defNoPrompt, "http://example.com")
	if card2.Description != "Upal input node: basic" {
		t.Errorf("expected fallback description, got %q", card2.Description)
	}
}

func TestAgentCardHandler(t *testing.T) {
	def := &engine.NodeDefinition{
		ID:     "my-agent",
		Type:   engine.NodeTypeAgent,
		Config: map[string]any{},
	}
	handler := NewAgentCardHandler(def, "http://localhost:8080")

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var card a2atypes.AgentCard
	if err := json.Unmarshal(rec.Body.Bytes(), &card); err != nil {
		t.Fatalf("failed to decode agent card: %v", err)
	}
	if card.Name != "my-agent" {
		t.Errorf("expected name %q, got %q", "my-agent", card.Name)
	}
	if card.URL != "http://localhost:8080/a2a/nodes/my-agent" {
		t.Errorf("expected URL %q, got %q", "http://localhost:8080/a2a/nodes/my-agent", card.URL)
	}
}
