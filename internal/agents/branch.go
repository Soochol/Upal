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

// BranchNodeBuilder creates agents that evaluate an expression against
// upstream data and store the result in session state. Downstream edges
// use Condition expressions to select which branch to follow.
//
// Config:
//   - mode: "expression" (default) or "llm"
//   - expression: expr-lang expression evaluated against session state
//   - prompt: (llm mode) prompt template sent to the LLM for classification
//   - model: (llm mode) model ID in "provider/model" format
type BranchNodeBuilder struct{}

func (b *BranchNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeBranch }

func (b *BranchNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	mode, _ := nd.Config["mode"].(string)
	if mode == "" {
		mode = "expression"
	}
	expression, _ := nd.Config["expression"].(string)
	promptTpl, _ := nd.Config["prompt"].(string)
	modelID, _ := nd.Config["model"].(string)

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Branch node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()
				var result string

				switch mode {
				case "llm":
					res, err := branchLLM(ctx, promptTpl, modelID, state, deps)
					if err != nil {
						yield(nil, fmt.Errorf("branch node %q LLM: %w", nodeID, err))
						return
					}
					result = res
				default: // "expression"
					resolved := resolveTemplateFromState(expression, state)
					val, err := evaluateCondition(resolved, state)
					if err != nil {
						result = "false"
					} else if val {
						result = "true"
					} else {
						result = "false"
					}
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

// branchLLM sends a classification prompt to an LLM and returns the response text.
func branchLLM(ctx agent.InvocationContext, promptTpl, modelID string, state session.State, deps BuildDeps) (string, error) {
	llm, modelName := resolveLLM(modelID, deps.LLMs)
	if llm == nil {
		return "", fmt.Errorf("no LLM found for model %q", modelID)
	}

	resolvedPrompt := resolveTemplateFromState(promptTpl, state)

	req := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(
				"You are a classifier. Reply with ONLY the classification label, no explanation.",
				genai.RoleUser,
			),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(resolvedPrompt, genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", err
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty LLM response")
	}

	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			return p.Text, nil
		}
	}
	return "", fmt.Errorf("no text in LLM response")
}
