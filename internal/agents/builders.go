package agents

import (
	"fmt"
	"iter"
	"regexp"
	"sort"
	"strings"

	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/agent/remoteagent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/genai"
)

// BuildAgent creates an ADK Agent from a NodeDefinition.
func BuildAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	switch nd.Type {
	case upal.NodeTypeInput:
		return buildInputAgent(nd)
	case upal.NodeTypeOutput:
		return buildOutputAgent(nd)
	case upal.NodeTypeTool:
		return buildToolAgent(nd, toolReg)
	case upal.NodeTypeAgent:
		return buildLLMAgent(nd, llms, toolReg)
	case upal.NodeTypeExternal:
		return buildRemoteAgent(nd)
	default:
		return nil, fmt.Errorf("unknown node type %q for node %q", nd.Type, nd.ID)
	}
}

// buildInputAgent creates a custom Agent that reads user input from session state.
// It reads __user_input__{nodeID} from state, stores it under the node ID, and yields an event.
func buildInputAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
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

// buildOutputAgent creates a custom Agent that collects all non-__ prefixed state keys,
// sorts them, joins their string values, and stores the result under the node ID.
func buildOutputAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Output node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

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

				result := strings.Join(parts, "\n\n")
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

// buildToolAgent creates a custom Agent that resolves templates in config["input"],
// calls the tool registry, and stores the result under the node ID.
func buildToolAgent(nd *upal.NodeDefinition, toolReg *tools.Registry) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Tool node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				toolName, _ := nd.Config["tool"].(string)
				inputTpl, _ := nd.Config["input"].(string)
				resolvedInput := resolveTemplateFromState(inputTpl, state)

				var result any
				var execErr error
				if toolReg != nil && toolName != "" {
					result, execErr = toolReg.Execute(ctx, toolName, resolvedInput)
				}

				if execErr != nil {
					yield(nil, fmt.Errorf("tool %q execution failed: %w", toolName, execErr))
					return
				}

				resultStr := fmt.Sprintf("%v", result)
				_ = state.Set(nodeID, resultStr)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(resultStr)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = resultStr
				yield(event, nil)
			}
		},
	})
}

// buildLLMAgent creates an LLM agent using llmagent.New() with model, instruction,
// output key, and tools from the node config.
func buildLLMAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	nodeID := nd.ID

	modelID, _ := nd.Config["model"].(string)
	systemPrompt, _ := nd.Config["system_prompt"].(string)

	// Resolve the LLM from "provider/model" format.
	var llm adkmodel.LLM
	if modelID != "" && llms != nil {
		parts := strings.SplitN(modelID, "/", 2)
		providerName := parts[0]
		if l, ok := llms[providerName]; ok {
			llm = l
		}
	}

	if llm == nil {
		return nil, fmt.Errorf("no LLM found for model %q in node %q", modelID, nodeID)
	}

	// Adapt tools from config.
	var adkTools []adktool.Tool
	if toolNames, ok := nd.Config["tools"].([]any); ok && toolReg != nil {
		for _, tn := range toolNames {
			name, ok := tn.(string)
			if !ok {
				continue
			}
			t, found := toolReg.Get(name)
			if found {
				adkTools = append(adkTools, NewADKTool(t))
			}
		}
	}

	cfg := llmagent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("LLM agent node %s", nodeID),
		Model:       llm,
		Instruction: systemPrompt,
		OutputKey:   nodeID,
		Tools:       adkTools,
	}

	return llmagent.New(cfg)
}

// buildRemoteAgent creates a remote A2A agent using remoteagent.NewA2A().
func buildRemoteAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	nodeID := nd.ID
	endpointURL, _ := nd.Config["endpoint_url"].(string)

	if endpointURL == "" {
		return nil, fmt.Errorf("external node %q missing endpoint_url", nodeID)
	}

	return remoteagent.NewA2A(remoteagent.A2AConfig{
		Name:            nodeID,
		Description:     fmt.Sprintf("Remote A2A agent node %s", nodeID),
		AgentCardSource: endpointURL,
	})
}

// templatePattern matches {{key}} or {{key.subkey}} placeholders.
var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

// resolveTemplateFromState replaces {{key}} placeholders in a template string
// with values from session state. Unresolved placeholders are left as-is.
func resolveTemplateFromState(template string, state session.State) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		val, err := state.Get(key)
		if err != nil || val == nil {
			return match
		}
		return fmt.Sprintf("%v", val)
	})
}
