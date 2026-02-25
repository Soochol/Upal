# Session Owns All Settings — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move all pipeline-level settings (sources, schedule, model, workflows, context) to the session level, making Pipeline a pure group/folder.

**Architecture:** Pipeline struct loses 5 settings fields. ContentSession already has them — we activate DB persistence, make the collector/analyzer read from session instead of pipeline, and swap the frontend settings panel from pipeline to session scope.

**Tech Stack:** Go 1.24, PostgreSQL, React 19, TypeScript, TanStack Query

---

### Task 1: Remove settings fields from Pipeline model

**Files:**
- Modify: `internal/upal/pipeline.go:6-22`

**Step 1: Remove fields from Pipeline struct**

```go
type Pipeline struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Stages       []Stage   `json:"stages"`
	ThumbnailSVG string    `json:"thumbnail_svg,omitempty"`
	LastCollectedAt      *time.Time `json:"last_collected_at,omitempty"`
	PendingSessionCount  int        `json:"pending_session_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

Removed: `Sources`, `Schedule`, `Model`, `Workflows`, `Context`

**Step 2: Run `go build ./...` to find all compilation errors**

Run: `go build ./...`
Expected: Multiple compilation errors in files that reference the removed fields

**Step 3: Commit**

```bash
git add internal/upal/pipeline.go
git commit -m "refactor: remove settings fields from Pipeline model"
```

---

### Task 2: Update DB pipeline CRUD (remove settings columns)

**Files:**
- Modify: `internal/db/db.go:170-172,241`
- Modify: `internal/db/pipeline.go:13-125`

**Step 1: Update DB schema migrations**

In `internal/db/db.go`, remove these ALTER TABLE statements (lines 170-172, 241):
```sql
-- REMOVE these lines:
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS context JSONB NOT NULL DEFAULT '{}';
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS schedule TEXT NOT NULL DEFAULT '';
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '';
```

**Step 2: Update CreatePipeline — remove context/sources/schedule/model**

```go
func (d *DB) CreatePipeline(ctx context.Context, p *upal.Pipeline) error {
	stagesJSON, err := json.Marshal(p.Stages)
	if err != nil {
		return fmt.Errorf("marshal stages: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO pipelines (id, name, description, stages, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.Name, p.Description, stagesJSON, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline: %w", err)
	}
	return nil
}
```

**Step 3: Update GetPipeline — remove context/sources/schedule/model scan**

```go
func (d *DB) GetPipeline(ctx context.Context, id string) (*upal.Pipeline, error) {
	var p upal.Pipeline
	var stagesJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, name, description, stages, created_at, updated_at
		 FROM pipelines WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}
	if err := json.Unmarshal(stagesJSON, &p.Stages); err != nil {
		return nil, fmt.Errorf("unmarshal stages: %w", err)
	}
	return &p, nil
}
```

**Step 4: Update ListPipelines — same pattern, remove settings fields**

**Step 5: Update UpdatePipeline — remove settings fields from UPDATE**

```go
func (d *DB) UpdatePipeline(ctx context.Context, p *upal.Pipeline) error {
	stagesJSON, err := json.Marshal(p.Stages)
	if err != nil {
		return fmt.Errorf("marshal stages: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE pipelines SET name = $1, description = $2, stages = $3, updated_at = $4
		 WHERE id = $5`,
		p.Name, p.Description, stagesJSON, p.UpdatedAt, p.ID,
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

**Step 6: Run `go build ./...`**

Expected: Fewer errors now (DB layer compiles), remaining errors in services

**Step 7: Commit**

```bash
git add internal/db/db.go internal/db/pipeline.go
git commit -m "refactor: remove settings columns from pipeline DB operations"
```

---

### Task 3: Add settings columns to content_sessions DB

**Files:**
- Modify: `internal/db/db.go` (add ALTER TABLE statements)
- Modify: `internal/db/content.go:14-205`

**Step 1: Add session settings columns to DB schema**

Append to `internal/db/db.go` (before closing backtick):
```sql
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS sources JSONB NOT NULL DEFAULT '[]';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS schedule TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS workflows JSONB NOT NULL DEFAULT '[]';
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS context JSONB;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS parent_session_id TEXT NOT NULL DEFAULT '';
```

**Step 2: Update CreateContentSession to persist new fields**

