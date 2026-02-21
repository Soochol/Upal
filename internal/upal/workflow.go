package upal

type NodeType string

const (
	NodeTypeInput  NodeType = "input"
	NodeTypeAgent  NodeType = "agent"
	NodeTypeOutput NodeType = "output"
)

type WorkflowDefinition struct {
	Name         string            `json:"name" yaml:"name"`
	Description  string            `json:"description,omitempty" yaml:"description,omitempty"`
	Version      int               `json:"version" yaml:"version"`
	Nodes        []NodeDefinition  `json:"nodes" yaml:"nodes"`
	Edges        []EdgeDefinition  `json:"edges" yaml:"edges"`
	Groups       []GroupDefinition `json:"groups,omitempty" yaml:"groups,omitempty"`
	ThumbnailSVG string            `json:"thumbnail_svg,omitempty" yaml:"thumbnail_svg,omitempty"`
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
