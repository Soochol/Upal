package engine

import (
	"encoding/json"
	"testing"
)

func TestWorkflowDefinition_JSONRoundTrip(t *testing.T) {
	wf := WorkflowDefinition{
		Name:    "test-workflow",
		Version: 1,
		Nodes: []NodeDefinition{
			{ID: "input1", Type: NodeTypeInput, Config: map[string]any{"input_type": "text", "label": "Enter topic"}},
			{ID: "agent1", Type: NodeTypeAgent, Config: map[string]any{"model": "ollama/llama3.2", "system_prompt": "You are a researcher."}},
		},
		Edges: []EdgeDefinition{{From: "input1", To: "agent1"}},
	}

	data, err := json.Marshal(wf)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got WorkflowDefinition
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != wf.Name {
		t.Errorf("name: got %q, want %q", got.Name, wf.Name)
	}
	if len(got.Nodes) != 2 {
		t.Errorf("nodes: got %d, want 2", len(got.Nodes))
	}
	if len(got.Edges) != 1 {
		t.Errorf("edges: got %d, want 1", len(got.Edges))
	}
	if got.Nodes[0].Type != NodeTypeInput {
		t.Errorf("node type: got %q, want %q", got.Nodes[0].Type, NodeTypeInput)
	}
}

func TestEventType_String(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventNodeStarted, "node.started"},
		{EventNodeCompleted, "node.completed"},
		{EventNodeError, "node.error"},
		{EventModelRequest, "model.request"},
		{EventModelResponse, "model.response"},
		{EventToolCall, "tool.call"},
		{EventToolResult, "tool.result"},
	}
	for _, tt := range tests {
		if got := string(tt.et); got != tt.want {
			t.Errorf("EventType: got %q, want %q", got, tt.want)
		}
	}
}

func TestSessionStatus(t *testing.T) {
	s := &Session{ID: "sess-1", WorkflowID: "wf-1", State: make(map[string]any), Status: SessionRunning}
	if s.Status != SessionRunning {
		t.Errorf("status: got %q, want %q", s.Status, SessionRunning)
	}
}
