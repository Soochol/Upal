package upal

import "time"

// Pipeline orchestrates a sequence of Stages (workflows, approvals, schedules).
// Settings (sources, schedule, model, workflows, context) live on ContentSession.
type Pipeline struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description,omitempty"`
	Stages              []Stage    `json:"stages"`
	ThumbnailSVG        string     `json:"thumbnail_svg,omitempty"`
	LastCollectedAt     *time.Time `json:"last_collected_at,omitempty"`
	PendingSessionCount int        `json:"pending_session_count,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
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
	Keywords    []string          `json:"keywords,omitempty"`     // Social: search keywords
	Accounts    []string          `json:"accounts,omitempty"`     // Social: follow account handles
	Geo         string            `json:"geo,omitempty"`          // Google Trends: country code
	// Research source fields
	Topic       string `json:"topic,omitempty"`        // Research: topic to investigate
	Model       string `json:"model,omitempty"`        // Research: LLM model ID ("provider/model")
	Depth       string `json:"depth,omitempty"`        // Research: "light" or "deep"
	MaxSearches int    `json:"max_searches,omitempty"` // Research: max search iterations for deep mode
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

// PipelineContext carries editorial brief injected into all child layers.
type PipelineContext struct {
	Purpose         string   `json:"purpose,omitempty"`
	TargetAudience  string   `json:"target_audience,omitempty"`
	ToneStyle       string   `json:"tone_style,omitempty"`
	FocusKeywords   []string `json:"focus_keywords,omitempty"`
	ExcludeKeywords []string `json:"exclude_keywords,omitempty"`
	ContentGoals    string   `json:"content_goals,omitempty"`
	Language        string   `json:"language,omitempty"` // "ko" | "en"
}

// PipelineSource defines a single data source attached to a Pipeline.
// Fields are stored as JSONB so the struct is intentionally loose to accommodate both
// the internal tool-centric format (ToolName + Config) and the frontend flat format.
type PipelineSource struct {
	ID         string         `json:"id"`
	PipelineID string         `json:"pipeline_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`   // "hn_fetch" | "reddit_fetch" | "rss_feed" | ...
	SourceType string         `json:"source_type"`           // "static" | "signal"
	Config     map[string]any `json:"config,omitempty"`      // tool-specific params
	Enabled    bool           `json:"enabled,omitempty"`
	// Frontend-compatible flat fields (stored as-is from the UI)
	Type      string   `json:"type,omitempty"`      // "rss" | "hn" | "reddit" | "google_trends" | "social" | "http"
	Label     string   `json:"label,omitempty"`     // display name
	URL       string   `json:"url,omitempty"`       // rss, http
	Subreddit string   `json:"subreddit,omitempty"` // reddit
	MinScore  int      `json:"min_score,omitempty"` // reddit, hn
	Keywords  []string `json:"keywords,omitempty"`  // google_trends, social
	Accounts  []string `json:"accounts,omitempty"` // social: follow account handles
	Geo       string   `json:"geo,omitempty"`      // google_trends: country code
	Limit     int      `json:"limit,omitempty"`
}

// PipelineWorkflow links an existing workflow to a pipeline for content production.
type PipelineWorkflow struct {
	WorkflowName string `json:"workflow_name"`
	Label        string `json:"label,omitempty"`
	AutoSelect   bool   `json:"auto_select,omitempty"`
	ChannelID    string `json:"channel_id,omitempty"`
}

// PipelineRunStatus represents the lifecycle state of a pipeline run.
type PipelineRunStatus string

const (
	PipelineRunPending   PipelineRunStatus = "pending"
	PipelineRunRunning   PipelineRunStatus = "running"
	PipelineRunWaiting   PipelineRunStatus = "waiting"
	PipelineRunCompleted PipelineRunStatus = "completed"
	PipelineRunFailed    PipelineRunStatus = "failed"
	PipelineRunRejected  PipelineRunStatus = "rejected"
)

// StageStatus represents the lifecycle state of a pipeline stage execution.
type StageStatus string

const (
	StageStatusPending   StageStatus = "pending"
	StageStatusRunning   StageStatus = "running"
	StageStatusWaiting   StageStatus = "waiting"
	StageStatusCompleted StageStatus = "completed"
	StageStatusFailed    StageStatus = "failed"
	StageStatusSkipped   StageStatus = "skipped"
)

// PipelineRun tracks a single execution of a Pipeline.
type PipelineRun struct {
	ID           string                  `json:"id"`
	PipelineID   string                  `json:"pipeline_id"`
	Status       PipelineRunStatus       `json:"status"`
	CurrentStage string                  `json:"current_stage,omitempty"`
	StageResults map[string]*StageResult `json:"stage_results,omitempty"`
	StartedAt    time.Time               `json:"started_at"`
	CompletedAt  *time.Time              `json:"completed_at,omitempty"`
}

// StageResult is the output of a completed Stage.
type StageResult struct {
	StageID     string      `json:"stage_id"`
	Status      StageStatus `json:"status"`
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}
