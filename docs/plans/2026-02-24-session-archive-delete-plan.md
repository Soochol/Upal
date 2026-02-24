# Session Archive & Delete Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add two-step session lifecycle management — archive (soft-delete with `archived_at` timestamp) and permanent delete (only from archived state).

**Architecture:** `archived_at TIMESTAMPTZ` column on `content_sessions`, orthogonal to `status`. List queries exclude archived by default. Delete requires archived guard. Clean architecture layers: domain → repository → service → API → frontend.

**Tech Stack:** Go 1.23, PostgreSQL, React 19 + TanStack Query, Zustand, Tailwind CSS v4

**Design doc:** `docs/plans/2026-02-24-session-archive-delete-design.md`

---

### Task 1: Domain Type — Add `ArchivedAt` field

**Files:**
- Modify: `internal/upal/content.go:22-30` (ContentSession struct)
- Modify: `internal/upal/content.go:124-137` (ContentSessionDetail struct)

**Step 1: Add ArchivedAt to ContentSession**

In `internal/upal/content.go`, add `ArchivedAt` field to `ContentSession`:

```go
type ContentSession struct {
	ID          string               `json:"id"`
	PipelineID  string               `json:"pipeline_id"`
	Status      ContentSessionStatus `json:"status"`
	TriggerType string               `json:"trigger_type"`
	SourceCount int                  `json:"source_count"`
	CreatedAt   time.Time            `json:"created_at"`
	ReviewedAt  *time.Time           `json:"reviewed_at,omitempty"`
	ArchivedAt  *time.Time           `json:"archived_at,omitempty"`
}
```

**Step 2: Add ArchivedAt to ContentSessionDetail**

In the same file, add `ArchivedAt` field to `ContentSessionDetail`:

```go
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
```

**Step 3: Run tests to verify no regressions**

Run: `go test ./internal/upal/... -v -race`
Expected: PASS (field additions are backward-compatible)

**Step 4: Commit**

```bash
git add internal/upal/content.go
git commit -m "feat(content): add ArchivedAt field to ContentSession and ContentSessionDetail"
```

---

### Task 2: DB Layer — Schema migration + archive/delete queries

**Files:**
- Modify: `internal/db/db.go` (migration SQL, add `archived_at` column)
- Modify: `internal/db/content.go` (new methods + update existing List queries)

**Step 1: Add migration for archived_at column**

In `internal/db/db.go`, append to `migrationSQL` (before the closing backtick):

```sql
ALTER TABLE content_sessions ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;
```

**Step 2: Update existing List queries to exclude archived by default**

In `internal/db/content.go`, update these query methods to add `WHERE archived_at IS NULL`:

`ListContentSessions`:
```sql
SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at
FROM content_sessions WHERE archived_at IS NULL ORDER BY created_at DESC
```

`ListContentSessionsByPipeline`:
```sql
SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at
FROM content_sessions WHERE pipeline_id = $1 AND archived_at IS NULL ORDER BY created_at DESC
```

`ListContentSessionsByStatus`:
```sql
SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at
FROM content_sessions WHERE status = $1 AND archived_at IS NULL ORDER BY created_at DESC
```

`ListContentSessionsByPipelineAndStatus`:
```sql
SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at
FROM content_sessions WHERE pipeline_id = $1 AND status = $2 AND archived_at IS NULL ORDER BY created_at DESC
```

Also update `CreateContentSession` and `GetContentSession` and `UpdateContentSession` to include `archived_at`:

`CreateContentSession`:
```sql
INSERT INTO content_sessions (id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
```

`GetContentSession` SELECT:
```sql
SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at
FROM content_sessions WHERE id = $1
```

`UpdateContentSession`:
```sql
UPDATE content_sessions SET status = $1, source_count = $2, reviewed_at = $3, archived_at = $4 WHERE id = $5
```

All `Scan` calls must include `&s.ArchivedAt`.

**Step 3: Add new DB methods**

Add to `internal/db/content.go`:

