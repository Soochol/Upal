package nodes

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/engine"
)

type InputNode struct{}

func (n *InputNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	key := "__user_input__" + def.ID
	val, ok := state[key]
	if !ok {
		return nil, fmt.Errorf("no user input for node %q (expected state key %q)", def.ID, key)
	}
	return val, nil
}
