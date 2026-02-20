package agents

import "context"

// NodeLogFunc is called to emit a log message scoped to a workflow node.
// The service layer provides the implementation that routes messages
// into the SSE event stream.
type NodeLogFunc func(nodeID, message string)

type nodeLogFuncKey struct{}

// WithNodeLogFunc returns a context carrying a node-scoped log function.
func WithNodeLogFunc(ctx context.Context, fn NodeLogFunc) context.Context {
	return context.WithValue(ctx, nodeLogFuncKey{}, fn)
}

func nodeLogFuncFromContext(ctx context.Context) NodeLogFunc {
	fn, _ := ctx.Value(nodeLogFuncKey{}).(NodeLogFunc)
	return fn
}