```go
func (d *DB) ListArchivedContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, trigger_type, source_count, created_at, reviewed_at, archived_at
		 FROM content_sessions WHERE pipeline_id = $1 AND archived_at IS NOT NULL ORDER BY archived_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list archived content_sessions: %w", err)
	}
	defer rows.Close()
	var result []*upal.ContentSession
	for rows.Next() {
		var s upal.ContentSession
		var status string
		if err := rows.Scan(&s.ID, &s.PipelineID, &status, &s.TriggerType, &s.SourceCount, &s.CreatedAt, &s.ReviewedAt, &s.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan content_session: %w", err)
		}
		s.Status = upal.ContentSessionStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (d *DB) DeleteContentSession(ctx context.Context, id string) error {
	res, err := d.Pool.ExecContext(ctx,
		`DELETE FROM content_sessions WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("delete content_session: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("content session %q not found", id)
	}
	return nil
}

func (d *DB) DeletePublishedContentBySession(ctx context.Context, sessionID string) error {
	_, err := d.Pool.ExecContext(ctx,
		`DELETE FROM published_content WHERE session_id = $1`, sessionID,
	)
	if err != nil {
		return fmt.Errorf("delete published_content by session: %w", err)
	}
	return nil
}
```

**Step 4: Run existing tests**

Run: `go test ./internal/db/... -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/db/db.go internal/db/content.go
git commit -m "feat(db): add archived_at column, archive/delete queries, exclude archived from default lists"
```

---

### Task 3: Repository Interface & Memory Implementation

**Files:**
- Modify: `internal/repository/content.go:10-18` (interface)
- Modify: `internal/repository/content_memory.go:14-66` (memory impl)
- Modify: `internal/repository/content_persistent.go:13-35` (ContentDB interface)
- Modify: `internal/repository/content_persistent.go:37-109` (persistent impl)

**Step 1: Extend ContentSessionRepository interface**

In `internal/repository/content.go`, add to `ContentSessionRepository`:

```go
type ContentSessionRepository interface {
	Create(ctx context.Context, s *upal.ContentSession) error
	Get(ctx context.Context, id string) (*upal.ContentSession, error)
	List(ctx context.Context) ([]*upal.ContentSession, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error)
	ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
	Update(ctx context.Context, s *upal.ContentSession) error
	Delete(ctx context.Context, id string) error
}
```

Add to `PublishedContentRepository`:

```go
type PublishedContentRepository interface {
	Create(ctx context.Context, pc *upal.PublishedContent) error
	List(ctx context.Context) ([]*upal.PublishedContent, error)
	ListBySession(ctx context.Context, sessionID string) ([]*upal.PublishedContent, error)
	ListByChannel(ctx context.Context, channel string) ([]*upal.PublishedContent, error)
	DeleteBySession(ctx context.Context, sessionID string) error
}
```

**Step 2: Implement memory repository methods**

In `internal/repository/content_memory.go`, update `List` to exclude archived:

```go
func (r *MemoryContentSessionRepository) List(ctx context.Context) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.ArchivedAt == nil
	})
}
```

Update `ListByPipeline` to exclude archived:

```go
func (r *MemoryContentSessionRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.ArchivedAt == nil
	})
}
```

Update `ListByStatus` to exclude archived:

```go
func (r *MemoryContentSessionRepository) ListByStatus(ctx context.Context, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.Status == status && s.ArchivedAt == nil
	})
}
```

Update `ListByPipelineAndStatus` to exclude archived:

```go
func (r *MemoryContentSessionRepository) ListByPipelineAndStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.Status == status && s.ArchivedAt == nil
	})
}
```

Add `ListArchivedByPipeline`:

```go
func (r *MemoryContentSessionRepository) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return r.store.Filter(ctx, func(s *upal.ContentSession) bool {
		return s.PipelineID == pipelineID && s.ArchivedAt != nil
	})
}
```

Add `Delete`:

```go
func (r *MemoryContentSessionRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("content session %q not found", id)
	}
	return err
}
```

Add `DeleteBySession` to `MemoryPublishedContentRepository`:

```go
func (r *MemoryPublishedContentRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	all, _ := r.store.All(ctx)
	for _, pc := range all {
		if pc.SessionID == sessionID {
			_ = r.store.Delete(ctx, pc.ID)
		}
	}
	return nil
}
```

**Step 3: Extend ContentDB interface and persistent repo**

In `internal/repository/content_persistent.go`, add to `ContentDB` interface:

```go
ListArchivedContentSessionsByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error)
DeleteContentSession(ctx context.Context, id string) error
DeletePublishedContentBySession(ctx context.Context, sessionID string) error
```

Add persistent repo methods:

```go
func (r *PersistentContentSessionRepository) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	sessions, err := r.db.ListArchivedContentSessionsByPipeline(ctx, pipelineID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list archived content_sessions failed, falling back to in-memory", "err", err)
	return r.mem.ListArchivedByPipeline(ctx, pipelineID)
}

