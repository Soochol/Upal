package agents

import (
	"sync"
	"testing"

	dagpkg "github.com/soochol/upal/internal/dag"
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
	a, err := NewDAGAgent(wf, DefaultRegistry(), BuildDeps{})
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
	_, err := NewDAGAgent(wf, DefaultRegistry(), BuildDeps{})
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
	a, err := NewDAGAgent(wf, DefaultRegistry(), BuildDeps{})
	if err != nil {
		t.Fatalf("new dag agent: %v", err)
	}
	if len(a.SubAgents()) != 3 {
		t.Fatalf("expected 3 sub-agents, got %d", len(a.SubAgents()))
	}
}

// --- shouldRun / triggerMatches unit tests ---

func buildTestDAG(t *testing.T, wf *upal.WorkflowDefinition) *dagpkg.DAG {
	t.Helper()
	d, err := dagpkg.Build(wf)
	if err != nil {
		t.Fatalf("build DAG: %v", err)
	}
	return d
}

func TestShouldRun_RootAlwaysRuns(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name:  "root-test",
		Nodes: []upal.NodeDefinition{{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}}},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex
	outcomes := map[string]*nodeOutcome{}

	if !shouldRun(d, "a", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("root node should always run")
	}
}

func TestShouldRun_DefaultTriggerOnSuccess(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "default-trigger",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "a", To: "b"}},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex

	// Parent succeeded → child should run.
	outcomes := map[string]*nodeOutcome{
		"a": {Status: upal.NodeStatusCompleted},
	}
	if !shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("child should run when parent succeeded with default trigger")
	}

	// Parent failed → child should NOT run (default = on_success).
	outcomes["a"] = &nodeOutcome{Status: upal.NodeStatusFailed}
	if shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("child should not run when parent failed with default trigger")
	}
}

func TestShouldRun_TriggerOnFailure(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "failure-trigger",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "a", To: "b", TriggerRule: upal.TriggerOnFailure}},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex

	// Parent failed → child should run.
	outcomes := map[string]*nodeOutcome{
		"a": {Status: upal.NodeStatusFailed},
	}
	if !shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("child should run when parent failed with on_failure trigger")
	}

	// Parent succeeded → child should NOT run.
	outcomes["a"] = &nodeOutcome{Status: upal.NodeStatusCompleted}
	if shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("child should not run when parent succeeded with on_failure trigger")
	}
}

func TestShouldRun_TriggerAlways(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "always-trigger",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "a", To: "b", TriggerRule: upal.TriggerAlways}},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex

	for _, status := range []upal.NodeStatus{upal.NodeStatusCompleted, upal.NodeStatusFailed, upal.NodeStatusSkipped} {
		outcomes := map[string]*nodeOutcome{
			"a": {Status: status},
		}
		if !shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
			t.Fatalf("child should run with always trigger when parent status=%s", status)
		}
	}
}

func TestShouldRun_ConditionExpression(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "condition-test",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "a", To: "b", Condition: "a == 'yes'"}},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex
	outcomes := map[string]*nodeOutcome{
		"a": {Status: upal.NodeStatusCompleted},
	}

	// Condition met.
	state := &testState{data: map[string]any{}}
	_ = state.Set("a", "yes")
	if !shouldRun(d, "b", outcomes, &mu, state) {
		t.Fatal("child should run when condition is met")
	}

	// Condition not met.
	state2 := &testState{data: map[string]any{}}
	_ = state2.Set("a", "no")
	if shouldRun(d, "b", outcomes, &mu, state2) {
		t.Fatal("child should not run when condition is not met")
	}
}

