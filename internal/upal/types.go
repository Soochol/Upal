package upal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// GenerateID creates a random ID with the given prefix, e.g. "wf-abc123".
func GenerateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

type NodeType string

const (
	NodeTypeInput    NodeType = "input"
	NodeTypeAgent    NodeType = "agent"
	NodeTypeOutput   NodeType = "output"
	NodeTypeBranch       NodeType = "branch"
	NodeTypeIterator     NodeType = "iterator"
	NodeTypeNotification NodeType = "notification"
	NodeTypeSensor       NodeType = "sensor"
	NodeTypeApproval     NodeType = "approval"
	NodeTypeSubWorkflow  NodeType = "subworkflow"
)

type WorkflowDefinition struct {
	Name    string            `json:"name" yaml:"name"`
	Version int               `json:"version" yaml:"version"`
	Nodes   []NodeDefinition  `json:"nodes" yaml:"nodes"`
	Edges   []EdgeDefinition  `json:"edges" yaml:"edges"`
	Groups  []GroupDefinition `json:"groups,omitempty" yaml:"groups,omitempty"`
}

type NodeDefinition struct {
	ID     string         `json:"id" yaml:"id"`
	Type   NodeType       `json:"type" yaml:"type"`
	Config map[string]any `json:"config" yaml:"config"`
	Group  string         `json:"group,omitempty" yaml:"group,omitempty"`
}

// TriggerRule determines when an edge is traversed based on the parent
// node's execution outcome.
type TriggerRule string

const (
	// TriggerOnSuccess traverses the edge only if the parent succeeded (default).
	TriggerOnSuccess TriggerRule = "on_success"
	// TriggerOnFailure traverses the edge only if the parent failed.
	TriggerOnFailure TriggerRule = "on_failure"
	// TriggerAlways traverses the edge regardless of parent outcome.
	TriggerAlways TriggerRule = "always"
)

// NodeStatus represents the execution state of a node within a run.
type NodeStatus string

const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusRunning   NodeStatus = "running"
	NodeStatusWaiting   NodeStatus = "waiting"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusFailed    NodeStatus = "failed"
	NodeStatusSkipped   NodeStatus = "skipped"
)

type EdgeDefinition struct {
	From        string      `json:"from" yaml:"from"`
	To          string      `json:"to" yaml:"to"`
	Loop        *LoopConfig `json:"loop,omitempty" yaml:"loop,omitempty"`
	Condition   string      `json:"condition,omitempty" yaml:"condition,omitempty"`
	TriggerRule TriggerRule `json:"trigger_rule,omitempty" yaml:"trigger_rule,omitempty"`
}

type LoopConfig struct {
	MaxIterations int    `json:"max_iterations" yaml:"max_iterations"`
	ExitWhen      string `json:"exit_when" yaml:"exit_when"`
}

type GroupDefinition struct {
	ID    string `json:"id" yaml:"id"`
	Label string `json:"label" yaml:"label"`
	Color string `json:"color,omitempty" yaml:"color,omitempty"`
}

// ConnectionType identifies the kind of external service a connection targets.
type ConnectionType string

const (
	ConnTypeTelegram ConnectionType = "telegram"
	ConnTypeSlack    ConnectionType = "slack"
	ConnTypeHTTP     ConnectionType = "http"
	ConnTypeSMTP     ConnectionType = "smtp"
)

// Connection stores credentials and configuration for an external service.
type Connection struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Type     ConnectionType `json:"type"`
	Host     string         `json:"host,omitempty"`
	Port     int            `json:"port,omitempty"`
	Login    string         `json:"login,omitempty"`
	Password string         `json:"password,omitempty"` // encrypted at rest
	Token    string         `json:"token,omitempty"`    // encrypted at rest
	Extras   map[string]any `json:"extras,omitempty"`
}

// ConnectionSafe is the API-safe view of a Connection with secrets masked.
type ConnectionSafe struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   ConnectionType `json:"type"`
	Host   string         `json:"host,omitempty"`
	Port   int            `json:"port,omitempty"`
	Login  string         `json:"login,omitempty"`
	Extras map[string]any `json:"extras,omitempty"`
}

// Safe returns a ConnectionSafe view with secrets removed.
func (c *Connection) Safe() ConnectionSafe {
	return ConnectionSafe{
		ID:     c.ID,
		Name:   c.Name,
		Type:   c.Type,
		Host:   c.Host,
		Port:   c.Port,
		Login:  c.Login,
		Extras: c.Extras,
	}
}

// ExecutionHandle represents a running workflow execution.
// Nodes that need to pause (sensor, approval) call WaitForResume
// to block until an external signal arrives via Resume.
type ExecutionHandle struct {
	RunID string

	mu      sync.Mutex
	waitChs map[string]chan map[string]any
}

// NewExecutionHandle creates a handle for a workflow run.
func NewExecutionHandle(runID string) *ExecutionHandle {
	return &ExecutionHandle{
		RunID:   runID,
		waitChs: make(map[string]chan map[string]any),
	}
}

// WaitForResume blocks until Resume is called for the given node.
func (h *ExecutionHandle) WaitForResume(nodeID string) map[string]any {
	h.mu.Lock()
	ch := make(chan map[string]any, 1)
	h.waitChs[nodeID] = ch
	h.mu.Unlock()
	return <-ch
}

// Resume unblocks a waiting node with the given payload.
func (h *ExecutionHandle) Resume(nodeID string, payload map[string]any) error {
	h.mu.Lock()
	ch, ok := h.waitChs[nodeID]
	if ok {
		delete(h.waitChs, nodeID)
	}
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("node %q is not waiting for resume", nodeID)
	}
	ch <- payload
	return nil
}

// Pipeline orchestrates a sequence of Stages (workflows, approvals, schedules).
type Pipeline struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Stages      []Stage   `json:"stages"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Stage is a single step in a Pipeline.
type Stage struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Type      string      `json:"type"` // "workflow", "approval", "schedule", "trigger", "transform"
	Config    StageConfig `json:"config"`
	DependsOn []string    `json:"depends_on,omitempty"`
}

// StageConfig holds type-specific configuration for a Stage.
type StageConfig struct {
	// Workflow stage
	WorkflowName string            `json:"workflow_name,omitempty"`
	InputMapping map[string]string `json:"input_mapping,omitempty"`

	// Approval stage
	Message      string `json:"message,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`
	Timeout      int    `json:"timeout,omitempty"`

	// Schedule stage
	Cron     string `json:"cron,omitempty"`
	Timezone string `json:"timezone,omitempty"`

	// Trigger stage
	TriggerID string `json:"trigger_id,omitempty"`

	// Transform stage
	Expression string `json:"expression,omitempty"`
}

// PipelineRun tracks a single execution of a Pipeline.
type PipelineRun struct {
	ID           string                  `json:"id"`
	PipelineID   string                  `json:"pipeline_id"`
	Status       string                  `json:"status"` // pending, running, waiting, completed, failed
	CurrentStage string                  `json:"current_stage,omitempty"`
	StageResults map[string]*StageResult `json:"stage_results,omitempty"`
	StartedAt    time.Time               `json:"started_at"`
	CompletedAt  *time.Time              `json:"completed_at,omitempty"`
}

// StageResult is the output of a completed Stage.
type StageResult struct {
	StageID     string         `json:"stage_id"`
	Status      string         `json:"status"` // pending, running, waiting, completed, skipped, failed
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}