func (r *PersistentContentSessionRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeleteContentSession(ctx, id); err != nil {
		return fmt.Errorf("db delete content_session: %w", err)
	}
	return nil
}
```

Add to `PersistentPublishedContentRepository`:

```go
func (r *PersistentPublishedContentRepository) DeleteBySession(ctx context.Context, sessionID string) error {
	_ = r.mem.DeleteBySession(ctx, sessionID)
	if err := r.db.DeletePublishedContentBySession(ctx, sessionID); err != nil {
		return fmt.Errorf("db delete published_content by session: %w", err)
	}
	return nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/repository/... -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/repository/content.go internal/repository/content_memory.go internal/repository/content_persistent.go
git commit -m "feat(repo): add ListArchivedByPipeline, Delete, DeleteBySession to content repositories"
```

---

### Task 4: Service Layer — Archive, Unarchive, Delete

**Files:**
- Modify: `internal/services/content_session_service.go` (new methods)
- Modify: `internal/services/content_session_service.go:269-282` (GetSessionDetail — pass through ArchivedAt)

**Step 1: Write failing tests**

Add to `internal/services/content_session_service_test.go`:

```go
func TestContentSessionService_ArchiveAndUnarchive(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	if err := svc.CreateSession(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Archive
	if err := svc.ArchiveSession(ctx, s.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	got, _ := svc.GetSession(ctx, s.ID)
	if got.ArchivedAt == nil {
		t.Error("expected ArchivedAt to be set after archive")
	}
	// Original status preserved
	if got.Status != upal.SessionCollecting {
		t.Errorf("expected status preserved as collecting, got %q", got.Status)
	}

	// Archived sessions excluded from ListByPipeline
	list, _ := svc.ListSessionsByPipeline(ctx, "pipe-1")
	if len(list) != 0 {
		t.Errorf("expected 0 active sessions, got %d", len(list))
	}

	// Listed in archived
	archived, _ := svc.ListArchivedByPipeline(ctx, "pipe-1")
	if len(archived) != 1 {
		t.Errorf("expected 1 archived session, got %d", len(archived))
	}

	// Unarchive
	if err := svc.UnarchiveSession(ctx, s.ID); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	got, _ = svc.GetSession(ctx, s.ID)
	if got.ArchivedAt != nil {
		t.Error("expected ArchivedAt to be nil after unarchive")
	}

	list, _ = svc.ListSessionsByPipeline(ctx, "pipe-1")
	if len(list) != 1 {
		t.Errorf("expected 1 active session after unarchive, got %d", len(list))
	}
}

func TestContentSessionService_DeleteRequiresArchived(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)

	// Delete without archiving should fail
	err := svc.DeleteSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when deleting non-archived session")
	}

	// Archive then delete should succeed
	svc.ArchiveSession(ctx, s.ID)
	if err := svc.DeleteSession(ctx, s.ID); err != nil {
		t.Fatalf("delete archived session: %v", err)
	}

	// Session should no longer exist
	_, err = svc.GetSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when getting deleted session")
	}
}

func TestContentSessionService_ArchiveAlreadyArchived(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)
	svc.ArchiveSession(ctx, s.ID)

	// Archiving again should fail
	err := svc.ArchiveSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when archiving already archived session")
	}
}

func TestContentSessionService_UnarchiveNotArchived(t *testing.T) {
	svc := newTestContentSvc()
	ctx := context.Background()

	s := &upal.ContentSession{PipelineID: "pipe-1", TriggerType: "manual"}
	svc.CreateSession(ctx, s)

	// Unarchiving non-archived should fail
	err := svc.UnarchiveSession(ctx, s.ID)
	if err == nil {
		t.Error("expected error when unarchiving non-archived session")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/... -v -race -run "TestContentSessionService_(Archive|Delete)"`
Expected: FAIL (methods don't exist yet)

**Step 3: Implement service methods**

Add to `internal/services/content_session_service.go`:

```go
func (s *ContentSessionService) ArchiveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt != nil {
		return fmt.Errorf("session %q is already archived", id)
	}
	now := time.Now()
	sess.ArchivedAt = &now
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) UnarchiveSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt == nil {
		return fmt.Errorf("session %q is not archived", id)
	}
	sess.ArchivedAt = nil
	return s.sessions.Update(ctx, sess)
}