func TestShouldRun_MultipleParents_AnyActiveEdge(t *testing.T) {
	// Node C has two parents: A (on_success) and B (on_failure).
	// If A succeeds and B fails, C should run because both edges are active.
	wf := &upal.WorkflowDefinition{
		Name: "multi-parent",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "c", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "c", TriggerRule: upal.TriggerOnSuccess},
			{From: "b", To: "c", TriggerRule: upal.TriggerOnFailure},
		},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex

	// A succeeded, B succeeded → only A's edge matches.
	outcomes := map[string]*nodeOutcome{
		"a": {Status: upal.NodeStatusCompleted},
		"b": {Status: upal.NodeStatusCompleted},
	}
	if !shouldRun(d, "c", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("C should run: A's on_success edge is active")
	}

	// A failed, B failed → only B's edge matches.
	outcomes["a"] = &nodeOutcome{Status: upal.NodeStatusFailed}
	outcomes["b"] = &nodeOutcome{Status: upal.NodeStatusFailed}
	if !shouldRun(d, "c", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("C should run: B's on_failure edge is active")
	}

	// A failed, B succeeded → neither edge matches.
	outcomes["a"] = &nodeOutcome{Status: upal.NodeStatusFailed}
	outcomes["b"] = &nodeOutcome{Status: upal.NodeStatusCompleted}
	if shouldRun(d, "c", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("C should not run: no active edges")
	}
}

// --- Error Fallback tests (Phase 1-C) ---

func TestShouldRun_ErrorFallbackPath(t *testing.T) {
	// A → B (on_success), A → C (on_failure), B → D, C → D (always)
	// When A fails: B should be skipped, C should run, D should run.
	wf := &upal.WorkflowDefinition{
		Name: "error-fallback",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "c", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "d", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "b", TriggerRule: upal.TriggerOnSuccess},
			{From: "a", To: "c", TriggerRule: upal.TriggerOnFailure},
			{From: "b", To: "d", TriggerRule: upal.TriggerOnSuccess},
			{From: "c", To: "d", TriggerRule: upal.TriggerAlways},
		},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex

	// Simulate A failed.
	outcomes := map[string]*nodeOutcome{
		"a": {Status: upal.NodeStatusFailed},
	}

	// B should NOT run (A failed, edge is on_success).
	if shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("B should not run when A fails with on_success trigger")
	}

	// C should run (A failed, edge is on_failure).
	if !shouldRun(d, "c", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("C should run when A fails with on_failure trigger")
	}

	// After B is skipped and C completes.
	outcomes["b"] = &nodeOutcome{Status: upal.NodeStatusSkipped}
	outcomes["c"] = &nodeOutcome{Status: upal.NodeStatusCompleted}

	// D should run (C→D is always, and C completed).
	if !shouldRun(d, "d", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("D should run: C→D (always) edge is active")
	}
}

func TestShouldRun_ErrorFallback_NoFallbackCancels(t *testing.T) {
	// A → B (on_success only). When A fails, B should NOT run.
	wf := &upal.WorkflowDefinition{
		Name: "no-fallback",
		Nodes: []upal.NodeDefinition{
			{ID: "a", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "b", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "a", To: "b"}, // default = on_success
		},
	}
	d := buildTestDAG(t, wf)
	var mu sync.RWMutex
	outcomes := map[string]*nodeOutcome{
		"a": {Status: upal.NodeStatusFailed},
	}
	if shouldRun(d, "b", outcomes, &mu, &testState{data: map[string]any{}}) {
		t.Fatal("B should not run when A fails and there's no failure edge")
	}
}

func TestTriggerMatches_NilParent(t *testing.T) {
	// nil parent (legacy path: done channel closed without recording outcome).
	if !triggerMatches("", nil) {
		t.Fatal("default trigger should match nil parent")
	}
	if !triggerMatches(upal.TriggerOnSuccess, nil) {
		t.Fatal("on_success should match nil parent")
	}
	if !triggerMatches(upal.TriggerAlways, nil) {
		t.Fatal("always should match nil parent")
	}
	if triggerMatches(upal.TriggerOnFailure, nil) {
		t.Fatal("on_failure should not match nil parent")
	}
}
