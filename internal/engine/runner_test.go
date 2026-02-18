package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

type mockNodeExecutor struct {
	results map[string]any
}

func (m *mockNodeExecutor) Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error) {
	if r, ok := m.results[def.ID]; ok {
		return r, nil
	}
	return fmt.Sprintf("output of %s", def.ID), nil
}

func TestRunner_LinearChain(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{{From: "a", To: "b"}, {From: "b", To: "c"}},
	}
	executor := &mockNodeExecutor{results: map[string]any{"a": "input data", "b": "processed data", "c": "final output"}}
	executors := map[NodeType]NodeExecutorInterface{
		NodeTypeInput: executor, NodeTypeAgent: executor, NodeTypeOutput: executor,
	}
	runner := NewRunner(NewEventBus(), NewSessionManager())
	result, err := runner.Run(context.Background(), wf, executors, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Status != SessionCompleted {
		t.Errorf("status: got %q, want completed", result.Status)
	}
}

func TestRunner_FanOutFanIn(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "fan-test",
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
	executed := make(map[string]bool)
	executor := &trackingExecutor{executed: executed}
	executors := map[NodeType]NodeExecutorInterface{
		NodeTypeInput: executor, NodeTypeAgent: executor, NodeTypeOutput: executor,
	}
	runner := NewRunner(NewEventBus(), NewSessionManager())
	_, err := runner.Run(context.Background(), wf, executors, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if !executed[id] {
			t.Errorf("node %q was not executed", id)
		}
	}
}

type trackingExecutor struct {
	mu       sync.Mutex
	executed map[string]bool
}

func (e *trackingExecutor) Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error) {
	e.mu.Lock()
	e.executed[def.ID] = true
	e.mu.Unlock()
	return fmt.Sprintf("output of %s", def.ID), nil
}
