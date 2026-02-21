package agents

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestBranchNodeBuilder_Expression(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "branch1",
		Type: upal.NodeTypeBranch,
		Config: map[string]any{
			"mode":       "expression",
			"expression": "x > 5",
		},
	}
	a, err := (&BranchNodeBuilder{}).Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("build branch node: %v", err)
	}
	if a.Name() != "branch1" {
		t.Fatalf("expected name 'branch1', got %q", a.Name())
	}
}

func TestBranchNodeBuilder_DefaultMode(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "branch2",
		Type: upal.NodeTypeBranch,
		Config: map[string]any{
			"expression": "1 == 1",
		},
	}
	a, err := (&BranchNodeBuilder{}).Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("build branch node: %v", err)
	}
	if a.Name() != "branch2" {
		t.Fatalf("expected name 'branch2', got %q", a.Name())
	}
}

func TestBranchNodeBuilder_RegisteredInDefault(t *testing.T) {
	reg := DefaultRegistry()
	nd := &upal.NodeDefinition{
		ID:     "branch3",
		Type:   upal.NodeTypeBranch,
		Config: map[string]any{"expression": "true"},
	}
	a, err := reg.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("build from registry: %v", err)
	}
	if a.Name() != "branch3" {
		t.Fatalf("expected name 'branch3', got %q", a.Name())
	}
}
