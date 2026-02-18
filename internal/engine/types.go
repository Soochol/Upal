package engine

import (
	"context"
	"time"
)

type NodeType string

const (
	NodeTypeInput  NodeType = "input"
	NodeTypeAgent  NodeType = "agent"
	NodeTypeTool   NodeType = "tool"
	NodeTypeOutput NodeType = "output"
)

type EventType string

const (
	EventNodeStarted   EventType = "node.started"
	EventNodeCompleted EventType = "node.completed"
	EventNodeError     EventType = "node.error"
	EventModelRequest  EventType = "model.request"
	EventModelResponse EventType = "model.response"
	EventToolCall      EventType = "tool.call"
	EventToolResult    EventType = "tool.result"
)

type SessionStatus string

const (
	SessionRunning   SessionStatus = "running"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
	SessionPaused    SessionStatus = "paused"
)

type WorkflowDefinition struct {
	Name    string           `json:"name" yaml:"name"`
	Version int              `json:"version" yaml:"version"`
	Nodes   []NodeDefinition `json:"nodes" yaml:"nodes"`
	Edges   []EdgeDefinition `json:"edges" yaml:"edges"`
}

type NodeDefinition struct {
	ID     string         `json:"id" yaml:"id"`
	Type   NodeType       `json:"type" yaml:"type"`
	Config map[string]any `json:"config" yaml:"config"`
}

type EdgeDefinition struct {
	From string      `json:"from" yaml:"from"`
	To   string      `json:"to" yaml:"to"`
	Loop *LoopConfig `json:"loop,omitempty" yaml:"loop,omitempty"`
}

type LoopConfig struct {
	MaxIterations int    `json:"max_iterations" yaml:"max_iterations"`
	ExitWhen      string `json:"exit_when" yaml:"exit_when"`
}

type Event struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	SessionID  string    `json:"session_id"`
	NodeID     string    `json:"node_id"`
	Type       EventType `json:"type"`
	Payload    any       `json:"payload"`
	Timestamp  time.Time `json:"timestamp"`
}

type Session struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflow_id"`
	State      map[string]any `json:"state"`
	Events     []Event        `json:"events"`
	Status     SessionStatus  `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// NodeExecutorInterface is the interface for executing nodes.
type NodeExecutorInterface interface {
	Execute(ctx context.Context, def *NodeDefinition, state map[string]any) (any, error)
}
