# Content Media Pipeline — Phase 1 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Lay the data foundation for the content media pipeline — domain types, DB schema, repository layer, service layer, and API endpoints for ContentSession, SourceFetch, LLMAnalysis, PublishedContent, and SurgeEvent.

**Architecture:** Extend the existing Pipeline struct with `Context` and `Sources` fields, add new content media types in `internal/upal/content.go`, wire them through the standard Upal layers (db → repository → service → api → main), and expose CRUD + review endpoints. No actual source collection or workflow execution in this phase — just the data scaffolding.

**Tech Stack:** Go 1.23, Chi router, PostgreSQL (optional, graceful in-memory fallback), standard `database/sql`, `encoding/json`.

**Run tests with:** `go test ./... -v -race`
**Build check:** `go build ./...`

---

## Task 1: Domain types — extend Pipeline struct

**Goal:** Add `PipelineContext` and `PipelineSource` types, embed them in `Pipeline`.

**Files:**
- Modify: `internal/upal/pipeline.go`

**Step 1: Verify current Pipeline struct compiles**

```bash
go build ./internal/upal/...
```
Expected: PASS (baseline check).

**Step 2: Add types to pipeline.go**

Add after the `StageConfig` struct (around line 67), before `PipelineRun`:

```go
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
type PipelineSource struct {
	ID         string         `json:"id"`
	PipelineID string         `json:"pipeline_id"`
	ToolName   string         `json:"tool_name"` // "hn_fetch" | "reddit_fetch" | "rss_feed" | ...
	SourceType string         `json:"source_type"` // "static" | "signal"
	Config     map[string]any `json:"config,omitempty"` // tool-specific params
	Enabled    bool           `json:"enabled"`
}
```

Update the `Pipeline` struct to include new fields:

```go
type Pipeline struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Stages       []Stage         `json:"stages"`
	Context      PipelineContext `json:"context,omitempty"`
	Sources      []PipelineSource `json:"sources,omitempty"`
	ThumbnailSVG string          `json:"thumbnail_svg,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}
```

**Step 3: Verify compilation**

```bash
go build ./internal/upal/...
```
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/upal/pipeline.go
git commit -m "feat(content): add PipelineContext and PipelineSource types to Pipeline"
```

---

## Task 2: Domain types — content media types

**Goal:** Define `ContentSession`, `SourceFetch`, `LLMAnalysis`, `PublishedContent`, `SurgeEvent` in a new file.

**Files:**
- Create: `internal/upal/content.go`

**Step 1: Create the file**

```go
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
```

**Step 2: Verify compilation**

```bash
go build ./internal/upal/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/upal/content.go
git commit -m "feat(content): add ContentSession, SourceFetch, LLMAnalysis, PublishedContent, SurgeEvent types"
```

---

## Task 3: Extend RunRecord with SessionID

**Goal:** Add `SessionID *string` to `RunRecord` so workflow runs can be traced back to the content session that triggered them.

**Files:**
- Modify: `internal/upal/scheduler.go`

**Step 1: Add field to RunRecord struct**

In `internal/upal/scheduler.go`, add `SessionID` after `RetryCount`:

```go
// RunRecord captures a single workflow execution with full provenance.
type RunRecord struct {
	ID           string          `json:"id"`
	WorkflowName string          `json:"workflow_name"`
	TriggerType  string          `json:"trigger_type"`
	TriggerRef   string          `json:"trigger_ref"`
	Status       RunStatus       `json:"status"`
	Inputs       map[string]any  `json:"inputs"`
	Outputs      map[string]any  `json:"outputs,omitempty"`
	Error        *string         `json:"error,omitempty"`
	RetryOf      *string         `json:"retry_of,omitempty"`
	RetryCount   int             `json:"retry_count"`
	SessionID    *string         `json:"session_id,omitempty"` // set when run was triggered from a ContentSession
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	NodeRuns     []NodeRunRecord `json:"node_runs,omitempty"`
	Usage        *TokenUsage     `json:"usage,omitempty"`
}
```

**Step 2: Verify compilation**

```bash
go build ./internal/upal/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/upal/scheduler.go
git commit -m "feat(content): add SessionID field to RunRecord"
```

---

## Task 4: DB schema — pipeline extension and content tables

**Goal:** Add `context` and `sources` columns to `pipelines`, add `session_id` to `runs`, and create the four new content tables.

**Files:**
- Modify: `internal/db/db.go`

**Step 1: Add ALTER TABLEs and new tables to migrationSQL**

Append the following to the `migrationSQL` constant in `internal/db/db.go`, just before the closing backtick:

```sql
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS context JSONB NOT NULL DEFAULT '{}';
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';
ALTER TABLE runs ADD COLUMN IF NOT EXISTS session_id TEXT;

CREATE TABLE IF NOT EXISTS content_sessions (
    id           TEXT PRIMARY KEY,
    pipeline_id  TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'collecting',
    trigger_type TEXT NOT NULL DEFAULT 'manual',
    source_count INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_content_sessions_pipeline_id ON content_sessions(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_content_sessions_status ON content_sessions(status);

CREATE TABLE IF NOT EXISTS source_fetches (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES content_sessions(id) ON DELETE CASCADE,
    tool_name   TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'static',
    raw_items   JSONB NOT NULL DEFAULT '[]',
    error       TEXT,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_source_fetches_session_id ON source_fetches(session_id);

CREATE TABLE IF NOT EXISTS llm_analyses (
    id               TEXT PRIMARY KEY,
    session_id       TEXT NOT NULL REFERENCES content_sessions(id) ON DELETE CASCADE,
    raw_item_count   INTEGER NOT NULL DEFAULT 0,
    filtered_count   INTEGER NOT NULL DEFAULT 0,
    summary          TEXT NOT NULL DEFAULT '',
    insights         JSONB NOT NULL DEFAULT '[]',
    suggested_angles JSONB NOT NULL DEFAULT '[]',
    overall_score    INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_llm_analyses_session_id ON llm_analyses(session_id);

CREATE TABLE IF NOT EXISTS published_content (
    id               TEXT PRIMARY KEY,
    workflow_run_id  TEXT NOT NULL,
    session_id       TEXT NOT NULL,
    channel          TEXT NOT NULL,
    external_url     TEXT NOT NULL DEFAULT '',
    title            TEXT NOT NULL DEFAULT '',
    published_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_published_content_session_id ON published_content(session_id);
CREATE INDEX IF NOT EXISTS idx_published_content_channel ON published_content(channel);

CREATE TABLE IF NOT EXISTS surge_events (
    id          TEXT PRIMARY KEY,
    keyword     TEXT NOT NULL,
    pipeline_id TEXT NOT NULL DEFAULT '',
    multiplier  DOUBLE PRECISION NOT NULL DEFAULT 0,
    sources     JSONB NOT NULL DEFAULT '[]',
    dismissed   BOOLEAN NOT NULL DEFAULT false,
    session_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_surge_events_dismissed ON surge_events(dismissed);
```

