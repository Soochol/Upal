# Session Template/Instance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Explicitly separate content sessions into templates (reusable configs in pipeline page) and instances (execution results in inbox), with `is_template` and `parent_session_id` fields.

**Architecture:** Add two fields to ContentSession. Templates stay in `draft` status forever and are displayed in the pipeline page. Triggering collection always creates a new instance session that flows through the full lifecycle into inbox. API filtering ensures templates and instances are queried separately.

**Tech Stack:** Go 1.24, PostgreSQL, React 19, TanStack Query

---

### Task 1: Domain Model — Add fields to ContentSession

**Files:**
- Modify: `internal/upal/content.go:23-39`

**Step 1: Add IsTemplate and ParentSessionID to ContentSession struct**

```go
type ContentSession struct {
	ID              string               `json:"id"`
	PipelineID      string               `json:"pipeline_id"`
	Name            string               `json:"name"`
	Status          ContentSessionStatus `json:"status"`
	TriggerType     string               `json:"trigger_type"`
	SourceCount     int                  `json:"source_count"`
	IsTemplate      bool                 `json:"is_template"`
	ParentSessionID string               `json:"parent_session_id,omitempty"`
	// Session-level settings (moved from Pipeline)
	Sources   []PipelineSource   `json:"session_sources,omitempty"`
	Schedule  string             `json:"schedule,omitempty"`
	Model     string             `json:"model,omitempty"`
	Workflows []PipelineWorkflow `json:"session_workflows,omitempty"`
	Context   *PipelineContext   `json:"context,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	ReviewedAt  *time.Time           `json:"reviewed_at,omitempty"`
	ArchivedAt  *time.Time           `json:"archived_at,omitempty"`
}
```

**Step 2: Add same fields to ContentSessionDetail**

In `ContentSessionDetail` struct (line ~150), add:
```go
	IsTemplate      bool                 `json:"is_template"`
	ParentSessionID string               `json:"parent_session_id,omitempty"`
