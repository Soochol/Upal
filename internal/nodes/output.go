package nodes

import (
	"context"
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/engine"
)

type OutputNode struct{}

func (n *OutputNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	var parts []string
	for key, val := range state {
		if strings.HasPrefix(key, "__") {
			continue
		}
		if s, ok := val.(string); ok && s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("output node %q: no data to output", def.ID)
	}
	return strings.Join(parts, "\n\n"), nil
}
