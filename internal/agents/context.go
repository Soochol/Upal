package agents

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type execHandleKeyType struct{}

var execHandleKey = execHandleKeyType{}

// WithExecutionHandle attaches an ExecutionHandle to the context.
func WithExecutionHandle(ctx context.Context, h *upal.ExecutionHandle) context.Context {
	return context.WithValue(ctx, execHandleKey, h)
}

// ExecutionHandleFromContext retrieves the ExecutionHandle from the context.
func ExecutionHandleFromContext(ctx context.Context) *upal.ExecutionHandle {
	h, _ := ctx.Value(execHandleKey).(*upal.ExecutionHandle)
	return h
}

// ── Sub-workflow call-stack tracking ──

type callStackKeyType struct{}

var callStackKey = callStackKeyType{}

// SubWorkflowCallStack tracks the chain of workflow names to detect cycles.
type SubWorkflowCallStack struct {
	Names []string
}

// Contains returns true if the workflow name is already in the call stack.
func (s *SubWorkflowCallStack) Contains(name string) bool {
	for _, n := range s.Names {
		if n == name {
			return true
		}
	}
	return false
}

// WithCallStack attaches a sub-workflow call stack to the context.
func WithCallStack(ctx context.Context, stack *SubWorkflowCallStack) context.Context {
	return context.WithValue(ctx, callStackKey, stack)
}

// CallStackFromContext retrieves the sub-workflow call stack from the context.
// Returns an empty stack if none is set.
func CallStackFromContext(ctx context.Context) *SubWorkflowCallStack {
	s, _ := ctx.Value(callStackKey).(*SubWorkflowCallStack)
	if s == nil {
		return &SubWorkflowCallStack{}
	}
	return s
}