**Step 2: Verify compilation**

```bash
go build ./internal/db/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat(content): add DB schema for content_sessions, source_fetches, llm_analyses, published_content, surge_events"
```

---

## Task 5: Update DB pipeline CRUD for context and sources

**Goal:** Update `internal/db/pipeline.go` so all pipeline read/write operations include the new `context` and `sources` columns.

**Files:**
- Modify: `internal/db/pipeline.go`

**Step 1: Update CreatePipeline**

Replace the existing `CreatePipeline` function:

```go
func (d *DB) CreatePipeline(ctx context.Context, p *upal.Pipeline) error {
	stagesJSON, err := json.Marshal(p.Stages)
	if err != nil {
		return fmt.Errorf("marshal stages: %w", err)
	}
	ctxJSON, err := json.Marshal(p.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	sourcesJSON, err := json.Marshal(p.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO pipelines (id, name, description, stages, context, sources, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.Name, p.Description, stagesJSON, ctxJSON, sourcesJSON, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline: %w", err)
	}
	return nil
}
```

**Step 2: Update GetPipeline**

Replace the existing `GetPipeline` function:

```go
func (d *DB) GetPipeline(ctx context.Context, id string) (*upal.Pipeline, error) {
	var p upal.Pipeline
	var stagesJSON, ctxJSON, sourcesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, description, stages, context, sources, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &ctxJSON, &sourcesJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}
	if err := json.Unmarshal(stagesJSON, &p.Stages); err != nil {
		return nil, fmt.Errorf("unmarshal stages: %w", err)
	}
	if err := json.Unmarshal(ctxJSON, &p.Context); err != nil {
		return nil, fmt.Errorf("unmarshal context: %w", err)
	}
	if err := json.Unmarshal(sourcesJSON, &p.Sources); err != nil {
		return nil, fmt.Errorf("unmarshal sources: %w", err)
	}
	return &p, nil
}
```

**Step 3: Update ListPipelines**

Replace the existing `ListPipelines` function:

```go
func (d *DB) ListPipelines(ctx context.Context) ([]*upal.Pipeline, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, description, stages, context, sources, created_at, updated_at
		 FROM pipelines ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	defer rows.Close()

	var result []*upal.Pipeline
	for rows.Next() {
		var p upal.Pipeline
		var stagesJSON, ctxJSON, sourcesJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &ctxJSON, &sourcesJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		if err := json.Unmarshal(stagesJSON, &p.Stages); err != nil {
			return nil, fmt.Errorf("unmarshal stages: %w", err)
		}
		if err := json.Unmarshal(ctxJSON, &p.Context); err != nil {
			return nil, fmt.Errorf("unmarshal context: %w", err)
		}
		if err := json.Unmarshal(sourcesJSON, &p.Sources); err != nil {
			return nil, fmt.Errorf("unmarshal sources: %w", err)
		}
		result = append(result, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pipelines: %w", err)
	}
	return result, nil
}
```

**Step 4: Update UpdatePipeline**

Replace the existing `UpdatePipeline` function:

```go
func (d *DB) UpdatePipeline(ctx context.Context, p *upal.Pipeline) error {
	stagesJSON, err := json.Marshal(p.Stages)
	if err != nil {
		return fmt.Errorf("marshal stages: %w", err)
	}
	ctxJSON, err := json.Marshal(p.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	sourcesJSON, err := json.Marshal(p.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE pipelines SET name = $1, description = $2, stages = $3, context = $4, sources = $5, updated_at = $6
		 WHERE id = $7`,
		p.Name, p.Description, stagesJSON, ctxJSON, sourcesJSON, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update pipeline: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline %q not found", p.ID)
	}
	return nil
}
```

**Step 5: Verify compilation**

```bash
go build ./internal/db/...
```
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/db/pipeline.go
git commit -m "feat(content): update pipeline DB CRUD to include context and sources columns"
```

---

## Task 6: DB CRUD for content types

**Goal:** Implement DB methods for ContentSession, SourceFetch, LLMAnalysis, PublishedContent, and SurgeEvent.

**Files:**
- Create: `internal/db/content.go`

**Step 1: Create the file**

```go
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// --- ContentSession ---

func (d *DB) CreateContentSession(ctx context.Context, s *upal.ContentSession) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO content_sessions (id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.PipelineID, string(s.Status), s.TriggerType, s.SourceCount, s.CreatedAt, s.ReviewedAt,
	)
	if err != nil {
		return fmt.Errorf("insert content_session: %w", err)
	}
	return nil
}

func (d *DB) GetContentSession(ctx context.Context, id string) (*upal.ContentSession, error) {
	var s upal.ContentSession
	var status string
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("content session %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get content_session: %w", err)
	}
	s.Status = upal.ContentSessionStatus(status)
	return &s, nil
}

func (d *DB) ListContentSessions(ctx context.Context) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) ListContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at
		 FROM content_sessions WHERE pipeline_id = $1 ORDER BY created_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list content_sessions by pipeline: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) UpdateContentSession(ctx context.Context, s *upal.ContentSession) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE content_sessions SET status = $1, source_count = $2, reviewed_at = $3 WHERE id = $4`,
		string(s.Status), s.SourceCount, s.ReviewedAt, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update content_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("content session %q not found", s.ID)
	}
	return nil
}

// --- SourceFetch ---

func (d *DB) CreateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	itemsJSON, err := json.Marshal(sf.RawItems)
	if err != nil {
		return fmt.Errorf("marshal raw_items: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO source_fetches (id, session_id, tool_name, source_type, raw_items, error, fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sf.ID, sf.SessionID, sf.ToolName, sf.SourceType, itemsJSON, sf.Error, sf.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("insert source_fetch: %w", err)
	}
	return nil
}

func (d *DB) ListSourceFetchesBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, session_id, tool_name, source_type, raw_items, error, fetched_at
		 FROM source_fetches WHERE session_id = $1 ORDER BY fetched_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list source_fetches: %w", err)
	}
	defer rows.Close()
	var result []*upal.SourceFetch
	for rows.Next() {
		var sf upal.SourceFetch
		var itemsJSON []byte
		if err := rows.Scan(&sf.ID, &sf.SessionID, &sf.ToolName, &sf.SourceType, &itemsJSON, &sf.Error, &sf.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan source_fetch: %w", err)
		}
		if err := json.Unmarshal(itemsJSON, &sf.RawItems); err != nil {
			return nil, fmt.Errorf("unmarshal raw_items: %w", err)
		}
		result = append(result, &sf)
	}
	return result, rows.Err()
}

// --- LLMAnalysis ---

