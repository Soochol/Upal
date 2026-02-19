package upal

type NodeType string

const (
	NodeTypeInput    NodeType = "input"
	NodeTypeAgent    NodeType = "agent"
	NodeTypeTool     NodeType = "tool"
	NodeTypeOutput   NodeType = "output"
	NodeTypeExternal NodeType = "external"
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

type EdgeDefinition struct {
	From string      `json:"from" yaml:"from"`
	To   string      `json:"to" yaml:"to"`
	Loop *LoopConfig `json:"loop,omitempty" yaml:"loop,omitempty"`
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
