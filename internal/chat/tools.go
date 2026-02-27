package chat

import "context"

type ChatTool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema for LLM function declaration
	Execute     func(ctx context.Context, args map[string]any) (any, error)
}
