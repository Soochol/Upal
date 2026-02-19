package agents

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestNewDAGAgent(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "test-wf",
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "input1", To: "output1"}},
	}
	a, err := NewDAGAgent(wf, nil, nil)
	if err != nil {
		t.Fatalf("new dag agent: %v", err)
	}
	if a.Name() != "test-wf" {
		t.Fatalf("expected 'test-wf', got %q", a.Name())
	}
}

func TestNewDAGAgent_InvalidDAG(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "bad-wf",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "a", To: "nonexistent"}},
	}
	_, err := NewDAGAgent(wf, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid DAG")
	}
}

func TestNewDAGAgent_SubAgents(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "multi-wf",
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "input2", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "input1", To: "output1"},
			{From: "input2", To: "output1"},
		},
	}
	a, err := NewDAGAgent(wf, nil, nil)
	if err != nil {
		t.Fatalf("new dag agent: %v", err)
	}
	if len(a.SubAgents()) != 3 {
		t.Fatalf("expected 3 sub-agents, got %d", len(a.SubAgents()))
	}
}
