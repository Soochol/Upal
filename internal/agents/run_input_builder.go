package agents

import (
	"fmt"
	"iter"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// RunInputNodeBuilder creates agents that read pipeline run briefs from session state.
// Unlike InputNodeBuilder (which reads __user_input__), this reads from __run_input__.
type RunInputNodeBuilder struct{}

func (b *RunInputNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeRunInput }

func (b *RunInputNodeBuilder) Build(nd *upal.NodeDefinition, _ BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Run input node %s — receives pipeline brief", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()
				key := "__run_input__" + nodeID
				val, err := state.Get(key)
				if err != nil || val == nil {
					val = ""
				}

				_ = state.Set(nodeID, val)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(fmt.Sprintf("%v", val))},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = val
				yield(event, nil)
			}
		},
	})
}
