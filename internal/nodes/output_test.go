package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestOutputNode_Execute(t *testing.T) {
	node := &OutputNode{}
	def := &engine.NodeDefinition{ID: "output1", Type: engine.NodeTypeOutput, Config: map[string]any{"output_type": "markdown"}}
	state := map[string]any{"agent1": "# Hello World\nThis is content."}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == nil {
		t.Error("result should not be nil")
	}
}