```go
func (d *DB) CreateContentSession(ctx context.Context, s *upal.ContentSession) error {
	sourcesJSON, err := json.Marshal(s.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	workflowsJSON, err := json.Marshal(s.Workflows)
	if err != nil {
		return fmt.Errorf("marshal workflows: %w", err)
	}
	ctxJSON, err := json.Marshal(s.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO content_sessions (id, pipeline_id, name, status, trigger_type, source_count, sources, schedule, model, workflows, context, is_template, parent_session_id, created_at, reviewed_at, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		s.ID, s.PipelineID, s.Name, string(s.Status), s.TriggerType, s.SourceCount,
		sourcesJSON, s.Schedule, s.Model, workflowsJSON, ctxJSON,
		s.IsTemplate, s.ParentSessionID, s.CreatedAt, s.ReviewedAt, s.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("insert content_session: %w", err)
	}
	return nil
}
```

**Step 3: Update GetContentSession to scan new fields**

Add scanning for: `name`, `sources` (JSONB), `schedule`, `model`, `workflows` (JSONB), `context` (JSONB), `is_template`, `parent_session_id`

**Step 4: Update all List* functions with the same column additions**

Functions to update: `ListContentSessions`, `ListContentSessionsByPipeline`, `ListContentSessionsByStatus`, `ListAllContentSessionsByStatus`, `ListContentSessionsByPipelineAndStatus`, `ListArchivedContentSessionsByPipeline`

**Step 5: Update UpdateContentSession to persist all settings fields**

