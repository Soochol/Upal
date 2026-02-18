package nodes

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/tools"
)

type ToolNode struct {
	tools *tools.Registry
}

func NewToolNode(tools *tools.Registry) *ToolNode {
	return &ToolNode{tools: tools}
}

func (n *ToolNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	toolName, _ := def.Config["tool"].(string)
	if toolName == "" {
		return nil, fmt.Errorf("tool node %q: tool name is required", def.ID)
	}
	inputTemplate, _ := def.Config["input"].(string)
	input := resolveTemplate(inputTemplate, state)
	return n.tools.Execute(ctx, toolName, input)
}
