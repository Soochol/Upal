package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/soochol/upal/internal/upal"
)

// mockWfLookup implements WorkflowLookup for tests.
type mockWfLookup struct {
	wfs map[string]*upal.WorkflowDefinition
}

func (m *mockWfLookup) Lookup(_ context.Context, name string) (*upal.WorkflowDefinition, error) {
	wf, ok := m.wfs[name]
	if !ok {
		return nil, fmt.Errorf("workflow %q not found", name)
	}
	return wf, nil
}

func TestSubWorkflowNodeBuilder_Build(t *testing.T) {
	builder := &SubWorkflowNodeBuilder{}
	if builder.NodeType() != upal.NodeTypeSubWorkflow {
		t.Fatalf("NodeType() = %q, want %q", builder.NodeType(), upal.NodeTypeSubWorkflow)
	}

	childWf := &upal.WorkflowDefinition{
		Name: "child-wf",
		Nodes: []upal.NodeDefinition{
			{ID: "inp", Type: upal.NodeTypeInput, Config: map[string]any{"value": "hi"}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "inp", To: "out"},
		},
	}

	wfLookup := &mockWfLookup{wfs: map[string]*upal.WorkflowDefinition{
		"child-wf": childWf,
	}}

	nd := &upal.NodeDefinition{
		ID:   "sub1",
		Type: upal.NodeTypeSubWorkflow,
		Config: map[string]any{
			"workflow_name": "child-wf",
			"input_mapping": map[string]any{
				"inp": "{{parent_input}}",
			},
		},
	}

	deps := BuildDeps{
		WfLookup:     wfLookup,
		NodeRegistry: DefaultRegistry(),
	}

	ag, err := builder.Build(nd, deps)
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if ag == nil {
		t.Fatal("Build() returned nil agent")
	}
}

func TestSubWorkflowNodeBuilder_MissingWorkflowName(t *testing.T) {
	builder := &SubWorkflowNodeBuilder{}

	nd := &upal.NodeDefinition{
		ID:     "sub1",
		Type:   upal.NodeTypeSubWorkflow,
		Config: map[string]any{},
	}

	_, err := builder.Build(nd, BuildDeps{})
	if err == nil {
		t.Fatal("expected error for missing workflow_name")
	}
}

func TestSubWorkflowNodeBuilder_NoDeps(t *testing.T) {
	builder := &SubWorkflowNodeBuilder{}

	nd := &upal.NodeDefinition{
		ID:   "sub1",
		Type: upal.NodeTypeSubWorkflow,
		Config: map[string]any{
			"workflow_name": "child-wf",
		},
	}

	// No WfLookup
	_, err := builder.Build(nd, BuildDeps{})
	if err == nil {
		t.Fatal("expected error when WfLookup is nil")
	}

	// No NodeRegistry
	wfLookup := &mockWfLookup{wfs: map[string]*upal.WorkflowDefinition{}}
	_, err = builder.Build(nd, BuildDeps{WfLookup: wfLookup})
	if err == nil {
		t.Fatal("expected error when NodeRegistry is nil")
	}
}

func TestSubWorkflowCallStack_CycleDetection(t *testing.T) {
	stack := &SubWorkflowCallStack{Names: []string{"wf-a", "wf-b"}}
	if !stack.Contains("wf-a") {
		t.Error("expected stack to contain wf-a")
	}
	if !stack.Contains("wf-b") {
		t.Error("expected stack to contain wf-b")
	}
	if stack.Contains("wf-c") {
		t.Error("expected stack to NOT contain wf-c")
	}
}

func TestCallStackFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	stack := CallStackFromContext(ctx)
	if len(stack.Names) != 0 {
		t.Errorf("expected empty stack, got %v", stack.Names)
	}
}

func TestCallStackFromContext_Roundtrip(t *testing.T) {
	ctx := context.Background()
	stack := &SubWorkflowCallStack{Names: []string{"wf-parent"}}
	ctx = WithCallStack(ctx, stack)

	got := CallStackFromContext(ctx)
	if len(got.Names) != 1 || got.Names[0] != "wf-parent" {
		t.Errorf("expected [wf-parent], got %v", got.Names)
	}
}
