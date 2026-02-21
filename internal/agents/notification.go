package agents

import (
	"context"
	"fmt"
	"iter"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// NotificationNodeBuilder creates agents that send messages to external
// channels (Telegram, Slack, email) via the configured connection.
//
// Config:
//   - connection_id: ID of the Connection to use for sending
//   - message: template string with {{key}} placeholders resolved from state
//   - channel: (optional) override destination (e.g. chat_id, channel name)
type NotificationNodeBuilder struct{}

func (b *NotificationNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeNotification }

func (b *NotificationNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	connID, _ := nd.Config["connection_id"].(string)
	messageTpl, _ := nd.Config["message"].(string)
	channel, _ := nd.Config["channel"].(string)

	if deps.SenderReg == nil {
		return nil, fmt.Errorf("notification node %q: no SenderRegistry in BuildDeps", nodeID)
	}
	if deps.ConnResolver == nil {
		return nil, fmt.Errorf("notification node %q: no ConnectionResolver in BuildDeps", nodeID)
	}

	senderReg := deps.SenderReg
	connResolver := deps.ConnResolver

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Notification node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				// Resolve the connection (decrypted).
				conn, err := connResolver.Resolve(context.Background(), connID)
				if err != nil {
					yield(nil, fmt.Errorf("notification node %q: resolve connection %q: %w", nodeID, connID, err))
					return
				}

				// Override channel if configured on the node.
				if channel != "" {
					if conn.Extras == nil {
						conn.Extras = make(map[string]any)
					}
					conn.Extras["chat_id"] = channel
					conn.Extras["channel"] = channel
				}

				// Resolve message template.
				message := resolveTemplateFromState(messageTpl, state)

				// Pick the sender and send.
				sender, err := senderReg.Get(conn.Type)
				if err != nil {
					yield(nil, fmt.Errorf("notification node %q: %w", nodeID, err))
					return
				}

				if err := sender.Send(context.Background(), conn, message); err != nil {
					yield(nil, fmt.Errorf("notification node %q: send: %w", nodeID, err))
					return
				}

				// Store result in session state.
				result := fmt.Sprintf("sent via %s", conn.Type)
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
