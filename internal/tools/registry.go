package tools

import (
	"context"
	"fmt"
	"sync"
)

// Registry holds both custom tools (executed by Upal) and native tools
// (executed by the LLM provider). It provides a unified list for the API
// and lets callers distinguish between the two via IsNative.
type Registry struct {
	mu     sync.RWMutex
	tools  map[string]Tool
	native map[string]NativeTool
}

func NewRegistry() *Registry {
	return &Registry{
		tools:  make(map[string]Tool),
		native: make(map[string]NativeTool),
	}
}

// Register adds a custom tool (executed by Upal at runtime).
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// RegisterNative adds a provider-managed tool (not executed by Upal).
func (r *Registry) RegisterNative(t NativeTool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.native[t.Name()] = t
}

// Get returns a custom tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// IsNative returns true if the name refers to a registered native tool.
func (r *Registry) IsNative(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.native[name]
	return ok
}

func (r *Registry) Execute(ctx context.Context, name string, input any) (any, error) {
	t, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	return t.Execute(ctx, input)
}

// List returns all custom tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// ListNative returns all registered native tools.
func (r *Registry) ListNative() []NativeTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]NativeTool, 0, len(r.native))
	for _, t := range r.native {
		result = append(result, t)
	}
	return result
}

// AllNames returns the names of all tools (custom + native) for API listing.
type ToolInfo struct {
	Name        string
	Description string
	Native      bool
}

func (r *Registry) AllTools() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ToolInfo, 0, len(r.native)+len(r.tools))
	for _, t := range r.native {
		result = append(result, ToolInfo{Name: t.Name(), Description: t.Description(), Native: true})
	}
	for _, t := range r.tools {
		result = append(result, ToolInfo{Name: t.Name(), Description: t.Description(), Native: false})
	}
	return result
}
