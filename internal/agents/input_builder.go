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

// InputNodeBuilder creates agents that read user input from session state.
type InputNodeBuilder struct{}

func (b *InputNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeInput }

func (b *InputNodeBuilder) Build(nd *upal.NodeDefinition, _ BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Input node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()
				key := "__user_input__" + nodeID
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
