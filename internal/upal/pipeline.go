package upal

import "time"

// Pipeline orchestrates a sequence of Stages (workflows, approvals, schedules).
type Pipeline struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Stages       []Stage   `json:"stages"`
	ThumbnailSVG string    `json:"thumbnail_svg,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Stage is a single step in a Pipeline.
type Stage struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type"` // "workflow", "approval", "notification", "schedule", "trigger", "transform", "collect"
	Config      StageConfig `json:"config"`
	DependsOn   []string    `json:"depends_on,omitempty"`
}

// CollectSource defines a single data source for a collect stage.
type CollectSource struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`                  // "rss" | "http" | "scrape"
	URL         string            `json:"url"`
	Limit       int               `json:"limit,omitempty"`        // RSS: max items (default 20)
	Method      string            `json:"method,omitempty"`       // HTTP: GET/POST (default GET)
	Headers     map[string]string `json:"headers,omitempty"`      // HTTP: request headers
	Body        string            `json:"body,omitempty"`         // HTTP: request body
	Selector    string            `json:"selector,omitempty"`     // Scrape: CSS selector
	Attribute   string            `json:"attribute,omitempty"`    // Scrape: attr to extract (default: text)
	ScrapeLimit int               `json:"scrape_limit,omitempty"` // Scrape: max elements (default 30)
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
	Cron       string `json:"cron,omitempty"`
	Timezone   string `json:"timezone,omitempty"`
	ScheduleID string `json:"schedule_id,omitempty"`

	// Notification stage (also shared with Approval for connection_id + message)
	Subject string `json:"subject,omitempty"` // optional email subject override

	// Trigger stage
	TriggerID string `json:"trigger_id,omitempty"`

	// Transform stage
	Expression string `json:"expression,omitempty"`

	// Collect stage
	Sources []CollectSource `json:"sources,omitempty"`
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
