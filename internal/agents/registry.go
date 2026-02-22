package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
)

// NodeBuilder constructs an ADK agent from a node definition.
// Each node type (input, output, agent) implements this interface.
type NodeBuilder interface {
	NodeType() upal.NodeType
	Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error)
}

// ConnectionResolver resolves a connection by ID with decrypted secrets.
type ConnectionResolver interface {
	Resolve(ctx context.Context, id string) (*upal.Connection, error)
}

// BuildDeps bundles dependencies available to all node builders.
// Fields may be nil — each builder checks for what it needs.
type BuildDeps struct {
	LLMs    map[string]adkmodel.LLM
	ToolReg *tools.Registry
}

// NodeRegistry maps node types to their builders.
type NodeRegistry struct {
	mu       sync.RWMutex
	builders map[upal.NodeType]NodeBuilder
}

// NewNodeRegistry creates an empty node registry.
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{builders: make(map[upal.NodeType]NodeBuilder)}
}

// Register adds a node builder to the registry.
func (r *NodeRegistry) Register(b NodeBuilder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[b.NodeType()] = b
}

// Build creates an ADK agent for the given node definition using the
// registered builder for that node type.
func (r *NodeRegistry) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	r.mu.RLock()
	b, ok := r.builders[nd.Type]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no builder registered for node type %q (node %q)", nd.Type, nd.ID)
	}
	return b.Build(nd, deps)
}

// DefaultRegistry returns a NodeRegistry pre-loaded with the built-in
// node types (input, output, agent). Useful for tests and backward compat.
func DefaultRegistry() *NodeRegistry {
	r := NewNodeRegistry()
	r.Register(&InputNodeBuilder{})
	r.Register(&OutputNodeBuilder{})
	r.Register(&LLMNodeBuilder{})
	r.Register(&ToolNodeBuilder{})
	return r
}
