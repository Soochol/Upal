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

// ToolNodeBuilder creates agents that execute a registered tool directly,
// bypassing the LLM entirely. Use for deterministic transformations (TTS,
// ffmpeg, file I/O) where no reasoning is required.
type ToolNodeBuilder struct{}

func (b *ToolNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeTool }

func (b *ToolNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID

	toolName, _ := nd.Config["tool"].(string)
	if toolName == "" {
		return nil, fmt.Errorf("tool node %q: missing required config field \"tool\"", nodeID)
	}

	if deps.ToolReg == nil {
		return nil, fmt.Errorf("tool node %q: no tool registry configured", nodeID)
	}

	if _, found := deps.ToolReg.Get(toolName); !found {
		return nil, fmt.Errorf("tool node %q: unknown tool %q", nodeID, toolName)
	}

	// Capture input config at build time; template resolution happens at runtime.
	var inputCfg map[string]any
	if raw, ok := nd.Config["input"].(map[string]any); ok {
		inputCfg = raw
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Tool node %s (tool: %s)", nodeID, toolName),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				// Resolve {{node_id}} templates in string values of the input map.
				resolved := resolveInputFromState(inputCfg, state)

				t, _ := deps.ToolReg.Get(toolName)
				result, err := t.Execute(ctx, resolved)
				if err != nil {
					yield(nil, fmt.Errorf("tool node %q: tool %q failed: %w", nodeID, toolName, err))
					return
				}

				_ = state.Set(nodeID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(fmt.Sprintf("%v", result))},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = result
				yield(event, nil)
			}
		},
	})
}

// resolveInputFromState resolves {{node_id}} template references in the string
// values of an input map using session state. Non-string values are passed through.
func resolveInputFromState(input map[string]any, state session.State) map[string]any {
	if input == nil {
		return nil
	}
	resolved := make(map[string]any, len(input))
	for k, v := range input {
		if s, ok := v.(string); ok {
			resolved[k] = resolveTemplateFromState(s, state)
		} else {
			resolved[k] = v
		}
	}
	return resolved
}
