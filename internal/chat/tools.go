package chat

import "context"

// ChatTool defines an action the chat LLM can invoke via tool calls.
type ChatTool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema for LLM function declaration
	Execute     func(ctx context.Context, args map[string]any) (any, error)
}