func (d *DB) CreateLLMAnalysis(ctx context.Context, a *upal.LLMAnalysis) error {
	insightsJSON, err := json.Marshal(a.Insights)
	if err != nil {
		return fmt.Errorf("marshal insights: %w", err)
	}
	anglesJSON, err := json.Marshal(a.SuggestedAngles)
	if err != nil {
		return fmt.Errorf("marshal suggested_angles: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO llm_analyses (id, session_id, raw_item_count, filtered_count, summary, insights, suggested_angles, overall_score, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.SessionID, a.RawItemCount, a.FilteredCount, a.Summary,
		insightsJSON, anglesJSON, a.OverallScore, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert llm_analysis: %w", err)
	}
	return nil
}

func (d *DB) GetLLMAnalysisBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	var a upal.LLMAnalysis
	var insightsJSON, anglesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, session_id, raw_item_count, filtered_count, summary, insights, suggested_angles, overall_score, created_at
		 FROM llm_analyses WHERE session_id = $1 ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	).Scan(&a.ID, &a.SessionID, &a.RawItemCount, &a.FilteredCount, &a.Summary,
		&insightsJSON, &anglesJSON, &a.OverallScore, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("llm analysis for session %q not found", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("get llm_analysis: %w", err)
	}
	if err := json.Unmarshal(insightsJSON, &a.Insights); err != nil {
		return nil, fmt.Errorf("unmarshal insights: %w", err)
	}
	if err := json.Unmarshal(anglesJSON, &a.SuggestedAngles); err != nil {
		return nil, fmt.Errorf("unmarshal suggested_angles: %w", err)
	}
	return &a, nil
}

// --- PublishedContent ---

func (d *DB) CreatePublishedContent(ctx context.Context, pc *upal.PublishedContent) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO published_content (id, workflow_run_id, session_id, channel, external_url, title, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		pc.ID, pc.WorkflowRunID, pc.SessionID, pc.Channel, pc.ExternalURL, pc.Title, pc.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("insert published_content: %w", err)
	}
	return nil
}

func (d *DB) ListPublishedContent(ctx context.Context) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content ORDER BY published_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func (d *DB) ListPublishedContentBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE session_id = $1 ORDER BY published_at DESC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content by session: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func (d *DB) ListPublishedContentByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, workflow_run_id, session_id, channel, external_url, title, published_at
		 FROM published_content WHERE channel = $1 ORDER BY published_at DESC`,
		channel,
	)
	if err != nil {
		return nil, fmt.Errorf("list published_content by channel: %w", err)
	}
	defer rows.Close()
	return scanPublishedContent(rows)
}

func scanPublishedContent(rows *sql.Rows) ([]*upal.PublishedContent, error) {
	var result []*upal.PublishedContent
	for rows.Next() {
		var pc upal.PublishedContent
		if err := rows.Scan(&pc.ID, &pc.WorkflowRunID, &pc.SessionID, &pc.Channel, &pc.ExternalURL, &pc.Title, &pc.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan published_content: %w", err)
		}
		result = append(result, &pc)
	}
	return result, rows.Err()
}

// --- SurgeEvent ---

func (d *DB) CreateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error {
	sourcesJSON, err := json.Marshal(se.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO surge_events (id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		se.ID, se.Keyword, se.PipelineID, se.Multiplier, sourcesJSON, se.Dismissed, se.SessionID, se.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert surge_event: %w", err)
	}
	return nil
}

func (d *DB) ListSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list surge_events: %w", err)
	}
	defer rows.Close()
	return scanSurgeEvents(rows)
}

func (d *DB) ListActiveSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, keyword, pipeline_id, multiplier, sources, dismissed, session_id, created_at
		 FROM surge_events WHERE dismissed = false ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active surge_events: %w", err)
	}
	defer rows.Close()
	return scanSurgeEvents(rows)
}

func (d *DB) UpdateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE surge_events SET dismissed = $1, session_id = $2 WHERE id = $3`,
		se.Dismissed, se.SessionID, se.ID,
	)
	if err != nil {
		return fmt.Errorf("update surge_event: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("surge event %q not found", se.ID)
	}
	return nil
}

func scanSurgeEvents(rows *sql.Rows) ([]*upal.SurgeEvent, error) {
	var result []*upal.SurgeEvent
	for rows.Next() {
		var se upal.SurgeEvent
		var sourcesJSON []byte
		if err := rows.Scan(&se.ID, &se.Keyword, &se.PipelineID, &se.Multiplier, &sourcesJSON, &se.Dismissed, &se.SessionID, &se.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan surge_event: %w", err)
		}
		if err := json.Unmarshal(sourcesJSON, &se.Sources); err != nil {
			return nil, fmt.Errorf("unmarshal sources: %w", err)
		}
		result = append(result, &se)
	}
	return result, rows.Err()
}
```

**Step 2: Verify compilation**

```bash
go build ./internal/db/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/db/content.go
git commit -m "feat(content): add DB CRUD methods for ContentSession, SourceFetch, LLMAnalysis, PublishedContent, SurgeEvent"
```

---

## Task 7: Repository interface for content types

**Goal:** Define the repository interfaces for content data.

**Files:**
- Create: `internal/repository/content.go`

**Step 1: Create the file**

```go
package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// ContentSessionRepository manages ContentSession persistence.
type ContentSessionRepository interface {
	Create(ctx context.Context, s *upal.ContentSession) error
	Get(ctx context.Context, id string) (*upal.ContentSession, error)
	List(ctx context.Context) ([]*upal.ContentSession, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	Update(ctx context.Context, s *upal.ContentSession) error
}

// SourceFetchRepository manages SourceFetch persistence.
type SourceFetchRepository interface {
	Create(ctx context.Context, sf *upal.SourceFetch) error
	ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)
}

// LLMAnalysisRepository manages LLMAnalysis persistence.
type LLMAnalysisRepository interface {
	Create(ctx context.Context, a *upal.LLMAnalysis) error
	GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
}

// PublishedContentRepository manages PublishedContent persistence.
type PublishedContentRepository interface {
	Create(ctx context.Context, pc *upal.PublishedContent) error
	List(ctx context.Context) ([]*upal.PublishedContent, error)
	ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)
}

// SurgeEventRepository manages SurgeEvent persistence.
type SurgeEventRepository interface {
	Create(ctx context.Context, se *upal.SurgeEvent) error
	List(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActive(ctx context.Context) ([]*upal.SurgeEvent, error)
	Get(ctx context.Context, id string) (*upal.SurgeEvent, error)
	Update(ctx context.Context, se *upal.SurgeEvent) error
}
```

**Step 2: Verify compilation**

```bash
go build ./internal/repository/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/repository/content.go
git commit -m "feat(content): add repository interfaces for content types"
```

---

## Task 8: Memory repositories for content types

**Goal:** Implement all five repository interfaces using the in-memory `Store[T]`.

**Files:**
- Create: `internal/repository/content_memory.go`

**Step 1: Write failing test first**

Create `internal/repository/content_memory_test.go`:

```go
package repository

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/upal"
)

