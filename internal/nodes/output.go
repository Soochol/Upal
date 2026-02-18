package nodes

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/soochol/upal/internal/engine"
)

type OutputNode struct{}

func (n *OutputNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	// Collect non-internal keys and sort for deterministic output
	var keys []string
	for key := range state {
		if strings.HasPrefix(key, "__") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		if s, ok := state[key].(string); ok && s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("output node %q: no data to output", def.ID)
	}
	return strings.Join(parts, "\n\n"), nil
}
