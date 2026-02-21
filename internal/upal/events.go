package upal

import (
	"github.com/soochol/upal/internal/llmutil"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// WorkflowEvent is a domain event emitted during workflow execution.
// It decouples the business logic from transport concerns (SSE, A2A).
type WorkflowEvent struct {
	Type    string         // "node_started", "tool_call", "tool_result", "node_completed"
	NodeID  string
	Payload map[string]any
}

// RunResult contains the final state after a workflow execution completes.
type RunResult struct {
	SessionID string
	State     map[string]any
}

// SSE event type constants.
const (
	EventNodeStarted   = "node_started"
	EventToolCall      = "tool_call"
	EventToolResult    = "tool_result"
	EventNodeCompleted = "node_completed"
	EventNodeSkipped   = "node_skipped"
	EventNodeWaiting   = "node_waiting"
	EventNodeResumed   = "node_resumed"
)

// ClassifyEvent inspects an ADK session.Event and returns a WorkflowEvent
// with the appropriate type and normalized payload.
func ClassifyEvent(event *session.Event) WorkflowEvent {
	nodeID := event.Author
	content := event.LLMResponse.Content

	// Check for special status markers in StateDelta (e.g. skipped nodes).
	if status, ok := event.Actions.StateDelta["__status__"].(string); ok {
		switch status {
		case "skipped":
			return WorkflowEvent{
				Type:    EventNodeSkipped,
				NodeID:  nodeID,
				Payload: map[string]any{"node_id": nodeID},
			}
		case "waiting":
			return WorkflowEvent{
				Type:    EventNodeWaiting,
				NodeID:  nodeID,
				Payload: map[string]any{"node_id": nodeID},
			}
		}
	}

	// No content → started event.
	if content == nil || len(content.Parts) == 0 {
		return WorkflowEvent{
			Type:    EventNodeStarted,
			NodeID:  nodeID,
			Payload: map[string]any{"node_id": nodeID},
		}
	}

	// FunctionCall parts → tool_call.
	if hasFunctionCalls(content.Parts) {
		return WorkflowEvent{
			Type:   EventToolCall,
			NodeID: nodeID,
			Payload: map[string]any{
				"node_id": nodeID,
				"calls":   extractFunctionCalls(content.Parts),
			},
		}
	}

	// FunctionResponse parts → tool_result.
	if hasFunctionResponses(content.Parts) {
		return WorkflowEvent{
			Type:   EventToolResult,
			NodeID: nodeID,
			Payload: map[string]any{
				"node_id": nodeID,
				"results": extractFunctionResponses(content.Parts),
			},
		}
	}

	// Content (text + images) → node_completed.
	return WorkflowEvent{
		Type:   EventNodeCompleted,
		NodeID: nodeID,
		Payload: map[string]any{
			"node_id":     nodeID,
			"output":      llmutil.ExtractContent(&event.LLMResponse),
			"state_delta": event.Actions.StateDelta,
		},
	}
}

func hasFunctionCalls(parts []*genai.Part) bool {
	for _, p := range parts {
		if p.FunctionCall != nil {
			return true
		}
	}
	return false
}

func hasFunctionResponses(parts []*genai.Part) bool {
	for _, p := range parts {
		if p.FunctionResponse != nil {
			return true
		}
	}
	return false
}

func extractFunctionCalls(parts []*genai.Part) []map[string]any {
	var calls []map[string]any
	for _, p := range parts {
		if p.FunctionCall != nil {
			calls = append(calls, map[string]any{
				"name": p.FunctionCall.Name,
				"args": p.FunctionCall.Args,
			})
		}
	}
	return calls
}

func extractFunctionResponses(parts []*genai.Part) []map[string]any {
	var results []map[string]any
	for _, p := range parts {
		if p.FunctionResponse != nil {
			results = append(results, map[string]any{
				"name":     p.FunctionResponse.Name,
				"response": p.FunctionResponse.Response,
			})
		}
	}
	return results
}
