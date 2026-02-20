package tools

import "context"

// Tool is a custom tool executed by the Upal backend during agent runs.
type Tool interface {
	Name() string
	Description() string
	InputSchema() map[string]any
	Execute(ctx context.Context, input any) (any, error)
}

// NativeTool describes a tool managed by the LLM provider (e.g. web search).
// It is NOT executed by Upal — the provider handles it server-side.
type NativeTool struct {
	ToolName    string
	ToolDesc    string
}

func (n NativeTool) Name() string        { return n.ToolName }
func (n NativeTool) Description() string  { return n.ToolDesc }

// Well-known native tools.
var WebSearch = NativeTool{
	ToolName: "web_search",
	ToolDesc: "Search the web using Google Search",
}