```

**Step 3: Run tests**

Run: `go build ./...`
Expected: PASS (no compile errors — fields added, not yet used)

**Step 4: Commit**

```bash
git add internal/upal/content.go
git commit -m "feat: add IsTemplate and ParentSessionID to ContentSession"
```

---

### Task 2: DB Schema — Add columns + migration

**Files:**
- Modify: `internal/db/db.go:176` (migration SQL)
- Modify: `internal/db/content.go` (all SQL queries)

**Step 1: Add columns to CREATE TABLE in migrationSQL**

In `internal/db/db.go`, update the `content_sessions` table definition:
```sql
CREATE TABLE IF NOT EXISTS content_sessions (
    id               TEXT PRIMARY KEY,
    pipeline_id      TEXT NOT NULL,
    name             TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'collecting',
    trigger_type     TEXT NOT NULL DEFAULT 'manual',
    source_count     INTEGER NOT NULL DEFAULT 0,
    is_template      BOOLEAN NOT NULL DEFAULT false,
    parent_session_id TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at      TIMESTAMPTZ,
    archived_at      TIMESTAMPTZ
);
```

Also add ALTER TABLE migration after table creation to handle existing DBs:
```sql
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS is_template BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS parent_session_id TEXT NOT NULL DEFAULT '';
```

**Step 2: Update CreateContentSession**

In `internal/db/content.go:14`, add the two columns to INSERT:
```go
func (d *DB) CreateContentSession(ctx context.Context, s *upal.ContentSession) error {
	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO content_sessions (id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		s.ID, s.PipelineID, s.Name, string(s.Status), s.TriggerType, s.SourceCount, s.IsTemplate, s.ParentSessionID, s.CreatedAt, s.ReviewedAt, s.ArchivedAt,
	)
	if err != nil {
		return fmt.Errorf("insert content_session: %w", err)
	}
	return nil
}
```

**Step 3: Update all SELECT scan functions**

Every `GetContentSession`, `ListContentSessions`, `ListContentSessionsByPipeline`, `ListContentSessionsByStatus`, `ListAllContentSessionsByStatus`, `ListContentSessionsByPipelineAndStatus`, `ListArchivedContentSessionsByPipeline` needs to:

1. Add `is_template, parent_session_id` to the SELECT column list
2. Add `&s.IsTemplate, &s.ParentSessionID` to the Scan call

Example for `GetContentSession`:
```go
func (d *DB) GetContentSession(ctx context.Context, id string) (*upal.ContentSession, error) {
	var s upal.ContentSession
	var status string
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.PipelineID, &s.Name, &status, &s.TriggerType, &s.SourceCount, &s.IsTemplate, &s.ParentSessionID, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt)
	// ... rest unchanged
}
```

Apply the same pattern to all 7 list/get functions.

**Step 4: Update UpdateContentSession**

```go
func (d *DB) UpdateContentSession(ctx context.Context, s *upal.ContentSession) error {
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE content_sessions SET name = $1, status = $2, source_count = $3, reviewed_at = $4, archived_at = $5, is_template = $6, parent_session_id = $7 WHERE id = $8`,
		s.Name, string(s.Status), s.SourceCount, s.ReviewedAt, s.ArchivedAt, s.IsTemplate, s.ParentSessionID, s.ID,
	)
	// ... rest unchanged
}
```

**Step 5: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/db/db.go internal/db/content.go
git commit -m "feat: add is_template and parent_session_id columns to content_sessions"
```

---

### Task 3: Memory Repository — Template filtering

**Files:**
- Modify: `internal/repository/content.go:10-21` (interface)
- Modify: `internal/repository/content_memory.go`
- Modify: `internal/repository/content_persistent.go`

**Step 1: Add ListTemplatesByPipeline to repository interface**

```go
type ContentSessionRepository interface {
	// ... existing methods ...
	ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
}
```

**Step 2: Implement in memory repository**

```go
func (r *MemoryContentSessionRepository) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.IsTemplate && s.ArchivedAt == nil
	})
}
```

**Step 3: Update existing ListByPipeline to exclude templates**

```go
func (r *MemoryContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && !s.IsTemplate && s.ArchivedAt == nil
	})
}
```

**Step 4: Update ListByStatus, ListAllByStatus to exclude templates**

Add `!s.IsTemplate` to existing filter functions:
```go
func (r *MemoryContentSessionRepository) ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.Status == status && !s.IsTemplate && s.ArchivedAt == nil
	})
}
```

Same for `ListAllByStatus` and `ListByPipelineAndStatus`.

**Step 5: Implement in persistent repository**

```go
func (r *PersistentContentSessionRepository) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.mem.ListTemplatesByPipeline(ctx, pipelineID)
}
```

**Step 6: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/repository/content.go internal/repository/content_memory.go internal/repository/content_persistent.go
git commit -m "feat: add template filtering to content session repositories"
```

---

### Task 4: DB Repository — Template-aware queries

**Files:**
- Modify: `internal/db/content.go`

**Step 1: Add ListTemplatesByPipeline DB method**

```go
func (d *DB) ListTemplateContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, name, status, trigger_type, source_count, is_template, parent_session_id, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE pipeline_id = $1 AND is_template = true AND archived_at IS NULL ORDER BY created_at DESC`,
		pipelineID,
	)
	// ... standard scan pattern
}
```

**Step 2: Update existing ListContentSessionsByStatus to filter out templates**

Add `AND is_template = false` to all status-based queries:
- `ListContentSessionsByStatus`
- `ListAllContentSessionsByStatus`
- `ListContentSessionsByPipelineAndStatus`

**Step 3: Update ListContentSessionsByPipeline to filter out templates**

Add `AND is_template = false` — this query returns instances within a pipeline (used by inbox when filtering by pipeline).

**Step 4: Wire into PersistentContentSessionRepository**

The persistent repo delegates to memory for reads (cache-first). The DB method is used when loading from DB. No additional wiring needed beyond the Sync/Load logic already present.

**Step 5: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/db/content.go
git commit -m "feat: add template-aware DB queries for content sessions"
```

---

### Task 5: Service Layer — Template-aware session creation

**Files:**
- Modify: `internal/services/content_session_service.go`

**Step 1: Update CreateSession to support IsTemplate**

