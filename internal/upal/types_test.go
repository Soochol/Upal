package upal

import (
	"encoding/json"
	"testing"
)

func TestWorkflowDefinitionJSON(t *testing.T) {
	wf := WorkflowDefinition{
		Name:    "test-wf",
		Version: 1,
		Nodes: []NodeDefinition{
			{ID: "input1", Type: NodeTypeInput, Config: map[string]any{}},
			{ID: "agent1", Type: NodeTypeAgent, Config: map[string]any{"model": "anthropic/claude-sonnet-4-20250514"}},
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
	if got.Name != wf.Name || len(got.Nodes) != 2 || len(got.Edges) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}
}
