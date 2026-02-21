package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"time"

	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// ApprovalNodeBuilder creates agents that send an approval request via a
// notification channel and then wait for a human response (approve/reject).
//
// Config:
//   - connection_id: ID of the Connection for sending the request
//   - message: template for the approval request message
//   - channel: (optional) override destination
//   - timeout: timeout in seconds (default 3600 = 1 hour)
//   - actions: (optional) JSON array of action labels, e.g. ["Approve","Reject"]
type ApprovalNodeBuilder struct{}

func (b *ApprovalNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeApproval }

func (b *ApprovalNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	connID, _ := nd.Config["connection_id"].(string)
	messageTpl, _ := nd.Config["message"].(string)
	channel, _ := nd.Config["channel"].(string)

	timeoutSec := 3600
	if v, ok := nd.Config["timeout"].(float64); ok && v > 0 {
		timeoutSec = int(v)
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Approval node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				// 1. Send notification if connection is configured.
				if connID != "" && deps.SenderReg != nil && deps.ConnResolver != nil {
					conn, err := deps.ConnResolver.Resolve(context.Background(), connID)
					if err != nil {
						yield(nil, fmt.Errorf("approval node %q: resolve connection: %w", nodeID, err))
						return
					}
					if channel != "" {
						if conn.Extras == nil {
							conn.Extras = make(map[string]any)
						}
						conn.Extras["chat_id"] = channel
						conn.Extras["channel"] = channel
					}
					message := resolveTemplateFromState(messageTpl, state)
					sender, err := deps.SenderReg.Get(conn.Type)
					if err != nil {
						yield(nil, fmt.Errorf("approval node %q: %w", nodeID, err))
						return
					}
					if err := sender.Send(context.Background(), conn, message); err != nil {
						yield(nil, fmt.Errorf("approval node %q: send: %w", nodeID, err))
						return
					}
				}

				// 2. Emit waiting event.
				waitEvent := session.NewEvent(ctx.InvocationID())
				waitEvent.Author = nodeID
				waitEvent.Branch = ctx.Branch()
				waitEvent.Actions.StateDelta["__status__"] = string(upal.NodeStatusWaiting)
				if !yield(waitEvent, nil) {
					return
				}

				// 3. Wait for resume.
				handle := ExecutionHandleFromContext(ctx)
				if handle == nil {
					yield(nil, fmt.Errorf("approval node %q: no ExecutionHandle in context", nodeID))
					return
				}

				timeout := time.Duration(timeoutSec) * time.Second
				done := make(chan map[string]any, 1)
				go func() {
					done <- handle.WaitForResume(nodeID)
				}()

				var payload map[string]any
				select {
				case <-time.After(timeout):
					yield(nil, fmt.Errorf("approval node %q: timeout after %v", nodeID, timeout))
					return
				case <-ctx.Done():
					yield(nil, ctx.Err())
					return
				case payload = <-done:
				}

				// 4. Store result in session state.
				result, _ := json.Marshal(payload)
				_ = state.Set(nodeID, string(result))

				action, _ := payload["action"].(string)
				_ = state.Set(nodeID+"_action", action)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(string(result))},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = string(result)
				event.Actions.StateDelta[nodeID+"_action"] = action
				yield(event, nil)
			}
		},
	})
}
