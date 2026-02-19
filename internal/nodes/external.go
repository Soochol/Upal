package nodes

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

type ExternalNode struct {
	client *a2aclient.Client
}

func NewExternalNode(client *a2aclient.Client) *ExternalNode {
	return &ExternalNode{client: client}
}

func (n *ExternalNode) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	endpointURL, ok := def.Config["endpoint_url"].(string)
	if !ok || endpointURL == "" {
		return nil, fmt.Errorf("external node %q: missing endpoint_url config", def.ID)
	}

	inputText := ""
	if msg, ok := state["__a2a_message__"].(string); ok {
		inputText = msg
	} else if input, ok := state["__user_input__"+def.ID]; ok {
		inputText = fmt.Sprintf("%v", input)
	}
	if inputText == "" {
		inputText = fmt.Sprintf("Execute external node %s", def.ID)
	}

	msg := a2atypes.Message{
		Role:  "user",
		Parts: []a2atypes.Part{a2atypes.TextPart(inputText)},
	}
	task, err := n.client.SendMessage(ctx, endpointURL, msg)
	if err != nil {
		return nil, fmt.Errorf("external agent call to %s: %w", endpointURL, err)
	}
	if task.Status == a2atypes.TaskFailed {
		return nil, fmt.Errorf("external agent %s failed", endpointURL)
	}
	if len(task.Artifacts) > 0 {
		text := task.Artifacts[0].FirstText()
		if text != "" {
			return text, nil
		}
		data := task.Artifacts[0].FirstData()
		if data != nil {
			return string(data), nil
		}
	}
	return "no output", nil
}