func TestMemoryContentSessionRepo_CRUD(t *testing.T) {
	repo := NewMemoryContentSessionRepository()
	ctx := context.Background()

	s := &upal.ContentSession{
		ID:          "csess-1",
		PipelineID:  "pipe-1",
		Status:      upal.SessionCollecting,
		TriggerType: "manual",
		CreatedAt:   time.Now(),
	}

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, "csess-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != upal.SessionCollecting {
		t.Errorf("expected status collecting, got %q", got.Status)
	}
	list, _ := repo.List(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 session, got %d", len(list))
	}
	byPipeline, _ := repo.ListByPipeline(ctx, "pipe-1")
	if len(byPipeline) != 1 {
		t.Fatalf("expected 1 by pipeline, got %d", len(byPipeline))
	}
	s.Status = upal.SessionPendingReview
	if err := repo.Update(ctx, s); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = repo.Get(ctx, "csess-1")
	if got.Status != upal.SessionPendingReview {
		t.Errorf("expected pending_review, got %q", got.Status)
	}
}

func TestMemorySourceFetchRepo(t *testing.T) {
	repo := NewMemorySourceFetchRepository()
	ctx := context.Background()

	sf := &upal.SourceFetch{
		ID:        "sf-1",
		SessionID: "csess-1",
		ToolName:  "hn_fetch",
		FetchedAt: time.Now(),
		RawItems:  []upal.SourceItem{{Title: "Test Article", Score: 100}},
	}
	if err := repo.Create(ctx, sf); err != nil {
		t.Fatalf("create: %v", err)
	}
	list, _ := repo.ListBySession(ctx, "csess-1")
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].ToolName != "hn_fetch" {
		t.Errorf("unexpected tool name: %q", list[0].ToolName)
	}
}

func TestMemorySurgeEventRepo(t *testing.T) {
	repo := NewMemorySurgeEventRepository()
	ctx := context.Background()

	se := &upal.SurgeEvent{
		ID:        "surge-1",
		Keyword:   "DeepSeek",
		Multiplier: 10.0,
		CreatedAt: time.Now(),
	}
	if err := repo.Create(ctx, se); err != nil {
		t.Fatalf("create: %v", err)
	}
	list, _ := repo.ListActive(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 active, got %d", len(list))
	}
	se.Dismissed = true
	repo.Update(ctx, se)
	active, _ := repo.ListActive(ctx)
	if len(active) != 0 {
		t.Fatalf("expected 0 active after dismiss, got %d", len(active))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/repository/... -run TestMemoryContentSession -v
```
Expected: FAIL — `NewMemoryContentSessionRepository` undefined.

**Step 3: Create the memory implementations**

Create `internal/repository/content_memory.go`:

```go
package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// --- ContentSession ---

type MemoryContentSessionRepository struct {
	store *memstore.Store[*upal.ContentSession]
}

func NewMemoryContentSessionRepository() *MemoryContentSessionRepository {
	return &MemoryContentSessionRepository{
		store: memstore.New(func(s *upal.ContentSession) string { return s.ID }),
	}
}

func (r *MemoryContentSessionRepository) Create(ctx context.Context, s *upal.ContentSession) error {
	if r.store.Has(ctx, s.ID) {
		return fmt.Errorf("content session %q already exists", s.ID)
	}
	return r.store.Set(ctx, s)
}

func (r *MemoryContentSessionRepository) Get(ctx context.Context, id string) (*upal.ContentSession, error) {
	s, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("content session %q not found", id)
	}
	return s, err
}

func (r *MemoryContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	return r.store.All(ctx)
}

func (r *MemoryContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID
	})
}

func (r *MemoryContentSessionRepository) Update(ctx context.Context, s *upal.ContentSession) error {
	if !r.store.Has(ctx, s.ID) {
		return fmt.Errorf("content session %q not found", s.ID)
	}
	return r.store.Set(ctx, s)
}

// --- SourceFetch ---

type MemorySourceFetchRepository struct {
	store *memstore.Store[*upal.SourceFetch]
}

func NewMemorySourceFetchRepository() *MemorySourceFetchRepository {
	return &MemorySourceFetchRepository{
		store: memstore.New(func(sf *upal.SourceFetch) string { return sf.ID }),
	}
}

func (r *MemorySourceFetchRepository) Create(ctx context.Context, sf *upal.SourceFetch) error {
	return r.store.Set(ctx, sf)
}

func (r *MemorySourceFetchRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	return r.store.Filter(ctx, func(sf *upal.SourceFetch) bool {
		return sf.SessionID == sessionID
	})
}

// --- LLMAnalysis ---

type MemoryLLMAnalysisRepository struct {
	store *memstore.Store[*upal.LLMAnalysis]
}

func NewMemoryLLMAnalysisRepository() *MemoryLLMAnalysisRepository {
	return &MemoryLLMAnalysisRepository{
		store: memstore.New(func(a *upal.LLMAnalysis) string { return a.ID }),
	}
}

func (r *MemoryLLMAnalysisRepository) Create(ctx context.Context, a *upal.LLMAnalysis) error {
	return r.store.Set(ctx, a)
}

func (r *MemoryLLMAnalysisRepository) GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	all, err := r.store.Filter(ctx, func(a *upal.LLMAnalysis) bool {
		return a.SessionID == sessionID
	})
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("llm analysis for session %q not found", sessionID)
	}
	// Return the most recent (last created — simple: return last in slice)
	return all[len(all)-1], nil
}

// --- PublishedContent ---

type MemoryPublishedContentRepository struct {
	store *memstore.Store[*upal.PublishedContent]
}

func NewMemoryPublishedContentRepository() *MemoryPublishedContentRepository {
	return &MemoryPublishedContentRepository{
		store: memstore.New(func(pc *upal.PublishedContent) string { return pc.ID }),
	}
}

func (r *MemoryPublishedContentRepository) Create(ctx context.Context, pc *upal.PublishedContent) error {
	return r.store.Set(ctx, pc)
}

func (r *MemoryPublishedContentRepository) List(ctx context.Context) ([]*upal.PublishedContent, error) {
	return r.store.All(ctx)
}

func (r *MemoryPublishedContentRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	return r.store.Filter(ctx, func(pc *upal.PublishedContent) bool {
		return pc.SessionID == sessionID
	})
}

func (r *MemoryPublishedContentRepository) ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	return r.store.Filter(ctx, func(pc *upal.PublishedContent) bool {
		return pc.Channel == channel
	})
}

// --- SurgeEvent ---

type MemorySurgeEventRepository struct {
	store *memstore.Store[*upal.SurgeEvent]
}

func NewMemorySurgeEventRepository() *MemorySurgeEventRepository {
	return &MemorySurgeEventRepository{
		store: memstore.New(func(se *upal.SurgeEvent) string { return se.ID }),
	}
}

func (r *MemorySurgeEventRepository) Create(ctx context.Context, se *upal.SurgeEvent) error {
	return r.store.Set(ctx, se)
}

func (r *MemorySurgeEventRepository) List(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return r.store.All(ctx)
}

func (r *MemorySurgeEventRepository) ListActive(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return r.store.Filter(ctx, func(se *upal.SurgeEvent) bool {
		return !se.Dismissed
	})
}

