package upal

// WorkflowEvent is a domain event emitted during workflow execution.
// It decouples the business logic from transport concerns (SSE, A2A).
type WorkflowEvent struct {
	Type    string         `json:"type"`
	NodeID  string         `json:"node_id,omitempty"`
	Payload map[string]any `json:"payload"`
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
	EventError         = "error"
)
