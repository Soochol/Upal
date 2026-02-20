package upal

import "time"

// --- Run Status ---

// RunStatus represents the lifecycle state of a workflow run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusSuccess   RunStatus = "success"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
	RunStatusRetrying  RunStatus = "retrying"
)

// RunRecord captures a single workflow execution with full provenance.
type RunRecord struct {
	ID           string         `json:"id"`
	WorkflowName string         `json:"workflow_name"`
	TriggerType  string         `json:"trigger_type"`  // "manual" | "cron" | "webhook"
	TriggerRef   string         `json:"trigger_ref"`   // schedule ID or trigger ID
	Status       RunStatus      `json:"status"`
	Inputs       map[string]any `json:"inputs"`
	Outputs      map[string]any `json:"outputs,omitempty"`
	Error        *string        `json:"error,omitempty"`
	RetryOf      *string        `json:"retry_of,omitempty"` // original run ID if this is a retry
	RetryCount   int            `json:"retry_count"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	NodeRuns     []NodeRunRecord `json:"node_runs,omitempty"`
}

// NodeRunRecord tracks execution of a single node within a run.
type NodeRunRecord struct {
	NodeID      string     `json:"node_id"`
	Status      string     `json:"status"` // "running" | "completed" | "error"
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       *string    `json:"error,omitempty"`
	RetryCount  int        `json:"retry_count"`
}

// --- Retry Policy ---

// RetryPolicy defines how failed runs should be retried.
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"    yaml:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"  yaml:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"      yaml:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor" yaml:"backoff_factor"`
}

// DefaultRetryPolicy returns a sensible default retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      5 * time.Minute,
		BackoffFactor: 2.0,
	}
}

// --- Concurrency ---

// ConcurrencyLimits controls how many workflows can execute simultaneously.
type ConcurrencyLimits struct {
	GlobalMax   int `json:"global_max"   yaml:"global_max"`
	PerWorkflow int `json:"per_workflow" yaml:"per_workflow"`
}

// DefaultConcurrencyLimits returns sensible defaults.
func DefaultConcurrencyLimits() ConcurrencyLimits {
	return ConcurrencyLimits{
		GlobalMax:   10,
		PerWorkflow: 3,
	}
}

// --- Schedule ---

// Schedule defines a cron-based recurring workflow execution.
type Schedule struct {
	ID           string         `json:"id"`
	WorkflowName string         `json:"workflow_name"`
	CronExpr     string         `json:"cron_expr"`
	Inputs       map[string]any `json:"inputs,omitempty"`
	Enabled      bool           `json:"enabled"`
	Timezone     string         `json:"timezone"`
	RetryPolicy  *RetryPolicy   `json:"retry_policy,omitempty"`
	NextRunAt    time.Time      `json:"next_run_at"`
	LastRunAt    *time.Time     `json:"last_run_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// --- Trigger ---

// TriggerType identifies how a workflow execution was initiated.
type TriggerType string

const (
	TriggerManual  TriggerType = "manual"
	TriggerCron    TriggerType = "cron"
	TriggerWebhook TriggerType = "webhook"
)

// Trigger defines an event-based workflow execution rule.
type Trigger struct {
	ID           string        `json:"id"`
	WorkflowName string        `json:"workflow_name"`
	Type         TriggerType   `json:"type"`
	Config       TriggerConfig `json:"config"`
	Enabled      bool          `json:"enabled"`
	CreatedAt    time.Time     `json:"created_at"`
}

// TriggerConfig holds type-specific trigger configuration.
type TriggerConfig struct {
	Secret       string            `json:"secret,omitempty"`
	InputMapping map[string]string `json:"input_mapping,omitempty"` // JSONPath → input key
}
