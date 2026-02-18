package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestInputNode_Execute(t *testing.T) {
	node := &InputNode{}
	def := &engine.NodeDefinition{ID: "input1", Type: engine.NodeTypeInput, Config: map[string]any{"input_type": "text", "label": "Enter topic"}}
	state := map[string]any{"__user_input__input1": "AI trends 2026"}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "AI trends 2026" {
		t.Errorf("result: got %q, want 'AI trends 2026'", result)
	}
}

func TestInputNode_Execute_MissingInput(t *testing.T) {
	node := &InputNode{}
	def := &engine.NodeDefinition{ID: "input1", Type: engine.NodeTypeInput}
	_, err := node.Execute(context.Background(), def, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}