func (r *MemorySurgeEventRepository) Get(ctx context.Context, id string) (*upal.SurgeEvent, error) {
	se, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("surge event %q not found", id)
	}
	return se, err
}

func (r *MemorySurgeEventRepository) Update(ctx context.Context, se *upal.SurgeEvent) error {
	if !r.store.Has(ctx, se.ID) {
		return fmt.Errorf("surge event %q not found", se.ID)
	}
	return r.store.Set(ctx, se)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/repository/... -run "TestMemoryContentSession|TestMemorySourceFetch|TestMemorySurgeEvent" -v -race
```
Expected: PASS (all three tests).

**Step 5: Commit**

```bash
git add internal/repository/content_memory.go internal/repository/content_memory_test.go
git commit -m "feat(content): add memory repositories for content types (with tests)"
```

---

## Task 9: Persistent repositories for content types

**Goal:** Wrap memory repos with DB backend for production use.

**Files:**
- Create: `internal/repository/content_persistent.go`

**Step 1: Create the file**

```go
package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// ContentDB defines the DB-layer methods needed by the persistent content repos.
// *db.DB satisfies this interface.
type ContentDB interface {
	CreateContentSession(ctx context.Context, s *upal.ContentSession) error
	GetContentSession(ctx context.Context, id string) (*upal.ContentSession, error)
	ListContentSessions(ctx context.Context) ([]*upal.ContentSession, error)
	ListContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	UpdateContentSession(ctx context.Context, s *upal.ContentSession) error
	CreateSourceFetch(ctx context.Context, sf *upal.SourceFetch) error
	ListSourceFetchesBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error)
	CreateLLMAnalysis(ctx context.Context, a *upal.LLMAnalysis) error
	GetLLMAnalysisBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error)
	CreatePublishedContent(ctx context.Context, pc *upal.PublishedContent) error
	ListPublishedContent(ctx context.Context) ([]*upal.PublishedContent, error)
	ListPublishedContentBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListPublishedContentByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)
	CreateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error
	ListSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActiveSurgeEvents(ctx context.Context) ([]*upal.SurgeEvent, error)
	UpdateSurgeEvent(ctx context.Context, se *upal.SurgeEvent) error
}

// PersistentContentSessionRepository wraps MemoryContentSessionRepository with DB backend.
type PersistentContentSessionRepository struct {
	mem *MemoryContentSessionRepository
	db  ContentDB
}

func NewPersistentContentSessionRepository(mem *MemoryContentSessionRepository, db ContentDB) *PersistentContentSessionRepository {
	return &PersistentContentSessionRepository{mem: mem, db: db}
}

func (r *PersistentContentSessionRepository) Create(ctx context.Context, s *upal.ContentSession) error {
	_ = r.mem.Create(ctx, s)
	if err := r.db.CreateContentSession(ctx, s); err != nil {
		return fmt.Errorf("db create content_session: %w", err)
	}
	return nil
}

func (r *PersistentContentSessionRepository) Get(ctx context.Context, id string) (*upal.ContentSession, error) {
	if s, err := r.mem.Get(ctx, id); err == nil {
		return s, nil
	}
	s, err := r.db.GetContentSession(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, s)
	return s, nil
}

