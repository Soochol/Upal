package dag

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestBuildDAG(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: "input"},
			{ID: "b", Type: "agent"},
			{ID: "c", Type: "output"},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}
	d, err := Build(wf)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	order := d.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	idx := map[string]int{}
	for i, id := range order {
		idx[id] = i
	}
	if idx["a"] >= idx["b"] || idx["b"] >= idx["c"] {
		t.Fatalf("wrong order: %v", order)
	}
}

func TestBuildDAGCycleDetection(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "a"}, {ID: "b"},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}
	_, err := Build(wf)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
