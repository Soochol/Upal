package nodes

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/tools"
)

type addTool struct{}

func (a *addTool) Name() string               { return "add" }
func (a *addTool) Description() string         { return "Adds numbers" }
func (a *addTool) InputSchema() map[string]any { return nil }
func (a *addTool) Execute(ctx context.Context, input any) (any, error) { return "result: 42", nil }

func TestToolNode_Execute(t *testing.T) {
	toolReg := tools.NewRegistry()
	toolReg.Register(&addTool{})
	node := NewToolNode(toolReg)
	def := &engine.NodeDefinition{ID: "tool1", Type: engine.NodeTypeTool, Config: map[string]any{"tool": "add", "input": "{{input1}}"}}
	state := map[string]any{"input1": "some data"}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "result: 42" {
		t.Errorf("result: got %v", result)
	}
}
