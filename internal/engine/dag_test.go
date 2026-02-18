package engine

import "testing"

func TestDAG_Build_LinearChain(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{{From: "a", To: "b"}, {From: "b", To: "c"}},
	}
	dag, err := BuildDAG(wf)
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}
	order := dag.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("order length: got %d, want 3", len(order))
	}
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("order: got %v, want [a b c]", order)
	}
}

func TestDAG_Build_FanOut(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeAgent},
			{ID: "d", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"}, {From: "a", To: "c"},
			{From: "b", To: "d"}, {From: "c", To: "d"},
		},
	}
	dag, err := BuildDAG(wf)
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}
	if len(dag.Children("a")) != 2 {
		t.Errorf("a children: got %d, want 2", len(dag.Children("a")))
	}
	if len(dag.Parents("d")) != 2 {
		t.Errorf("d parents: got %d, want 2", len(dag.Parents("d")))
	}
	roots := dag.Roots()
	if len(roots) != 1 || roots[0] != "a" {
		t.Errorf("roots: got %v, want [a]", roots)
	}
}

func TestDAG_Build_InvalidNode(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{{ID: "a", Type: NodeTypeInput}},
		Edges: []EdgeDefinition{{From: "a", To: "nonexistent"}},
	}
	_, err := BuildDAG(wf)
	if err == nil {
		t.Fatal("expected error for edge to nonexistent node")
	}
}

func TestDAG_DetectBackEdges(t *testing.T) {
	wf := &WorkflowDefinition{
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeAgent},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeAgent},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "b", Loop: &LoopConfig{MaxIterations: 3, ExitWhen: "done"}},
		},
	}
	dag, err := BuildDAG(wf)
	if err != nil {
		t.Fatalf("BuildDAG: %v", err)
	}
	backEdges := dag.BackEdges()
	if len(backEdges) != 1 {
		t.Fatalf("back edges: got %d, want 1", len(backEdges))
	}
	if backEdges[0].From != "c" || backEdges[0].To != "b" {
		t.Errorf("back edge: got %s->%s, want c->b", backEdges[0].From, backEdges[0].To)
	}
}