The `CreateSession` method (line ~49) should preserve the `IsTemplate` and `ParentSessionID` values set by the caller. No changes needed here — the caller sets these fields before calling `CreateSession`.

**Step 2: Add ListTemplatesByPipeline service method**

```go
func (s *ContentSessionService) ListTemplatesByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListTemplatesByPipeline(ctx, pipelineID)
}
```

**Step 3: Add ListTemplateDetailsByPipeline for composed responses**

Similar to existing `ListSessionDetails`, but queries templates instead:
```go
func (s *ContentSessionService) ListTemplateDetailsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListTemplatesByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	// compose details (same pattern as ListSessionDetails)
	// ...
}
```

**Step 4: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/services/content_session_service.go
git commit -m "feat: add template-aware service methods"
```

---

### Task 6: API — createDraftSession creates template

**Files:**
- Modify: `internal/api/content.go:20-47`

**Step 1: Set IsTemplate=true in createDraftSession**

```go
sess := &upal.ContentSession{
	PipelineID:  body.PipelineID,
	Name:        body.Name,
	Status:      upal.SessionDraft,
	TriggerType: "manual",
	IsTemplate:  true,  // NEW
}
```

**Step 2: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/api/content.go
git commit -m "feat: createDraftSession now creates template sessions"
```

---

### Task 7: API — collectSession always creates instance

**Files:**
- Modify: `internal/api/content.go:602-651`

**Step 1: Change collectSession to always create a new instance**

Remove the dual-mode (draft vs non-draft) logic. Always create a child instance:

```go
func (s *Server) collectSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Always create a new instance from this template/session.
	execSess := &upal.ContentSession{
		PipelineID:      session.PipelineID,
		Name:            session.Name,
		TriggerType:     "manual",
		IsTemplate:      false,
		ParentSessionID: session.ID,
		Sources:         session.Sources,
		Model:           session.Model,
		Workflows:       session.Workflows,
		Context:         session.Context,
	}
	if err := s.contentSvc.CreateSession(r.Context(), execSess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), execSess, body.IsTest, body.Limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": execSess.ID})
}
```

**Step 2: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/api/content.go
git commit -m "feat: collectSession always creates instance from template"
```

---

### Task 8: API — listContentSessions template filtering

**Files:**
- Modify: `internal/api/content.go:52-123`

**Step 1: When pipeline_id is provided, return templates by default**

Update the `listContentSessions` handler. When `pipeline_id` is set, use `ListTemplateDetailsByPipeline` instead of `ListSessionDetails`:

```go
if pipelineID != "" {
	archivedOnly := r.URL.Query().Get("archived_only") == "true"
	templatesOnly := r.URL.Query().Get("templates") != "false" // default true for pipeline queries

	var details []*upal.ContentSessionDetail
	var err error
	if archivedOnly {
		details, err = s.contentSvc.ListArchivedSessionDetails(ctx, pipelineID)
	} else if templatesOnly {
		details, err = s.contentSvc.ListTemplateDetailsByPipeline(ctx, pipelineID)
	} else {
		details, err = s.contentSvc.ListSessionDetails(ctx, pipelineID)
	}
	// ... rest unchanged
}
```

**Step 2: Status-based queries already exclude templates (from Task 3)**

No additional changes needed — `ListByStatus` already filters `!s.IsTemplate`.

**Step 3: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/api/content.go
git commit -m "feat: pipeline session list returns templates by default"
```

---

### Task 9: Collector — collectPipeline creates instance

**Files:**
- Modify: `internal/services/content_collector.go:69-88`
- Modify: `internal/api/content.go:557-597`

**Step 1: Update CollectPipeline in collector**

```go
func (c *ContentCollector) CollectPipeline(ctx context.Context, pipelineID string) error {
	pipeline, err := c.pipelineRepo.Get(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("pipeline %s: %w", pipelineID, err)
	}
	sess := &upal.ContentSession{
		PipelineID:  pipelineID,
		TriggerType: "scheduled",
		IsTemplate:  false,  // NEW: explicitly instance
		Sources:     pipeline.Sources,
		Schedule:    pipeline.Schedule,
		Model:       pipeline.Model,
		Workflows:   pipeline.Workflows,
		Context:     pipeline.Context,
	}
	if err := c.contentSvc.CreateSession(ctx, sess); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	go c.CollectAndAnalyze(context.Background(), sess, false, 0)
	return nil
}
```

