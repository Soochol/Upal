package upal

import "time"

// SessionStatus represents the lifecycle of a Session.
type SessionStatus string

const (
	SessionStatusDraft    SessionStatus = "draft"
	SessionStatusActive   SessionStatus = "active"
	SessionStatusArchived SessionStatus = "archived"
)

// Session is the top-level container that replaces Pipeline.
// It owns sources, schedules, workflows, stages, and spawns Runs.
type Session struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Sources         []SessionSource   `json:"sources,omitempty"`
	Schedule        string            `json:"schedule,omitempty"`
	Model           string            `json:"model,omitempty"`
	Workflows       []SessionWorkflow `json:"workflows,omitempty"`
	Context         *SessionContext   `json:"context,omitempty"`
	Stages          []Stage           `json:"stages,omitempty"`
	Status          SessionStatus     `json:"status"`
	ThumbnailSVG    string            `json:"thumbnail_svg,omitempty"`
	PendingRunCount int               `json:"pending_run_count,omitempty"`
	LastCollectedAt *time.Time        `json:"last_collected_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// NOTE: SessionSettings is defined in content.go using the old Pipeline* types.
// It will be migrated to use Session* types (SessionSource, SessionWorkflow,
// SessionContext) when the old types are removed in a later task.

// Stage is a single step in a Session (copied from Pipeline — both coexist temporarily).
// type Stage struct { ... }  // defined in pipeline.go

// StageConfig holds type-specific configuration for a Stage.
// type StageConfig struct { ... }  // defined in pipeline.go

// CollectSource defines a single data source for a collect stage.
// type CollectSource struct { ... }  // defined in pipeline.go

// SessionContext carries session-level context injected into all child layers.
// JSON tags match the original PipelineContext for frontend compatibility.
type SessionContext struct {
	Prompt        string `json:"prompt,omitempty"`
	Language      string `json:"language,omitempty"`       // "ko" | "en"
	ResearchDepth string `json:"research_depth,omitempty"` // "light" | "deep" (default: "deep")
	ResearchModel string `json:"research_model,omitempty"` // model ID for research LLM
}

// SessionSource defines a single data source attached to a Session.
// JSON tags match the original PipelineSource for frontend compatibility.
type SessionSource struct {
	ID         string         `json:"id"`
	PipelineID string         `json:"pipeline_id,omitempty"` // kept for backward compat; will be renamed later
	ToolName   string         `json:"tool_name,omitempty"`   // "hn_fetch" | "reddit_fetch" | "rss_feed" | ...
	SourceType string         `json:"source_type"`           // "static" | "signal" | "research"
	Config     map[string]any `json:"config,omitempty"`      // tool-specific params
	Enabled    bool           `json:"enabled,omitempty"`
	// Frontend-compatible flat fields (stored as-is from the UI)
	Type      string   `json:"type,omitempty"`      // "rss" | "hn" | "reddit" | "google_trends" | "social" | "http" | "research"
	Label     string   `json:"label,omitempty"`     // display name
	URL       string   `json:"url,omitempty"`       // rss, http
	Subreddit string   `json:"subreddit,omitempty"` // reddit
	MinScore  int      `json:"min_score,omitempty"` // reddit, hn
	Keywords  []string `json:"keywords,omitempty"`  // google_trends, social
	Accounts  []string `json:"accounts,omitempty"`  // social: follow account handles
	Geo       string   `json:"geo,omitempty"`       // google_trends: country code
	Limit     int      `json:"limit,omitempty"`
	// Research source fields
	Topic string `json:"topic,omitempty"` // research: subject to investigate
	Depth string `json:"depth,omitempty"` // research: "light" | "deep"
	Model string `json:"model,omitempty"` // research: LLM model ID ("provider/model")
}

// SessionWorkflow links an existing workflow to a Session for content production.
// JSON tags match the original PipelineWorkflow for frontend compatibility.
type SessionWorkflow struct {
	WorkflowName string `json:"workflow_name"`
	Label        string `json:"label,omitempty"`
	AutoSelect   bool   `json:"auto_select,omitempty"`
	ChannelID    string `json:"channel_id,omitempty"`
}