func (s *ContentSessionService) DeleteSession(ctx context.Context, id string) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if sess.ArchivedAt == nil {
		return fmt.Errorf("session %q must be archived before deletion", id)
	}

	// Clean up published_content (no FK cascade)
	if err := s.published.DeleteBySession(ctx, id); err != nil {
		return fmt.Errorf("delete published content: %w", err)
	}

	// Delete session (source_fetches + llm_analyses cascade in DB)
	if err := s.sessions.Delete(ctx, id); err != nil {
		return err
	}

	// Clean up in-memory workflow results
	s.mu.Lock()
	delete(s.workflowResults, id)
	s.mu.Unlock()

	return nil
}

func (s *ContentSessionService) ListArchivedByPipeline(ctx context.Context, pipelineID string) ([]*upal.ContentSession, error) {
	return s.sessions.ListArchivedByPipeline(ctx, pipelineID)
}
```

Update `GetSessionDetail` to pass through `ArchivedAt`. In the return struct at the bottom, add:

```go
ArchivedAt:      sess.ArchivedAt,
```

Update `ListSessionDetails` similarly — in the `details = append(details, ...)` block, add:

```go
ArchivedAt:      sess.ArchivedAt,
```

**Step 4: Run tests**

Run: `go test ./internal/services/... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/services/content_session_service.go internal/services/content_session_service_test.go
git commit -m "feat(service): add ArchiveSession, UnarchiveSession, DeleteSession with guard logic"
```

---

### Task 5: API Handlers — Archive, Unarchive, Delete endpoints

**Files:**
- Modify: `internal/api/content.go` (new handlers)
- Modify: `internal/api/server.go:113-122` (route registration)
- Modify: `internal/api/content.go:16-44` (listContentSessions — add archived_only param)

**Step 1: Add archive/unarchive/delete handlers**

Add to `internal/api/content.go`:

```go
// POST /api/content-sessions/{id}/archive
func (s *Server) archiveContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.ArchiveSession(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "already archived") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	sess, _ := s.contentSvc.GetSession(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// POST /api/content-sessions/{id}/unarchive
func (s *Server) unarchiveContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.UnarchiveSession(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "not archived") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	sess, _ := s.contentSvc.GetSession(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// DELETE /api/content-sessions/{id}
func (s *Server) deleteContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DeleteSession(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "must be archived") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

**Step 2: Update listContentSessions to support `archived_only` param**

In `listContentSessions`, when `pipelineID != ""`, add handling for `archived_only`:

```go
if pipelineID != "" {
	archivedOnly := r.URL.Query().Get("archived_only") == "true"
	if archivedOnly {
		sessions, err := s.contentSvc.ListArchivedByPipeline(ctx, pipelineID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Build detail views for archived sessions
		// ... (compose detail views same as non-archived path)
	}
	// existing path for non-archived
}
```

Actually, the simpler approach: extend `ListSessionDetails` in the service to accept an `archivedOnly` parameter, or add a separate method. But to keep it clean, let's add a dedicated list method in the service and call it here.

Add to `internal/services/content_session_service.go`:

```go
func (s *ContentSessionService) ListArchivedSessionDetails(ctx context.Context, pipelineID string) ([]*upal.ContentSessionDetail, error) {
	sessions, err := s.sessions.ListArchivedByPipeline(ctx, pipelineID)
	if err != nil {
		return nil, err
	}
	// Same composition logic as ListSessionDetails
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.Before(sessions[j].CreatedAt)
	})
	details := make([]*upal.ContentSessionDetail, 0, len(sessions))
	for i, sess := range sessions {
		sources, err := s.fetches.ListBySession(ctx, sess.ID)
		if err != nil {
			return nil, fmt.Errorf("list sources for session %s: %w", sess.ID, err)
		}
		analysis, _ := s.analyses.GetBySession(ctx, sess.ID)
		wfResults := s.GetWorkflowResults(ctx, sess.ID)
		details = append(details, &upal.ContentSessionDetail{
			ID:              sess.ID,
			PipelineID:      sess.PipelineID,
			SessionNumber:   i + 1,
			Status:          sess.Status,
			TriggerType:     sess.TriggerType,
			SourceCount:     sess.SourceCount,
			Sources:         sources,
			Analysis:        analysis,
			WorkflowResults: wfResults,
			CreatedAt:       sess.CreatedAt,
			ReviewedAt:      sess.ReviewedAt,
			ArchivedAt:      sess.ArchivedAt,
		})
	}
	sort.Slice(details, func(i, j int) bool {
		return details[i].CreatedAt.After(details[j].CreatedAt)
	})
	return details, nil
}
```

Update `listContentSessions` handler — add before the existing `pipelineID != ""` block:

```go
if pipelineID != "" {
	archivedOnly := r.URL.Query().Get("archived_only") == "true"

	var details []*upal.ContentSessionDetail
	var err error
	if archivedOnly {
		details, err = s.contentSvc.ListArchivedSessionDetails(ctx, pipelineID)
	} else {
		details, err = s.contentSvc.ListSessionDetails(ctx, pipelineID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// ... rest of status filter + JSON encode
}
```

**Step 3: Register routes**

In `internal/api/server.go`, add to the content-sessions route group:

```go
r.Route("/content-sessions", func(r chi.Router) {
	r.Get("/", s.listContentSessions)
	r.Get("/{id}", s.getContentSession)
	r.Patch("/{id}", s.patchContentSession)
	r.Delete("/{id}", s.deleteContentSession)           // NEW
	r.Post("/{id}/archive", s.archiveContentSession)     // NEW
	r.Post("/{id}/unarchive", s.unarchiveContentSession) // NEW
	r.Post("/{id}/produce", s.produceContentSession)
	r.Get("/{id}/sources", s.listSessionSources)
	r.Get("/{id}/analysis", s.getSessionAnalysis)
	r.Patch("/{id}/analysis", s.patchSessionAnalysis)
	r.Post("/{id}/publish", s.publishContentSession)
})
```

**Step 4: Run all Go tests**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/api/content.go internal/api/server.go internal/services/content_session_service.go
git commit -m "feat(api): add archive, unarchive, delete endpoints for content sessions"
```

---

### Task 6: Frontend — Types, API client, store updates

**Files:**
- Modify: `web/src/entities/content-session/types.ts:47-59` (add `archived_at`)
- Modify: `web/src/entities/content-session/api.ts` (add archive/unarchive/delete functions + `archived_only` param)
- Modify: `web/src/entities/content-session/store.ts` (add archive/delete actions)

**Step 1: Add archived_at to ContentSession type**

In `web/src/entities/content-session/types.ts`, add to `ContentSession`:

```ts
export type ContentSession = {
  id: string
  pipeline_id: string
  pipeline_name?: string
  session_number?: number
  trigger_type: 'schedule' | 'manual' | 'surge'
  status: ContentSessionStatus
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_results?: WorkflowResult[]
  created_at: string
  updated_at?: string
  archived_at?: string  // NEW
}
```

**Step 2: Add API functions**

In `web/src/entities/content-session/api.ts`, update `fetchContentSessions` to support `archivedOnly`:

```ts
export async function fetchContentSessions(params?: {
  pipelineId?: string
  status?: string
  archivedOnly?: boolean
}): Promise<ContentSession[]> {
  const qs = new URLSearchParams()
  if (params?.pipelineId) qs.set('pipeline_id', params.pipelineId)
  if (params?.status) qs.set('status', params.status)
  if (params?.archivedOnly) qs.set('archived_only', 'true')
  const query = qs.toString() ? `?${qs}` : ''
  return apiFetch<ContentSession[]>(`${BASE}${query}`)
}
```

Add new functions:

```ts
export async function archiveSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/archive`, {
    method: 'POST',
  })
}

export async function unarchiveSession(id: string): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/unarchive`, {
    method: 'POST',
  })
}

