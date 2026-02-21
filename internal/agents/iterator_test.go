package agents

import (
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestIteratorNodeBuilder_Build(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "iter1",
		Type: upal.NodeTypeIterator,
		Config: map[string]any{
			"source":   `["a","b","c"]`,
			"item_key": "current",
		},
	}
	a, err := (&IteratorNodeBuilder{}).Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("build iterator node: %v", err)
	}
	if a.Name() != "iter1" {
		t.Fatalf("expected name 'iter1', got %q", a.Name())
	}
}

func TestIteratorNodeBuilder_RegisteredInDefault(t *testing.T) {
	reg := DefaultRegistry()
	nd := &upal.NodeDefinition{
		ID:     "iter2",
		Type:   upal.NodeTypeIterator,
		Config: map[string]any{"source": "[]"},
	}
	a, err := reg.Build(nd, BuildDeps{})
	if err != nil {
		t.Fatalf("build from registry: %v", err)
	}
	if a.Name() != "iter2" {
		t.Fatalf("expected name 'iter2', got %q", a.Name())
	}
}

func TestParseJSONArray(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"json array", `["a","b","c"]`, 3},
		{"empty array", `[]`, 0},
		{"empty string", ``, 0},
		{"newline separated", "line1\nline2\nline3", 3},
		{"json number array", `[1,2,3]`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := parseJSONArray(tt.input)
			if err != nil {
				t.Fatalf("parseJSONArray(%q): %v", tt.input, err)
			}
			if len(items) != tt.want {
				t.Fatalf("parseJSONArray(%q) got %d items, want %d", tt.input, len(items), tt.want)
			}
		})
	}
}