func (r *PersistentContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListContentSessions(ctx)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListContentSessionsByPipeline(ctx, pipelineID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list content_sessions by pipeline failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) Update(ctx context.Context, s *upal.ContentSession) error {
	_ = r.mem.Update(ctx, s)
	if err := r.db.UpdateContentSession(ctx, s); err != nil {
		return fmt.Errorf("db update content_session: %w", err)
	}
	return nil
}

// PersistentSourceFetchRepository wraps MemorySourceFetchRepository with DB backend.
type PersistentSourceFetchRepository struct {
	mem *MemorySourceFetchRepository
	db  ContentDB
}

func NewPersistentSourceFetchRepository(mem *MemorySourceFetchRepository, db ContentDB) *PersistentSourceFetchRepository {
	return &PersistentSourceFetchRepository{mem: mem, db: db}
}

func (r *PersistentSourceFetchRepository) Create(ctx context.Context, sf *upal.SourceFetch) error {
	_ = r.mem.Create(ctx, sf)
	if err := r.db.CreateSourceFetch(ctx, sf); err != nil {
		return fmt.Errorf("db create source_fetch: %w", err)
	}
	return nil
}

func (r *PersistentSourceFetchRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	fetches, err := r.db.ListSourceFetchesBySession(ctx, sessionID)
	if err == nil {
		return fetches, nil
	}
	slog.Warn("db list source_fetches failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

// PersistentLLMAnalysisRepository wraps MemoryLLMAnalysisRepository with DB backend.
type PersistentLLMAnalysisRepository struct {
	mem *MemoryLLMAnalysisRepository
	db  ContentDB
}

func NewPersistentLLMAnalysisRepository(mem *MemoryLLMAnalysisRepository, db ContentDB) *PersistentLLMAnalysisRepository {
	return &PersistentLLMAnalysisRepository{mem: mem, db: db}
}

func (r *PersistentLLMAnalysisRepository) Create(ctx context.Context, a *upal.LLMAnalysis) error {
	_ = r.mem.Create(ctx, a)
	if err := r.db.CreateLLMAnalysis(ctx, a); err != nil {
		return fmt.Errorf("db create llm_analysis: %w", err)
	}
	return nil
}

func (r *PersistentLLMAnalysisRepository) GetBySession(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	if a, err := r.mem.GetBySession(ctx, sessionID); err == nil {
		return a, nil
	}
	return r.db.GetLLMAnalysisBySession(ctx, sessionID)
}

// PersistentPublishedContentRepository wraps MemoryPublishedContentRepository with DB backend.
type PersistentPublishedContentRepository struct {
	mem *MemoryPublishedContentRepository
	db  ContentDB
}

func NewPersistentPublishedContentRepository(mem *MemoryPublishedContentRepository, db ContentDB) *PersistentPublishedContentRepository {
	return &PersistentPublishedContentRepository{mem: mem, db: db}
}

func (r *PersistentPublishedContentRepository) Create(ctx context.Context, pc *upal.PublishedContent) error {
	_ = r.mem.Create(ctx, pc)
	if err := r.db.CreatePublishedContent(ctx, pc); err != nil {
		return fmt.Errorf("db create published_content: %w", err)
	}
	return nil
}

func (r *PersistentPublishedContentRepository) List(ctx context.Context) ([]*upal.PublishedContent, error) {
	pcs, err := r.db.ListPublishedContent(ctx)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentPublishedContentRepository) ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	pcs, err := r.db.ListPublishedContentBySession(ctx, sessionID)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content by session failed, falling back to in-memory", "err", err)
	return r.mem.ListBySession(ctx, sessionID)
}

func (r *PersistentPublishedContentRepository) ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	pcs, err := r.db.ListPublishedContentByChannel(ctx, channel)
	if err == nil {
		return pcs, nil
	}
	slog.Warn("db list published_content by channel failed, falling back to in-memory", "err", err)
	return r.mem.ListByChannel(ctx, channel)
}

// PersistentSurgeEventRepository wraps MemorySurgeEventRepository with DB backend.
type PersistentSurgeEventRepository struct {
	mem *MemorySurgeEventRepository
	db  ContentDB
}

func NewPersistentSurgeEventRepository(mem *MemorySurgeEventRepository, db ContentDB) *PersistentSurgeEventRepository {
	return &PersistentSurgeEventRepository{mem: mem, db: db}
}

func (r *PersistentSurgeEventRepository) Create(ctx context.Context, se *upal.SurgeEvent) error {
	_ = r.mem.Create(ctx, se)
	if err := r.db.CreateSurgeEvent(ctx, se); err != nil {
		return fmt.Errorf("db create surge_event: %w", err)
	}
	return nil
}

func (r *PersistentSurgeEventRepository) List(ctx context.Context) ([]*upal.SurgeEvent, error) {
	events, err := r.db.ListSurgeEvents(ctx)
	if err == nil {
		return events, nil
	}
	slog.Warn("db list surge_events failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentSurgeEventRepository) ListActive(ctx context.Context) ([]*upal.SurgeEvent, error) {
	events, err := r.db.ListActiveSurgeEvents(ctx)
	if err == nil {
		return events, nil
	}
	slog.Warn("db list active surge_events failed, falling back to in-memory", "err", err)
	return r.mem.ListActive(ctx)
}

func (r *PersistentSurgeEventRepository) Get(ctx context.Context, id string) (*upal.SurgeEvent, error) {
	if se, err := r.mem.Get(ctx, id); err == nil {
		return se, nil
	}
	return nil, fmt.Errorf("surge event %q not found", id)
}

func (r *PersistentSurgeEventRepository) Update(ctx context.Context, se *upal.SurgeEvent) error {
	_ = r.mem.Update(ctx, se)
	if err := r.db.UpdateSurgeEvent(ctx, se); err != nil {
		return fmt.Errorf("db update surge_event: %w", err)
	}
	return nil
}
```

**Step 2: Verify compilation**

```bash
go build ./internal/repository/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/repository/content_persistent.go
git commit -m "feat(content): add persistent repositories for content types"
```

---

## Task 10: ContentSessionService

**Goal:** Thin service layer orchestrating CRUD + approve/reject for content sessions.

**Files:**
- Create: `internal/services/content_session_service.go`

**Step 1: Write failing test**

Create `internal/services/content_session_service_test.go`:

```go
package services_test

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

func TestContentSessionService_CreateAndApprove(t *testing.T) {
	sessionRepo := repository.NewMemoryContentSessionRepository()
	fetchRepo := repository.NewMemorySourceFetchRepository()
	analysisRepo := repository.NewMemoryLLMAnalysisRepository()
	publishedRepo := repository.NewMemoryPublishedContentRepository()
	surgeRepo := repository.NewMemorySurgeEventRepository()

	svc := services.NewContentSessionService(sessionRepo, fetchRepo, analysisRepo, publishedRepo, surgeRepo)
	ctx := context.Background()

	s := &upal.ContentSession{
		PipelineID:  "pipe-1",
		TriggerType: "manual",
	}
	if err := svc.CreateSession(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	if s.ID == "" {
		t.Error("expected ID to be generated")
	}
	if s.Status != upal.SessionCollecting {
		t.Errorf("expected collecting, got %q", s.Status)
	}

	if err := svc.ApproveSession(ctx, s.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	got, _ := svc.GetSession(ctx, s.ID)
	if got.Status != upal.SessionApproved {
		t.Errorf("expected approved, got %q", got.Status)
	}
	if got.ReviewedAt == nil {
		t.Error("expected ReviewedAt to be set on approval")
	}
}

func TestContentSessionService_Reject(t *testing.T) {
	sessionRepo := repository.NewMemoryContentSessionRepository()
	svc := services.NewContentSessionService(
		sessionRepo,
		repository.NewMemorySourceFetchRepository(),
		repository.NewMemoryLLMAnalysisRepository(),
		repository.NewMemoryPublishedContentRepository(),
		repository.NewMemorySurgeEventRepository(),
	)
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)
	if err := svc.RejectSession(ctx, s.ID); err != nil {
		t.Fatalf("reject: %v", err)
	}
	got, _ := svc.GetSession(ctx, s.ID)
	if got.Status != upal.SessionRejected {
		t.Errorf("expected rejected, got %q", got.Status)
	}
}

func TestContentSessionService_DismissSurge(t *testing.T) {
	svc := services.NewContentSessionService(
		repository.NewMemoryContentSessionRepository(),
		repository.NewMemorySourceFetchRepository(),
		repository.NewMemoryLLMAnalysisRepository(),
		repository.NewMemoryPublishedContentRepository(),
		repository.NewMemorySurgeEventRepository(),
	)
	ctx := context.Background()

	surge := &upal.SurgeEvent{Keyword: "test", Multiplier: 5.0}
	if err := svc.CreateSurge(ctx, surge); err != nil {
		t.Fatalf("create surge: %v", err)
	}
	if err := svc.DismissSurge(ctx, surge.ID); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	active, _ := svc.ListActiveSurges(ctx)
	if len(active) != 0 {
		t.Fatalf("expected 0 active surges, got %d", len(active))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/services/... -run "TestContentSessionService" -v 2>&1 | head -20
```
Expected: FAIL — `services.NewContentSessionService` undefined.

**Step 3: Create the service**

Create `internal/services/content_session_service.go`:

```go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// ContentSessionService manages content collection sessions and related records.
type ContentSessionService struct {
	sessions  repository.ContentSessionRepository
	fetches   repository.SourceFetchRepository
	analyses  repository.LLMAnalysisRepository
	published repository.PublishedContentRepository
	surges    repository.SurgeEventRepository
}

func NewContentSessionService(
	sessions repository.ContentSessionRepository,
	fetches repository.SourceFetchRepository,
	analyses repository.LLMAnalysisRepository,
	published repository.PublishedContentRepository,
	surges repository.SurgeEventRepository,
) *ContentSessionService {
	return &ContentSessionService{
		sessions:  sessions,
		fetches:   fetches,
		analyses:  analyses,
		published: published,
		surges:    surges,
	}
}

// --- ContentSession ---

func (s *ContentSessionService) CreateSession(ctx context.Context, sess *upal.ContentSession) error {
	if sess.PipelineID == "" {
		return fmt.Errorf("pipeline_id is required")
	}
	if sess.ID == "" {
		sess.ID = upal.GenerateID("csess")
	}
	if sess.Status == "" {
		sess.Status = upal.SessionCollecting
	}
	if sess.TriggerType == "" {
		sess.TriggerType = "manual"
	}
	sess.CreatedAt = time.Now()
	return s.sessions.Create(ctx, sess)
}

func (s *ContentSessionService) GetSession(ctx context.Context, id string) (*upal.ContentSession, error) {
	return s.sessions.Get(ctx, id)
}

func (s *ContentSessionService) ListSessions(ctx context.Context) ([]*upal.ContentSession, error) {
	return s.sessions.List(ctx)
}

func (s *ContentSessionService) ListSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListByPipeline(ctx, pipelineID)
}

func (s *ContentSessionService) UpdateSessionStatus(ctx context.Context, id string, status upal.ContentSessionStatus) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	sess.Status = status
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) ApproveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	sess.Status = upal.SessionApproved
	sess.ReviewedAt = &now
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) RejectSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now()
	sess.Status = upal.SessionRejected
	sess.ReviewedAt = &now
	return s.sessions.Update(ctx, sess)
}

// --- SourceFetch ---

func (s *ContentSessionService) RecordSourceFetch(ctx context.Context, sf *upal.SourceFetch) error {
	if sf.ID == "" {
		sf.ID = upal.GenerateID("sfetch")
	}
	sf.FetchedAt = time.Now()
	return s.fetches.Create(ctx, sf)
}

func (s *ContentSessionService) ListSourceFetches(ctx context.Context, sessionID string) ([]*upal.SourceFetch, error) {
	return s.fetches.ListBySession(ctx, sessionID)
}

// --- LLMAnalysis ---

func (s *ContentSessionService) RecordAnalysis(ctx context.Context, a *upal.LLMAnalysis) error {
	if a.ID == "" {
		a.ID = upal.GenerateID("anlys")
	}
	a.CreatedAt = time.Now()
	return s.analyses.Create(ctx, a)
}

func (s *ContentSessionService) GetAnalysis(ctx context.Context, sessionID string) (*upal.LLMAnalysis, error) {
	return s.analyses.GetBySession(ctx, sessionID)
}

// --- PublishedContent ---

func (s *ContentSessionService) RecordPublished(ctx context.Context, pc *upal.PublishedContent) error {
	if pc.ID == "" {
		pc.ID = upal.GenerateID("pub")
	}
	pc.PublishedAt = time.Now()
	return s.published.Create(ctx, pc)
}

func (s *ContentSessionService) ListPublished(ctx context.Context) ([]*upal.PublishedContent, error) {
	return s.published.List(ctx)
}

func (s *ContentSessionService) ListPublishedBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error) {
	return s.published.ListBySession(ctx, sessionID)
}

