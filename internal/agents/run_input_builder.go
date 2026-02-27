package agents

import (
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
)

// RunInputNodeBuilder creates agents that read pipeline run briefs from session state.
// Unlike InputNodeBuilder (which reads __user_input__), this reads from __run_input__.
type RunInputNodeBuilder struct{}

func (b *RunInputNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeRunInput }

func (b *RunInputNodeBuilder) Build(nd *upal.NodeDefinition, _ BuildDeps) (agent.Agent, error) {
	return buildStateReaderAgent(nd.ID, "__run_input__", "Run input node %s — receives pipeline brief")
}