```go
func (d *DB) UpdateContentSession(ctx context.Context, s *upal.ContentSession) error {
	sourcesJSON, err := json.Marshal(s.Sources)
	if err != nil {
		return fmt.Errorf("marshal sources: %w", err)
	}
	workflowsJSON, err := json.Marshal(s.Workflows)
	if err != nil {
		return fmt.Errorf("marshal workflows: %w", err)
	}
	ctxJSON, err := json.Marshal(s.Context)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE content_sessions SET name = $1, status = $2, source_count = $3, sources = $4, schedule = $5, model = $6, workflows = $7, context = $8, is_template = $9, parent_session_id = $10, reviewed_at = $11, archived_at = $12 WHERE id = $13`,
		s.Name, string(s.Status), s.SourceCount,
		sourcesJSON, s.Schedule, s.Model, workflowsJSON, ctxJSON,
		s.IsTemplate, s.ParentSessionID, s.ReviewedAt, s.ArchivedAt, s.ID,
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
```

**Step 6: Run `go build ./...`**

Expected: DB layer compiles

**Step 7: Commit**

```bash
git add internal/db/db.go internal/db/content.go
git commit -m "feat: add settings columns to content_sessions DB layer"
```

---

### Task 4: Update ContentCollector to use session settings

**Files:**
- Modify: `internal/services/content_collector.go:69-83,91-130,277-297,399-464,685-744`

**Step 1: Update CollectPipeline to copy session sources**

In `CollectPipeline` (line 69-83): The newly created session currently has no sources. The scheduler dispatch should find the active template session for the pipeline and use its sources. Change `CollectAndAnalyze` signature to accept session (which now owns sources) instead of pipeline.

**Step 2: Update CollectAndAnalyze to read from session.Sources**

```go
func (c *ContentCollector) CollectAndAnalyze(ctx context.Context, pipeline *upal.Pipeline, session *upal.ContentSession, isTest bool, limit int) {
	// Use session sources instead of pipeline sources
	sources := mapPipelineSources(session.Sources, isTest, limit)
	// ...rest same but use session.Sources[mapped.pipelineIndex] instead of pipeline.Sources[mapped.pipelineIndex]
```

Line 93: `pipeline.Sources` → `session.Sources`
Line 106: `pipeline.Sources[mapped.pipelineIndex]` → `session.Sources[mapped.pipelineIndex]`

**Step 3: Update runAnalysis to use session.Model**

Line 291: `c.resolver.Resolve(pipeline.Model)` → `c.resolver.Resolve(session.Model)`

**Step 4: Update buildAnalysisPrompt to accept session instead of pipeline**

Change signature: `buildAnalysisPrompt(systemPromptBase string, session *upal.ContentSession, ...)`

Line 405: `pipeline.Context` → `session.Context`
Line 427: `pipeline.Name` → use pipeline name from repo if needed for fallback
Line 432: `pipeline.Workflows` → `session.Workflows`

**Step 5: Update GenerateWorkflowForAngle to read context from session**

Lines 721-744: `pipeline.Context` → `session.Context`

**Step 6: Run `go build ./...`**

Expected: Compiles (or close to it)

**Step 7: Commit**

```bash
git add internal/services/content_collector.go
git commit -m "refactor: collector reads settings from session instead of pipeline"
```

---

### Task 5: Update scheduler dispatch

**Files:**
- Modify: `internal/services/scheduler/dispatch.go:22-42`

**Step 1: Update the content pipeline detection**

Currently line 34: `if len(pipeline.Sources) > 0` — this no longer works since pipeline has no sources.

Change to: The scheduler should call `CollectPipeline` unconditionally for pipelines that have sources configured in their template sessions. Alternatively, simplify: if the schedule has a PipelineID and collector is available, always use the collector path.

```go
// Content pipelines always go through the content collector lifecycle.
if s.contentCollector != nil {
	if err := s.contentCollector.CollectPipeline(ctx, schedule.PipelineID); err != nil {
		slog.Error("scheduler: content pipeline collection failed",
			"schedule", schedule.ID, "pipeline", schedule.PipelineID, "err", err)
	}
} else if _, err := s.pipelineRunner.Start(ctx, pipeline, nil); err != nil {
```

**Step 2: Update CollectPipeline to find template session and copy its sources**

In `content_collector.go:CollectPipeline`, we need to find the active template session for this pipeline and create an instance from it:

```go
func (c *ContentCollector) CollectPipeline(ctx context.Context, pipelineID string) error {
	pipeline, err := c.pipelineRepo.Get(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("pipeline %s: %w", pipelineID, err)
	}

	// Find the active template session for this pipeline
	templates, err := c.contentSvc.ListTemplatesByPipeline(ctx, pipelineID)
	if err != nil || len(templates) == 0 {
		return fmt.Errorf("no template session for pipeline %s", pipelineID)
	}
	template := templates[0] // use first active template

	// Create instance session with template's settings
	sess := &upal.ContentSession{
		PipelineID:      pipelineID,
		TriggerType:     "scheduled",
		ParentSessionID: template.ID,
		Sources:         template.Sources,
		Schedule:        template.Schedule,
		Model:           template.Model,
		Workflows:       template.Workflows,
		Context:         template.Context,
	}
	if err := c.contentSvc.CreateSession(ctx, sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	go c.CollectAndAnalyze(context.Background(), pipeline, sess, false, 0)
	return nil
}
```

**Step 3: Run tests**

Run: `go test ./internal/services/... -v -race`
Expected: Passes (or identifies test files to update)

**Step 4: Commit**

```bash
git add internal/services/scheduler/dispatch.go internal/services/content_collector.go
git commit -m "refactor: scheduler and collector use template session settings"
```

---

### Task 6: Update API handlers — collect pipeline endpoint

**Files:**
- Modify: `internal/api/content.go:609-644` (collectPipeline handler)
- Modify: `internal/api/content.go:525-552` (collectSession handler)

**Step 1: Update collectPipeline handler**

The `collectPipeline` handler (line 609) currently creates a bare session and passes the pipeline to the collector. Now it needs to find the template session and copy settings:

```go
func (s *Server) collectPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	pipeline, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	// Find active template session to get sources
	templates, err := s.contentSvc.ListTemplatesByPipeline(r.Context(), id)
	if err != nil || len(templates) == 0 {
		http.Error(w, "no template session found for this pipeline", http.StatusBadRequest)
		return
	}
	template := templates[0]

	sess := &upal.ContentSession{
		PipelineID:      id,
		TriggerType:     "manual",
		ParentSessionID: template.ID,
		Sources:         template.Sources,
		Schedule:        template.Schedule,
		Model:           template.Model,
		Workflows:       template.Workflows,
		Context:         template.Context,
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), pipeline, sess, body.IsTest, body.Limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": sess.ID})
}
```

**Step 2: Update collectSession handler**

The `collectSession` handler (line 525) passes the pipeline to `CollectAndAnalyze` — it should pass the session which now owns the sources:

Line 546: Already passes `pipeline` and `sess`. The collector now reads from `sess.Sources`, so this is fine as long as the session has sources. If the session doesn't have sources, the collector will log "no fetchable sources" and skip.

**Step 3: Update configure_pipeline.go**

The configure pipeline endpoint (line 61-69) currently reads pipeline settings. This should now read from the selected session instead. However, this is an AI chat endpoint — it can stay as-is for now since it receives `current_sources` etc. from the frontend, which will send session data.

**Step 4: Run `go build ./...`**

Expected: Compiles

**Step 5: Commit**

```bash
git add internal/api/content.go
git commit -m "refactor: collect pipeline handler uses template session settings"
```

---

### Task 7: Fix remaining Go compilation errors

**Files:**
- Potentially: `internal/generate/pipeline.go` (PipelineBundle uses `upal.Pipeline` which no longer has settings)
- Potentially: `internal/api/pipelines.go`
- Any other files found by `go build ./...`

**Step 1: Run `go build ./...` and fix all remaining errors**

The PipelineBundle in `internal/generate/pipeline.go` returns `upal.Pipeline` which no longer has Sources/Schedule/Model etc. The generation may still produce these fields in the JSON response — they'll just be ignored since Pipeline struct no longer has them. This is acceptable.

**Step 2: Run full test suite**

Run: `make test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add -A
git commit -m "fix: resolve remaining compilation errors after pipeline field removal"
```

---

### Task 8: Update frontend Pipeline type

**Files:**
- Modify: `web/src/entities/pipeline/types.ts:42-58`
- Modify: `web/src/entities/pipeline/api/index.ts:73-88`

**Step 1: Remove settings fields from Pipeline type**

```typescript
export type Pipeline = {
  id: string
  name: string
  description?: string
  stages: Stage[]
  thumbnail_svg?: string
  last_collected_at?: string
  pending_session_count?: number
  created_at: string
  updated_at: string
}
```

Removed: `sources`, `workflows`, `context`, `schedule`, `model`

**Step 2: Remove `updatePipelineSources` and `updatePipelineContext` from API**

Delete functions at lines 73-88 in `web/src/entities/pipeline/api/index.ts` — these are pipeline-level and no longer needed. The session settings API (`updateSessionSettings`) replaces them.

**Step 3: Run type-check**

Run: `cd web && npx tsc -b`
Expected: Type errors in Pipelines.tsx and PipelineSettingsPanel.tsx (expected — we fix those next)

**Step 4: Commit**

```bash
git add web/src/entities/pipeline/types.ts web/src/entities/pipeline/api/index.ts
git commit -m "refactor: remove settings fields from frontend Pipeline type"
```

---

### Task 9: Replace PipelineSettingsPanel with session-scoped settings

**Files:**
- Modify: `web/src/pages/Pipelines.tsx:1-421`
- Modify: `web/src/pages/pipelines/PipelineSettingsPanel.tsx:1-294`

**Step 1: Update PipelineSettingsPanel props to use session data**

Change the component to receive a `sessionId` and call `updateSessionSettings` instead of `updatePipeline`. The component's UI stays the same (sources, schedule, model, workflows, editorial brief, AI chat) — only the data source and save target changes.

Props change from:
```typescript
{
  pipelineId: string
  sources: PipelineSource[]
  schedule: string
  // ...pipeline-level settings
}
```
To:
```typescript
{
  pipelineId: string
  session: ContentSession | null
  channels: { id: string; name: string; type: string }[]
}
```

The component internally manages local state from `session.session_sources`, `session.schedule`, etc. and auto-saves via `updateSessionSettings(session.id, ...)`.

**Step 2: Update Pipelines.tsx — remove pipeline-level settings state**

Remove:
- `localSources`, `localSchedule`, `localWorkflows`, `localModel` state (lines 144-148)
- All pipeline sync effects (lines 151-223)
- `updateContextMutation` (line 136-140)
- Pipeline settings props passed to PipelineSettingsPanel (lines 376-390)

Replace with: Pass `selectedSessionId` and selected session data to `PipelineSettingsPanel`.

**Step 3: PipelineSettingsPanel handles its own data fetching and auto-save**

The panel should:
1. Fetch session detail when `sessionId` changes
2. Initialize local state from `session.session_sources`, `session.schedule`, `session.model`, `session.session_workflows`, `session.context`
3. Auto-save dirty state via `updateSessionSettings(sessionId, { sources, schedule, ... })`
4. Show "No session selected" placeholder when no session is selected

**Step 4: Update AI chat to use session context**

The `PipelineChatEditor` receives `currentSources` etc. — update these to come from session state instead of pipeline state.

**Step 5: Run type-check and dev server**

Run: `cd web && npx tsc -b`
Expected: No type errors

**Step 6: Commit**

```bash
git add web/src/pages/Pipelines.tsx web/src/pages/pipelines/PipelineSettingsPanel.tsx
git commit -m "feat: settings panel reads/writes session-level settings"
```

---

### Task 10: Final integration test and cleanup

**Step 1: Run backend tests**

Run: `make test`
Expected: All pass

**Step 2: Run frontend type-check**

Run: `make test-frontend`
Expected: Clean

**Step 3: Manual smoke test**

1. Open pipeline detail page
2. Select a session → settings panel shows that session's settings
3. Edit sources → auto-saves to session
4. Switch session → settings panel updates
5. No session selected → settings panel shows placeholder

**Step 4: Commit**

```bash
git add -A
git commit -m "feat: session owns all pipeline settings — pipeline is group only"
```