func (s *ContentSessionService) ListPublishedByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error) {
	return s.published.ListByChannel(ctx, channel)
}

// --- SurgeEvent ---

func (s *ContentSessionService) CreateSurge(ctx context.Context, se *upal.SurgeEvent) error {
	if se.ID == "" {
		se.ID = upal.GenerateID("surge")
	}
	se.CreatedAt = time.Now()
	return s.surges.Create(ctx, se)
}

func (s *ContentSessionService) ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return s.surges.List(ctx)
}

func (s *ContentSessionService) ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error) {
	return s.surges.ListActive(ctx)
}

func (s *ContentSessionService) DismissSurge(ctx context.Context, id string) error {
	se, err := s.surges.Get(ctx, id)
	if err != nil {
		return err
	}
	se.Dismissed = true
	return s.surges.Update(ctx, se)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/services/... -run "TestContentSessionService" -v -race
```
Expected: PASS (all three tests).

**Step 5: Commit**

```bash
git add internal/services/content_session_service.go internal/services/content_session_service_test.go
git commit -m "feat(content): add ContentSessionService with approve/reject/surge (with tests)"
```

---

## Task 11: API handlers for content endpoints

**Goal:** Add HTTP handlers for `/api/content-sessions`, `/api/published`, `/api/surges`, and wire them into `Server`.

**Files:**
- Create: `internal/api/content.go`
- Modify: `internal/api/server.go`

**Step 1: Create content.go**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// GET /api/content-sessions
// Query params: pipeline_id=X, status=pending_review
func (s *Server) listContentSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pipelineID := r.URL.Query().Get("pipeline_id")
	var (
		sessions []*upal.ContentSession
		err      error
	)
	if pipelineID != "" {
		sessions, err = s.contentSvc.ListSessionsByPipeline(ctx, pipelineID)
	} else {
		sessions, err = s.contentSvc.ListSessions(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Optional client-side filter by status
	if status := r.URL.Query().Get("status"); status != "" {
		filtered := sessions[:0]
		for _, sess := range sessions {
			if string(sess.Status) == status {
				filtered = append(filtered, sess)
			}
		}
		sessions = filtered
	}
	if sessions == nil {
		sessions = []*upal.ContentSession{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// GET /api/content-sessions/{id}
func (s *Server) getContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// PATCH /api/content-sessions/{id}
// Body: {"action": "approve" | "reject"}
func (s *Server) patchContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	var err error
	switch body.Action {
	case "approve":
		err = s.contentSvc.ApproveSession(ctx, id)
	case "reject":
		err = s.contentSvc.RejectSession(ctx, id)
	default:
		http.Error(w, "action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sess, _ := s.contentSvc.GetSession(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// POST /api/content-sessions/{id}/produce
// Body: {"workflows": ["blog", "shorts"]}
// Phase 1: Records intent (actual execution wired in Phase 3).
func (s *Server) produceContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Workflows []string `json:"workflows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.Workflows) == 0 {
		http.Error(w, "workflows list is required", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	if err := s.contentSvc.UpdateSessionStatus(ctx, id, upal.SessionProducing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"session_id": id,
		"workflows":  body.Workflows,
		"status":     "accepted",
	})
}

