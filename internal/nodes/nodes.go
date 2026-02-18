package nodes

import (
	"context"

	"github.com/soochol/upal/internal/engine"
)

// NodeExecutor executes a node with the given definition and session state.
type NodeExecutor interface {
	Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error)
}