export async function deleteSession(id: string): Promise<void> {
  await apiFetch(`${BASE}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
}
```

**Step 3: Update re-exports if needed**

Check `web/src/entities/content-session/index.ts` and ensure new API functions are exported if the barrel file re-exports from api.ts.

**Step 4: Run type check**

Run: `cd web && npx tsc -b`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/entities/content-session/
git commit -m "feat(frontend): add archived_at type, archive/unarchive/delete API functions"
```

---

### Task 7: Frontend UI — Archive filter tab, archive/delete actions in PipelineDetail

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx` (filter tabs, session card actions, archive/delete mutations)
- Modify: `web/src/pages/pipelines/session/SessionDetailPreview.tsx` (archive button in detail view)

**Step 1: Update SessionFilter type and filter tabs**

In `web/src/pages/pipelines/PipelineDetail.tsx`, update:

```ts
type SessionFilter = 'all' | 'pending_review' | 'producing' | 'published' | 'rejected' | 'archived'
```

Update `filterTabs` array to add archived tab:

```ts
const filterTabs: { value: SessionFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending_review', label: 'Pending' },
  { value: 'producing', label: 'Producing' },
  { value: 'published', label: 'Published' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'archived', label: 'Archived' },
]
```

**Step 2: Add archived sessions query**

Add a separate query for archived sessions:

```ts
const { data: archivedSessions = [] } = useQuery({
  queryKey: ['content-sessions', { pipelineId: id, archived: true }],
  queryFn: () => fetchContentSessions({ pipelineId: id, archivedOnly: true }),
  enabled: !!id && activeFilter === 'archived',
})
```

Update `filterCounts` to include archived:

```ts
const filterCounts = useMemo(() => {
  const counts: Record<SessionFilter, number> = {
    all: sessions.length,
    pending_review: 0,
    producing: 0,
    published: 0,
    rejected: 0,
    archived: archivedSessions.length,
  }
  for (const s of sessions) {
    if (s.status in counts) counts[s.status as SessionFilter]++
  }
  return counts
}, [sessions, archivedSessions])
```

Update `filteredSessions` to use archived sessions when filter is 'archived':

```ts
const filteredSessions = (activeFilter === 'archived' ? archivedSessions : sessions)
  .filter(s => activeFilter === 'all' || activeFilter === 'archived' || s.status === activeFilter)
  .filter(s => {
    if (!search) return true
    const q = search.toLowerCase()
    return `session ${s.session_number}`.includes(q) ||
      s.analysis?.summary?.toLowerCase().includes(q) ||
      s.status.includes(q)
  })
```

**Step 3: Add archive/delete mutations**

Import `archiveSession`, `unarchiveSession`, `deleteSession` from the API module.

```ts
import { fetchContentSessions, archiveSession as archiveSessionApi, unarchiveSession as unarchiveSessionApi, deleteSession as deleteSessionApi } from '@/entities/content-session/api'
```

Add mutations:

```ts
const archiveMutation = useMutation({
  mutationFn: (sessionId: string) => archiveSessionApi(sessionId),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id, archived: true }] })
  },
})

const unarchiveMutation = useMutation({
  mutationFn: (sessionId: string) => unarchiveSessionApi(sessionId),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id, archived: true }] })
  },
})

const deleteMutation = useMutation({
  mutationFn: (sessionId: string) => deleteSessionApi(sessionId),
  onSuccess: () => {
    setSelectedSessionId(null)
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id, archived: true }] })
  },
})
```

**Step 4: Add archive action to session card**

In the session list item `<button>`, add a context action. Add an `Archive` icon import:

```ts
import { Archive, ArchiveRestore } from 'lucide-react'
```

In the session card, after the `<StatusBadge>` row, add an archive/unarchive/delete button row when hovering or for archived sessions:

For non-archived sessions, add a small archive button:

```tsx
<button
  onClick={(e) => {
    e.stopPropagation()
    archiveMutation.mutate(s.id)
  }}
  className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-foreground transition-all cursor-pointer"
  title="Archive"
>
  <Archive className="h-3 w-3" />
</button>
```

For archived sessions (when `activeFilter === 'archived'`), show unarchive and delete buttons:

```tsx
{s.archived_at && (
  <div className="flex items-center gap-1 ml-auto">
    <button
      onClick={(e) => { e.stopPropagation(); unarchiveMutation.mutate(s.id) }}
      className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
      title="Unarchive"
    >
      <ArchiveRestore className="h-3 w-3" />
    </button>
    <button
      onClick={(e) => {
        e.stopPropagation()
        if (confirm('Permanently delete this session? This cannot be undone.')) {
          deleteMutation.mutate(s.id)
        }
      }}
      className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
      title="Delete permanently"
    >
      <Trash2 className="h-3 w-3" />
    </button>
  </div>
)}
```

Make the session card button a `group` for hover effects:

```tsx
<button
  key={s.id}
  onClick={() => setSelectedSessionId(s.id)}
  className={`group w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border ...`}
>
```

**Step 5: Run type check and lint**

Run: `cd web && npx tsc -b && npm run lint`
Expected: No errors

**Step 6: Commit**

```bash
git add web/src/pages/pipelines/PipelineDetail.tsx web/src/pages/pipelines/session/SessionDetailPreview.tsx
git commit -m "feat(ui): add Archive tab, archive/unarchive/delete actions to session list"
```

---

### Task 8: Frontend UI — Archive button in SessionDetailPreview

**Files:**
- Modify: `web/src/pages/pipelines/session/SessionDetailPreview.tsx`

**Step 1: Add archive/unarchive action in the detail preview**

Add an archive button to the `SessionDetailPreview` component header area. Import the needed functions:

```ts
import { archiveSession as archiveSessionApi } from '@/entities/content-session/api'
import { useMutation } from '@tanstack/react-query'
import { Archive } from 'lucide-react'
```

Add the mutation inside the component:

```ts
const archiveMutation = useMutation({
  mutationFn: () => archiveSessionApi(sessionId),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId, archived: true }] })
  },
})
```

Add an archive button next to the progress bar or in a header area. The exact placement depends on the current layout — position it as a subtle action in the top-right of the detail panel.

**Step 2: Run type check**

Run: `cd web && npx tsc -b`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/SessionDetailPreview.tsx
git commit -m "feat(ui): add archive action to SessionDetailPreview"
```

---

### Task 9: Repository tests — Verify archive filtering

**Files:**
- Modify: `internal/repository/content_memory_test.go` (add archive tests)

**Step 1: Add test for archive filtering**

Add to `internal/repository/content_memory_test.go`:

```go
func TestMemoryContentSessionRepo_ArchiveFiltering(t *testing.T) {
	repo := NewMemoryContentSessionRepository()
	ctx := context.Background()
	now := time.Now()

	active := &upal.ContentSession{
		ID: "csess-active", PipelineID: "pipe-1",
		Status: upal.SessionPendingReview, CreatedAt: now,
	}
	archived := &upal.ContentSession{
		ID: "csess-archived", PipelineID: "pipe-1",
		Status: upal.SessionRejected, CreatedAt: now, ArchivedAt: &now,
	}

	repo.Create(ctx, active)
	repo.Create(ctx, archived)

	// List excludes archived
	list, _ := repo.List(ctx)
	if len(list) != 1 {
		t.Errorf("List: expected 1, got %d", len(list))
	}

	// ListByPipeline excludes archived
	byPipeline, _ := repo.ListByPipeline(ctx, "pipe-1")
	if len(byPipeline) != 1 {
		t.Errorf("ListByPipeline: expected 1, got %d", len(byPipeline))
	}

	// ListArchivedByPipeline returns only archived
	archivedList, _ := repo.ListArchivedByPipeline(ctx, "pipe-1")
	if len(archivedList) != 1 {
		t.Errorf("ListArchivedByPipeline: expected 1, got %d", len(archivedList))
	}
	if archivedList[0].ID != "csess-archived" {
		t.Errorf("expected archived session ID, got %q", archivedList[0].ID)
	}
}

func TestMemoryContentSessionRepo_Delete(t *testing.T) {
	repo := NewMemoryContentSessionRepository()
	ctx := context.Background()

	s := &upal.ContentSession{
		ID: "csess-del", PipelineID: "pipe-1",
		Status: upal.SessionCollecting, CreatedAt: time.Now(),
	}
	repo.Create(ctx, s)

	if err := repo.Delete(ctx, "csess-del"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := repo.Get(ctx, "csess-del")
	if err == nil {
		t.Error("expected error getting deleted session")
	}

	// Delete non-existent
	err = repo.Delete(ctx, "csess-nonexistent")
	if err == nil {
		t.Error("expected error deleting non-existent session")
	}
}
```

**Step 2: Run tests**

Run: `go test ./internal/repository/... -v -race`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add internal/repository/content_memory_test.go
git commit -m "test: add archive filtering and delete tests for memory content session repo"
```

---

### Task 10: Final verification

**Step 1: Run full Go test suite**

Run: `go test ./... -v -race`
Expected: ALL PASS

**Step 2: Run frontend type check**

Run: `cd web && npx tsc -b`
Expected: No errors

**Step 3: Run frontend lint**

Run: `cd web && npm run lint`
Expected: No errors

**Step 4: Manual smoke test**

Start the dev servers and verify:
1. Sessions list shows without archived
2. Archive a session → it disappears from All tab
3. Click Archived tab → see the session with original status badge
4. Unarchive → returns to All tab
5. Archive again → Delete permanently → confirmation → gone

**Step 5: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address any issues found during verification"
```
