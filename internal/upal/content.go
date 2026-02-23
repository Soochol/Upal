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
	ToolName   string       `json:"tool_name"`
	SourceType string       `json:"source_type"` // "static" | "signal"
	RawItems   []SourceItem `json:"raw_items,omitempty"`
	Error      *string      `json:"error,omitempty"` // non-nil means this source failed
	FetchedAt  time.Time    `json:"fetched_at"`
}

// --- LLMAnalysis ---

// ContentAngle is a suggested content angle from the LLM analysis.
type ContentAngle struct {
	Format    string `json:"format"`    // "blog" | "shorts" | "newsletter" | "longform"
	Headline  string `json:"headline"`
	Rationale string `json:"rationale,omitempty"`
}

// LLMAnalysis stores the synthesized result of the LLM's analysis step.
type LLMAnalysis struct {
	ID              string         `json:"id"`
	SessionID       string         `json:"session_id"`
	RawItemCount    int            `json:"raw_item_count"`
	FilteredCount   int            `json:"filtered_count"`
	Summary         string         `json:"summary"`
	Insights        []string       `json:"insights,omitempty"`
	SuggestedAngles []ContentAngle `json:"suggested_angles,omitempty"`
	OverallScore    int            `json:"overall_score"` // 0–100
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
