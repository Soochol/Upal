package agents

import (
	"iter"
	"testing"

	"github.com/soochol/upal/internal/upal"
)

func TestBuildAgent_Input(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "input1", Type: upal.NodeTypeInput}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "input1" {
		t.Fatalf("expected 'input1', got %q", a.Name())
	}
}

func TestBuildAgent_Output(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "output1", Type: upal.NodeTypeOutput}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "output1" {
		t.Fatalf("expected 'output1', got %q", a.Name())
	}
}

func TestBuildAgent_Tool(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "tool1",
		Type: upal.NodeTypeTool,
		Config: map[string]any{
			"tool":  "test_tool",
			"input": "hello {{name}}",
		},
	}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "tool1" {
		t.Fatalf("expected 'tool1', got %q", a.Name())
	}
}

func TestBuildAgent_UnknownType(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "x", Type: "unknown"}
	_, err := BuildAgent(nd, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveTemplateFromState(t *testing.T) {
	state := &testState{
		data: map[string]any{
			"name":  "World",
			"input1": "Hello",
		},
	}

	tests := []struct {
		template string
		expected string
	}{
		{"Hello {{name}}", "Hello World"},
		{"{{input1}} {{name}}", "Hello World"},
		{"No templates here", "No templates here"},
		{"{{missing}}", "{{missing}}"},
	}

	for _, tt := range tests {
		result := resolveTemplateFromState(tt.template, state)
		if result != tt.expected {
			t.Errorf("resolveTemplateFromState(%q) = %q, want %q", tt.template, result, tt.expected)
		}
	}
}

// testState implements session.State for testing.
type testState struct {
	data map[string]any
}

func (s *testState) Get(key string) (any, error) {
	v, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (s *testState) Set(key string, val any) error {
	s.data[key] = val
	return nil
}

func (s *testState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}
