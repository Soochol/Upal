package upal

import "time"

// --- ContentSession ---

// ContentSessionStatus represents the lifecycle of a content collection session.
type ContentSessionStatus string

const (
	SessionCollecting    ContentSessionStatus = "collecting"
	SessionAnalyzing     ContentSessionStatus = "analyzing"
	SessionPendingReview ContentSessionStatus = "pending_review"
	SessionApproved      ContentSessionStatus = "approved"
	SessionRejected      ContentSessionStatus = "rejected"
	SessionProducing     ContentSessionStatus = "producing"
	SessionPublished     ContentSessionStatus = "published"
	SessionError         ContentSessionStatus = "error"
)

// ContentSession tracks a single collection + analysis cycle for a Pipeline.
type ContentSession struct {
	ID          string               `json:"id"`
	PipelineID  string               `json:"pipeline_id"`
	Status      ContentSessionStatus `json:"status"`
	TriggerType string               `json:"trigger_type"` // "schedule" | "manual" | "surge"
	SourceCount int                  `json:"source_count"` // total raw items collected
	CreatedAt   time.Time            `json:"created_at"`
	ReviewedAt  *time.Time           `json:"reviewed_at,omitempty"`
	ArchivedAt  *time.Time           `json:"archived_at,omitempty"`
}

// --- SourceFetch ---

// SourceItem is a single piece of content collected from a source.
type SourceItem struct {
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	Content     string `json:"content,omitempty"` // body or excerpt
	Score       int    `json:"score,omitempty"`   // HN points, upvotes, search volume, etc.
	SignalType  string `json:"signal_type,omitempty"` // "upvotes" | "search_volume" | "article_count" | ...
	FetchedFrom string `json:"fetched_from,omitempty"`
}

// SourceFetch records the raw result of one source tool invocation in a session.
type SourceFetch struct {
	ID         string       `json:"id"`
	SessionID  string       `json:"session_id"`
	ToolName   string       `json:"tool"`                  // frontend expects "tool"
	SourceType string       `json:"source_type"`           // "static" | "signal"
	Label      string       `json:"label,omitempty"`       // display name for UI
	Count      int          `json:"count"`                 // number of items
	RawItems   []SourceItem `json:"items,omitempty"`       // frontend expects "items"
	Error      *string      `json:"error,omitempty"`       // non-nil means this source failed
	FetchedAt  time.Time    `json:"fetched_at"`
}

// --- LLMAnalysis ---

// ContentAngle is a suggested content angle from the LLM analysis.
type ContentAngle struct {
	ID        string `json:"id,omitempty"`
	Format    string `json:"format"`    // "blog" | "shorts" | "newsletter" | "longform"
	Headline  string `json:"title"`     // frontend expects "title"
	Rationale string `json:"rationale,omitempty"`
	Selected  bool   `json:"selected"`
}

// LLMAnalysis stores the synthesized result of the LLM's analysis step.
type LLMAnalysis struct {
	ID              string         `json:"id"`
	SessionID       string         `json:"session_id"`
	RawItemCount    int            `json:"total_collected"`  // frontend expects "total_collected"
	FilteredCount   int            `json:"total_selected"`   // frontend expects "total_selected"
	Summary         string         `json:"summary"`
	Insights        []string       `json:"insights,omitempty"`
	SuggestedAngles []ContentAngle `json:"angles,omitempty"` // frontend expects "angles"
	OverallScore    int            `json:"score"`            // frontend expects "score"
	CreatedAt       time.Time      `json:"created_at"`
}

// --- PublishedContent ---

// PublishedContent records a single piece of content published to an external channel.
type PublishedContent struct {
	ID            string    `json:"id"`
	WorkflowRunID string    `json:"workflow_run_id"`
	SessionID     string    `json:"session_id"` // reverse lookup
	Channel       string    `json:"channel"`    // "youtube" | "substack" | "discord"
	ExternalURL   string    `json:"external_url,omitempty"`
	Title         string    `json:"title,omitempty"`
	PublishedAt   time.Time `json:"published_at"`
}

// --- SurgeEvent ---

// SurgeEvent records a detected spike in keyword mentions (notify-only; no auto-session).
type SurgeEvent struct {
	ID         string    `json:"id"`
	Keyword    string    `json:"keyword"`
	PipelineID string    `json:"pipeline_id,omitempty"` // which pipeline owns this keyword
	Multiplier float64   `json:"multiplier"`            // observed N× vs baseline
	Sources    []string  `json:"sources,omitempty"`     // which sources detected the surge
	Dismissed  bool      `json:"dismissed"`
	SessionID  *string   `json:"session_id,omitempty"` // set when user creates a session from surge
	CreatedAt  time.Time `json:"created_at"`
}

// --- WorkflowResult ---

// WorkflowResult tracks workflow execution results for the produce stage.
type WorkflowResult struct {
	WorkflowName string     `json:"workflow_name"`
	RunID        string     `json:"run_id"`
	Status       string     `json:"status"` // "pending" | "running" | "success" | "failed"
	OutputURL    string     `json:"output_url,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// --- ContentSessionDetail ---

// ContentSessionDetail is a composed response type that includes all related
// data inline for the frontend GET endpoint. It is NOT an embedded struct —
// field names are kept flat to match the frontend's expected JSON shape.
type ContentSessionDetail struct {
	ID              string               `json:"id"`
	PipelineID      string               `json:"pipeline_id"`
	PipelineName    string               `json:"pipeline_name,omitempty"`
	SessionNumber   int                  `json:"session_number,omitempty"`
	Status          ContentSessionStatus `json:"status"`
	TriggerType     string               `json:"trigger_type"`
	SourceCount     int                  `json:"source_count"`
	Sources         []*SourceFetch       `json:"sources,omitempty"`
	Analysis        *LLMAnalysis         `json:"analysis,omitempty"`
	WorkflowResults []WorkflowResult     `json:"workflow_results,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	ReviewedAt      *time.Time           `json:"reviewed_at,omitempty"`
	ArchivedAt      *time.Time           `json:"archived_at,omitempty"`
}
