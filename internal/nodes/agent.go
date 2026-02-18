package nodes

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/provider"
	"github.com/soochol/upal/internal/tools"
)

type AgentNode struct {
	providers *provider.Registry
	tools     *tools.Registry
	eventBus  *engine.EventBus
}

func NewAgentNode(providers *provider.Registry, tools *tools.Registry, eventBus *engine.EventBus) *AgentNode {
	return &AgentNode{providers: providers, tools: tools, eventBus: eventBus}
}

func (n *AgentNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	modelID, _ := def.Config["model"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("agent node %q: model is required", def.ID)
	}
	prov, modelName, err := n.providers.Resolve(modelID)
	if err != nil {
		return nil, fmt.Errorf("agent node %q: %w", def.ID, err)
	}

	var messages []provider.Message
	if sp, ok := def.Config["system_prompt"].(string); ok && sp != "" {
		messages = append(messages, provider.Message{Role: provider.RoleSystem, Content: resolveTemplate(sp, state)})
	}
	prompt, _ := def.Config["prompt"].(string)
	if prompt != "" {
		messages = append(messages, provider.Message{Role: provider.RoleUser, Content: resolveTemplate(prompt, state)})
	}

	var toolDefs []provider.ToolDefinition
	if toolNames, ok := def.Config["tools"].([]any); ok {
		for _, tn := range toolNames {
			name, _ := tn.(string)
			if t, ok := n.tools.Get(name); ok {
				toolDefs = append(toolDefs, provider.ToolDefinition{Name: t.Name(), Description: t.Description(), Parameters: t.InputSchema()})
			}
		}
	}

	maxTurns := 1
	if mt, ok := def.Config["max_turns"].(float64); ok {
		maxTurns = int(mt)
	}
	if len(toolDefs) > 0 && maxTurns < 2 {
		maxTurns = 10
	}

	for turn := 0; turn < maxTurns; turn++ {
		n.eventBus.Publish(engine.Event{
			ID: engine.GenerateID("ev"), NodeID: def.ID, Type: engine.EventModelRequest,
			Payload: map[string]any{"turn": turn, "model": modelID}, Timestamp: time.Now(),
		})

		resp, err := prov.ChatCompletion(ctx, &provider.ChatRequest{Model: modelName, Messages: messages, Tools: toolDefs})
		if err != nil {
			return nil, fmt.Errorf("agent node %q turn %d: %w", def.ID, turn, err)
		}

		n.eventBus.Publish(engine.Event{
			ID: engine.GenerateID("ev"), NodeID: def.ID, Type: engine.EventModelResponse,
			Payload: map[string]any{"content": resp.Content, "tool_calls": resp.ToolCalls}, Timestamp: time.Now(),
		})

		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		messages = append(messages, provider.Message{Role: provider.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls})

		for _, tc := range resp.ToolCalls {
			n.eventBus.Publish(engine.Event{
				ID: engine.GenerateID("ev"), NodeID: def.ID, Type: engine.EventToolCall,
				Payload: map[string]any{"tool": tc.Name, "args": tc.Arguments}, Timestamp: time.Now(),
			})
			result, err := n.tools.Execute(ctx, tc.Name, tc.Arguments)
			var resultStr string
			if err != nil {
				resultStr = fmt.Sprintf("Error: %v", err)
			} else {
				resultStr = fmt.Sprintf("%v", result)
			}
			n.eventBus.Publish(engine.Event{
				ID: engine.GenerateID("ev"), NodeID: def.ID, Type: engine.EventToolResult,
				Payload: map[string]any{"tool": tc.Name, "result": resultStr}, Timestamp: time.Now(),
			})
			messages = append(messages, provider.Message{Role: provider.RoleTool, Content: resultStr, ToolCallID: tc.ID})
		}
	}

	return nil, fmt.Errorf("agent node %q: exceeded max turns (%d)", def.ID, maxTurns)
}

var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

func resolveTemplate(template string, state map[string]any) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		parts := strings.SplitN(key, ".", 2)
		if val, ok := state[parts[0]]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})
}