// GET /api/content-sessions/{id}/sources
func (s *Server) listSessionSources(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fetches, err := s.contentSvc.ListSourceFetches(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fetches == nil {
		fetches = []*upal.SourceFetch{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fetches)
}

// GET /api/content-sessions/{id}/analysis
func (s *Server) getSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.contentSvc.GetAnalysis(r.Context(), id)
	if err != nil {
		http.Error(w, "analysis not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// GET /api/published
// Query params: pipeline_id=X, session_id=X, channel=youtube
func (s *Server) listPublished(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var (
		items []*upal.PublishedContent
		err   error
	)
	if channel := r.URL.Query().Get("channel"); channel != "" {
		items, err = s.contentSvc.ListPublishedByChannel(ctx, channel)
	} else if sessionID := r.URL.Query().Get("session_id"); sessionID != "" {
		items, err = s.contentSvc.ListPublishedBySession(ctx, sessionID)
	} else {
		items, err = s.contentSvc.ListPublished(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []*upal.PublishedContent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GET /api/surges
func (s *Server) listSurges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var (
		events []*upal.SurgeEvent
		err    error
	)
	if r.URL.Query().Get("active") == "true" {
		events, err = s.contentSvc.ListActiveSurges(ctx)
	} else {
		events, err = s.contentSvc.ListSurges(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []*upal.SurgeEvent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// POST /api/surges/{id}/dismiss
func (s *Server) dismissSurge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DismissSurge(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/surges/{id}/create-session
// Phase 1: Creates a session from a surge event.
func (s *Server) createSessionFromSurge(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id") // surge ID — session creation from surge wired in Phase 2
	http.Error(w, "not implemented in Phase 1", http.StatusNotImplemented)
}

// POST /api/pipelines/{id}/collect
// Phase 1: Creates a session manually (actual collection wired in Phase 2).
func (s *Server) collectPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess := &upal.ContentSession{
		PipelineID:  id,
		TriggerType: "manual",
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sess)
}
```

**Step 2: Add contentSvc to Server and register routes in server.go**

In `internal/api/server.go`:

1. Add `contentSvc *services.ContentSessionService` to the `Server` struct (after `pipelineRunner`).

2. Add a setter method after `SetPipelineRunner`:

```go
// SetContentSessionService configures the content session management service.
func (s *Server) SetContentSessionService(svc *services.ContentSessionService) {
	s.contentSvc = svc
}
```

3. In `Handler()`, add new route groups inside `r.Route("/api", ...)`, after the pipelines block:

```go
		if s.contentSvc != nil {
			r.Route("/content-sessions", func(r chi.Router) {
				r.Get("/", s.listContentSessions)
				r.Get("/{id}", s.getContentSession)
				r.Patch("/{id}", s.patchContentSession)
				r.Post("/{id}/produce", s.produceContentSession)
				r.Get("/{id}/sources", s.listSessionSources)
				r.Get("/{id}/analysis", s.getSessionAnalysis)
			})
			r.Route("/published", func(r chi.Router) {
				r.Get("/", s.listPublished)
			})
			r.Route("/surges", func(r chi.Router) {
				r.Get("/", s.listSurges)
				r.Post("/{id}/dismiss", s.dismissSurge)
				r.Post("/{id}/create-session", s.createSessionFromSurge)
			})
		}
```

4. Also add the `collect` route to the existing pipelines block:

```go
			r.Post("/{id}/collect", s.collectPipeline)
```

**Step 3: Verify compilation**

```bash
go build ./internal/api/...
```
Expected: PASS.

**Step 4: Commit**

```bash
git add internal/api/content.go internal/api/server.go
git commit -m "feat(content): add API handlers for /content-sessions, /published, /surges"
```

---

## Task 12: New connection types

**Goal:** Add connection type constants for external services used in Phase 2+ tools.

**Files:**
- Modify: `internal/upal/connection.go`

**Step 1: Read the existing file and add constants**

In `internal/upal/connection.go`, add to the existing `const` block:

```go
const (
	ConnTypeTelegram  ConnectionType = "telegram"
	ConnTypeSlack     ConnectionType = "slack"
	ConnTypeHTTP      ConnectionType = "http"
	ConnTypeSMTP      ConnectionType = "smtp"
	// Content media pipeline
	ConnTypeReddit    ConnectionType = "reddit"
	ConnTypeYouTube   ConnectionType = "youtube"
	ConnTypeDiscord   ConnectionType = "discord"
	ConnTypeSubstack  ConnectionType = "substack"
	ConnTypeNewsAPI   ConnectionType = "newsapi"
	ConnTypeSerpAPI   ConnectionType = "serpapi"
)
```

**Step 2: Verify compilation**

```bash
go build ./internal/upal/...
```
Expected: PASS.

**Step 3: Commit**

```bash
git add internal/upal/connection.go
git commit -m "feat(content): add connection type constants for Reddit, YouTube, Discord, Substack, NewsAPI, SerpAPI"
```

---

## Task 13: Wire everything in main.go

**Goal:** Construct content repos → service → register with server. Also add `session_id` to `runs` DB upsert if needed.

**Files:**
- Modify: `cmd/upal/main.go`

**Step 1: Add content wiring after the pipeline block (~line 257)**

Find the section `// Pipeline` in `cmd/upal/main.go` and add the following block after `schedulerSvc.SetPipelineService(pipelineSvc)`:

```go
	// Content media pipeline
	memContentSessionRepo := repository.NewMemoryContentSessionRepository()
	memSourceFetchRepo := repository.NewMemorySourceFetchRepository()
	memLLMAnalysisRepo := repository.NewMemoryLLMAnalysisRepository()
	memPublishedRepo := repository.NewMemoryPublishedContentRepository()
	memSurgeRepo := repository.NewMemorySurgeEventRepository()

	var contentSessionRepo repository.ContentSessionRepository = memContentSessionRepo
	var sourceFetchRepo repository.SourceFetchRepository = memSourceFetchRepo
	var llmAnalysisRepo repository.LLMAnalysisRepository = memLLMAnalysisRepo
	var publishedRepo repository.PublishedContentRepository = memPublishedRepo
	var surgeRepo repository.SurgeEventRepository = memSurgeRepo

	if database != nil {
		contentSessionRepo = repository.NewPersistentContentSessionRepository(memContentSessionRepo, database)
		sourceFetchRepo = repository.NewPersistentSourceFetchRepository(memSourceFetchRepo, database)
		llmAnalysisRepo = repository.NewPersistentLLMAnalysisRepository(memLLMAnalysisRepo, database)
		publishedRepo = repository.NewPersistentPublishedContentRepository(memPublishedRepo, database)
		surgeRepo = repository.NewPersistentSurgeEventRepository(memSurgeRepo, database)
	}

	contentSvc := services.NewContentSessionService(
		contentSessionRepo, sourceFetchRepo, llmAnalysisRepo, publishedRepo, surgeRepo,
	)
	srv.SetContentSessionService(contentSvc)
```

**Step 2: Verify full build**

```bash
go build ./...
```
Expected: PASS.

**Step 3: Run all tests**

```bash
go test ./... -race
```
Expected: All existing tests pass + new content tests pass.

**Step 4: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat(content): wire content session service in main — Phase 1 complete"
```

---

## Verification

After all tasks complete:

```bash
# Full build
go build ./...

# All tests
go test ./... -v -race

# Start dev server and verify endpoints respond
make dev-backend &
curl -s http://localhost:8080/api/content-sessions | jq .
curl -s http://localhost:8080/api/published | jq .
curl -s http://localhost:8080/api/surges | jq .
curl -s http://localhost:8080/api/pipelines | jq '.[0].context'
```

Expected: All return valid JSON (empty arrays for new endpoints, context field present in pipeline objects).

---

## Phase 1 Summary

| Layer | Files Changed |
|-------|--------------|
| Domain | `internal/upal/pipeline.go`, `internal/upal/content.go`, `internal/upal/scheduler.go`, `internal/upal/connection.go` |
| DB | `internal/db/db.go`, `internal/db/pipeline.go`, `internal/db/content.go` |
| Repository | `internal/repository/content.go`, `content_memory.go`, `content_persistent.go` |
| Service | `internal/services/content_session_service.go` |
| API | `internal/api/content.go`, `internal/api/server.go` |
| Wire | `cmd/upal/main.go` |