**Step 2: Update collectPipeline API handler similarly**

Add `IsTemplate: false` to the session created in `collectPipeline` handler (line ~579).

**Step 3: Run tests**

Run: `go build ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/services/content_collector.go internal/api/content.go
git commit -m "feat: pipeline collect creates explicit instance sessions"
```

---

### Task 10: Frontend Types — Add template fields

**Files:**
- Modify: `web/src/entities/content-session/types.ts`

**Step 1: Add fields to ContentSession type**

```typescript
export type ContentSession = {
  id: string
  pipeline_id: string
  name?: string
  pipeline_name?: string
  session_number?: number
  trigger_type: 'schedule' | 'manual' | 'surge'
  status: ContentSessionStatus
  is_template?: boolean           // NEW
  parent_session_id?: string      // NEW
  // ... rest unchanged
}
```

**Step 2: Commit**

```bash
git add web/src/entities/content-session/types.ts
git commit -m "feat: add is_template and parent_session_id to frontend types"
```

---

### Task 11: Frontend — SessionListPanel shows templates only

**Files:**
- Modify: `web/src/pages/pipelines/SessionListPanel.tsx`

**Step 1: Clean up status dot (templates don't need it)**

Remove the `STATUS_DOT` constant and status dot rendering. Templates are always draft — no visual status needed.

**Step 2: Remove status-related search matching**

Update the search filter to remove `s.status.includes(q)` since templates don't have meaningful status.

**Step 3: Remove auto-select logic for pending_review**

Templates don't have pending_review status. Simplify to just select the first session:
```typescript
useEffect(() => {
  if (!selectedSessionId && filteredSessions.length > 0) {
    onSelectSession(filteredSessions[0].id)
  }
}, [filteredSessions, selectedSessionId, onSelectSession])
```

**Step 4: Commit**

```bash
git add web/src/pages/pipelines/SessionListPanel.tsx
git commit -m "feat: simplify SessionListPanel for template-only display"
```

---

### Task 12: Frontend — SessionSetupView collect creates instance

**Files:**
- Modify: `web/src/pages/pipelines/session/SessionSetupView.tsx`

**Step 1: Update the collect button behavior**

After triggering collect, show a toast/notification saying "수집을 시작했습니다. Inbox에서 확인하세요" instead of navigating to the instance. The template stays selected.

Find the collect trigger handler and update:
```typescript
// After successful collect API call:
toast.success('수집을 시작했습니다. Inbox에서 결과를 확인하세요.')
// Do NOT navigate away — template stays open
```

**Step 2: Commit**

```bash
git add web/src/pages/pipelines/session/SessionSetupView.tsx
git commit -m "feat: collect from template shows inbox redirect toast"
```

---

### Task 13: Data Migration — Mark existing sessions

**Files:**
- Modify: `internal/db/db.go` (add migration SQL)

**Step 1: Add migration to classify existing sessions**

Add after the ALTER TABLE statements:
```sql
-- Migrate: existing draft sessions become templates
UPDATE content_sessions SET is_template = true WHERE status = 'draft' AND is_template = false;
```

This is safe because:
- Existing draft sessions are the ones users configured as reusable setups
- Non-draft sessions are already execution instances

**Step 2: Run full test suite**

Run: `make test`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat: migrate existing draft sessions to templates"
```

---

### Task 14: Integration Test — Full flow verification

**Step 1: Manual verification**

1. Start dev environment: `make dev-backend` + `make dev-frontend`
2. Open pipeline page → verify only template sessions show (no status badges)
3. Create new session → verify `is_template=true` in response
4. Click "수집 시작" on template → verify new instance created (check API response)
5. Open Inbox → verify instance appears with `pending_review` status
6. Approve in Inbox → verify moves to Publish Inbox

**Step 2: Run full test suite**

Run: `make test && make test-frontend`
Expected: PASS

**Step 3: Commit any fixes**

```bash
git commit -m "fix: address integration test findings"
```
