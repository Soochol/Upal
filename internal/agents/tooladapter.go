package agents

import (
	"github.com/soochol/upal/internal/tools"
	adktool "google.golang.org/adk/tool"
)

// ADKTool wraps a Upal tools.Tool to satisfy ADK's tool.Tool interface.
type ADKTool struct {
	inner tools.Tool
}

// NewADKTool creates an ADKTool adapter from a Upal Tool.
func NewADKTool(t tools.Tool) *ADKTool {
	return &ADKTool{inner: t}
}

// Name returns the tool name.
func (a *ADKTool) Name() string { return a.inner.Name() }

// Description returns the tool description.
func (a *ADKTool) Description() string { return a.inner.Description() }

// IsLongRunning returns false; Upal tools are synchronous.
func (a *ADKTool) IsLongRunning() bool { return false }

// AdaptTools converts a slice of Upal tools to ADK tool.Tool interfaces.
func AdaptTools(upalTools []tools.Tool) []adktool.Tool {
	result := make([]adktool.Tool, len(upalTools))
	for i, t := range upalTools {
		result[i] = NewADKTool(t)
	}
	return result
}
