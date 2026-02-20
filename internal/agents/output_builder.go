package agents

import (
	"fmt"
	"iter"
	"sort"
	"strings"

	"github.com/soochol/upal/internal/output"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// OutputNodeBuilder creates agents that collect upstream results and
// optionally format them via an LLM.
type OutputNodeBuilder struct{}

func (b *OutputNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeOutput }

func (b *OutputNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	promptTpl, _ := nd.Config["prompt"].(string)
	formatter := output.NewFormatter(nd.Config, deps.LLMs, resolveLLM)

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Output node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				content := collectOutputContent(promptTpl, nodeID, state)

				result, err := formatter.Format(ctx, content)
				if err != nil {
					result = content
				}

				_ = state.Set(nodeID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(result)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = result
				yield(event, nil)
			}
		},
	})
}

// collectOutputContent resolves the prompt template from session state,
// or falls back to collecting all non-internal state values.
func collectOutputContent(promptTpl, nodeID string, state session.State) string {
	if promptTpl != "" {
		return resolveTemplateFromState(promptTpl, state)
	}

	var keys []string
	for k := range state.All() {
		if !strings.HasPrefix(k, "__") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		if k == nodeID {
			continue
		}
		v, err := state.Get(k)
		if err != nil || v == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%v", v))
	}

	return strings.Join(parts, "\n\n")
}
