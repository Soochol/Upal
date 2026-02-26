# Session/Run Restructuring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the Pipeline → ContentSession hierarchy with a flat Session/Run model where Session holds configuration and Run holds execution state.

**Architecture:** Session absorbs Pipeline + ContentSession(template). Run absorbs ContentSession(instance). WorkflowRun replaces WorkflowResult. PipelineRun and PipelineRunner are deleted. Stage orchestration is handled by Run's state machine.

**Tech Stack:** Go 1.24 (backend), React 19 + TypeScript (frontend), Chi router, in-memory + PostgreSQL repositories

**Design doc:** `docs/plans/2026-02-26-session-run-restructuring-design.md`

---

## Task 1: Define New Domain Types

**Files:**
- Create: `internal/upal/session.go`
- Modify: `internal/upal/content.go`
- Delete: `internal/upal/pipeline.go` (after migration)

**Step 1: Create Session type**

Create `internal/upal/session.go` with:

```go
package upal

import "time"

type SessionStatus string

const (
	SessionStatusDraft    SessionStatus = "draft"
	SessionStatusActive   SessionStatus = "active"
	SessionStatusArchived SessionStatus = "archived"
)

type Session struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Sources     []PipelineSource  `json:"sources,omitempty"`
	Schedule    string            `json:"schedule,omitempty"`
	Model       string            `json:"model,omitempty"`
	Workflows   []PipelineWorkflow `json:"workflows,omitempty"`
	Context     *PipelineContext  `json:"context,omitempty"`
	Stages      []Stage           `json:"stages,omitempty"`
	Status      SessionStatus     `json:"status"`
	ThumbnailSVG string           `json:"thumbnail_svg,omitempty"`
	PendingRunCount int           `json:"pending_run_count,omitempty"`
	LastCollectedAt *time.Time    `json:"last_collected_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
```

Note: `PipelineSource`, `PipelineWorkflow`, `PipelineContext`, `Stage`, `StageConfig`, `CollectSource` types remain in their current form — they are data structures used by Session. Just move them from `pipeline.go` to `session.go` and rename where needed later.

**Step 2: Create Run type in `content.go`**

Replace `ContentSession` with `Run`:

```go
type RunStatus string

const (
	RunStatusCollecting    RunStatus = "collecting"
	RunStatusAnalyzing     RunStatus = "analyzing"
	RunStatusPendingReview RunStatus = "pending_review"
	RunStatusApproved      RunStatus = "approved"
	RunStatusRejected      RunStatus = "rejected"
	RunStatusProducing     RunStatus = "producing"
	RunStatusPublished     RunStatus = "published"
	RunStatusError         RunStatus = "error"
)

type Run struct {
	ID            string        `json:"id"`
	SessionID     string        `json:"session_id"`
	Status        RunStatus     `json:"status"`
	TriggerType   string        `json:"trigger_type"`
	SourceCount   int           `json:"source_count,omitempty"`
	ScheduleID    string        `json:"schedule_id,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	ReviewedAt    *time.Time    `json:"reviewed_at,omitempty"`
}

type RunDetail struct {
	Run
	SessionName     string          `json:"session_name,omitempty"`
	Sources         []*SourceFetch  `json:"sources,omitempty"`
	Analysis        *LLMAnalysis    `json:"analysis,omitempty"`
	WorkflowRuns    []WorkflowRun   `json:"workflow_runs,omitempty"`
}
```

**Step 3: Replace WorkflowResult with WorkflowRun**

```go
type WorkflowRun struct {
	WorkflowName string    `json:"workflow_name"`
	RunID        string    `json:"run_id"`
	Status       string    `json:"status"`
	ChannelID    string    `json:"channel_id,omitempty"`
	OutputURL    string    `json:"output_url,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	FailedNodeID string    `json:"failed_node_id,omitempty"`
}
```

**Step 4: Update SourceFetch and LLMAnalysis**

Change `SessionID` field to `RunID` in both `SourceFetch` and `LLMAnalysis` structs.

**Step 5: Verify build compiles (will fail — many references)**

Run: `go build ./internal/upal/...`
Expected: Compiles for upal package. Other packages will fail — that's expected.

**Step 6: Commit**

```
feat: define Session and Run domain types
```

---

## Task 2: Port Interfaces

**Files:**
- Create: `internal/upal/ports/session.go`
- Modify: `internal/upal/ports/content.go`
- Delete: `internal/upal/ports/pipeline.go` (after migration)

**Step 1: Create SessionServicePort**

```go
package ports

type SessionServicePort interface {
	CreateSession(ctx context.Context, s *upal.Session) (*upal.Session, error)
	GetSession(ctx context.Context, id string) (*upal.Session, error)
	ListSessions(ctx context.Context) ([]*upal.Session, error)
	UpdateSession(ctx context.Context, s *upal.Session) error
	DeleteSession(ctx context.Context, id string) error
}
```

**Step 2: Replace ContentSessionPort with RunServicePort**

```go
type RunServicePort interface {
	CreateRun(ctx context.Context, sessionID string, triggerType string) (*upal.Run, error)
	GetRun(ctx context.Context, id string) (*upal.Run, error)
	GetRunDetail(ctx context.Context, id string) (*upal.RunDetail, error)
	ListRuns(ctx context.Context) ([]*upal.RunDetail, error)
	ListRunsBySession(ctx context.Context, sessionID string) ([]*upal.RunDetail, error)
	ListRunsByStatus(ctx context.Context, status upal.RunStatus) ([]*upal.RunDetail, error)
	UpdateRunStatus(ctx context.Context, id string, status upal.RunStatus) error
	ApproveRun(ctx context.Context, id string) error
	RejectRun(ctx context.Context, id string) error
	DeleteRun(ctx context.Context, id string) error

	// Source fetches
	RecordSourceFetch(ctx context.Context, fetch *upal.SourceFetch) error
	UpdateSourceFetch(ctx context.Context, fetch *upal.SourceFetch) error
	ListSourceFetches(ctx context.Context, runID string) ([]*upal.SourceFetch, error)

	// Analysis
	RecordAnalysis(ctx context.Context, analysis *upal.LLMAnalysis) error
	GetAnalysis(ctx context.Context, runID string) (*upal.LLMAnalysis, error)
	UpdateAnalysis(ctx context.Context, analysis *upal.LLMAnalysis) error
	UpdateAnalysisAngles(ctx context.Context, runID string, angles []upal.ContentAngle) error
	UpdateAngleWorkflow(ctx context.Context, runID, angleID, workflowName, workflowRunID string) error

	// Workflow runs
	SetWorkflowRuns(ctx context.Context, runID string, results []upal.WorkflowRun) error
	GetWorkflowRuns(ctx context.Context, runID string) ([]upal.WorkflowRun, error)

	// Published
	RecordPublished(ctx context.Context, p *upal.PublishedContent) error
	ListPublished(ctx context.Context) ([]*upal.PublishedContent, error)
	ListPublishedByRun(ctx context.Context, runID string) ([]*upal.PublishedContent, error)
	ListPublishedByChannel(ctx context.Context, channelID string) ([]*upal.PublishedContent, error)

	// Surge
	CreateSurge(ctx context.Context, event *upal.SurgeEvent) error
	ListSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	ListActiveSurges(ctx context.Context) ([]*upal.SurgeEvent, error)
	DismissSurge(ctx context.Context, id string) error
}
```

**Step 3: Commit**

```
feat: define Session and Run port interfaces
```

---

## Task 3: Repository Interfaces + Memory Implementations

**Files:**
- Create: `internal/repository/session.go` (interfaces)
- Create: `internal/repository/session_memory.go`
- Create: `internal/repository/run.go` (interfaces)
- Create: `internal/repository/run_memory.go`
- Modify: `internal/repository/content.go` (update SourceFetch/Analysis repos to use runID)
- Modify: `internal/repository/content_memory.go`

**Step 1: Define SessionRepository interface**

```go
type SessionRepository interface {
	Create(ctx context.Context, s *upal.Session) error
	Get(ctx context.Context, id string) (*upal.Session, error)
	List(ctx context.Context) ([]*upal.Session, error)
	Update(ctx context.Context, s *upal.Session) error
	Delete(ctx context.Context, id string) error
}
```

**Step 2: Implement MemorySessionRepository**

Standard in-memory implementation with `sync.RWMutex`, following existing patterns in `pipeline_memory.go`.

**Step 3: Define RunRepository interface**

```go
type RunRepository interface {
	Create(ctx context.Context, r *upal.Run) error
	Get(ctx context.Context, id string) (*upal.Run, error)
	List(ctx context.Context) ([]*upal.Run, error)
	ListBySession(ctx context.Context, sessionID string) ([]*upal.Run, error)
	ListByStatus(ctx context.Context, status upal.RunStatus) ([]*upal.Run, error)
	Update(ctx context.Context, r *upal.Run) error
	Delete(ctx context.Context, id string) error
	DeleteBySession(ctx context.Context, sessionID string) error
}
```

**Step 4: Implement MemoryRunRepository**

Standard in-memory implementation.

**Step 5: Update content repository interfaces**

In `content.go`, change `SourceFetchRepository` and `LLMAnalysisRepository` method signatures from `sessionID` to `runID`. Update `PublishedContentRepository.ListBySession` → `ListByRun`. Update `WorkflowResultRepository` → `WorkflowRunRepository`.

**Step 6: Update `content_memory.go`**

Update all in-memory implementations to match new interfaces (runID instead of sessionID).

**Step 7: Write tests for MemorySessionRepository**

Test file: `internal/repository/session_memory_test.go`

```go
func TestMemorySessionRepository_CRUD(t *testing.T) {
	// Create, Get, List, Update, Delete
}
```

**Step 8: Write tests for MemoryRunRepository**

Test file: `internal/repository/run_memory_test.go`

```go
func TestMemoryRunRepository_CRUD(t *testing.T) {
	// Create, Get, List, ListBySession, ListByStatus, Update, Delete, DeleteBySession
}
```

**Step 9: Run tests**

Run: `go test ./internal/repository/... -v -race`
Expected: All new tests pass.

**Step 10: Commit**

```
feat: add Session and Run repository interfaces and memory implementations
```

---

## Task 4: SessionService

**Files:**
- Create: `internal/services/session_service.go`
- Create: `internal/services/session_service_test.go`
- Delete: `internal/services/pipeline_service.go` (after migration)

**Step 1: Write test for SessionService.Create**

```go
func TestSessionService_Create(t *testing.T) {
	repo := repository.NewMemorySessionRepository()
	svc := NewSessionService(repo)

	sess, err := svc.Create(ctx, &upal.Session{Name: "Test"})
	require.NoError(t, err)
	assert.HasPrefix(t, sess.ID, "sess_")
	assert.Equal(t, upal.SessionStatusDraft, sess.Status)
}
```

**Step 2: Run test → FAIL**

**Step 3: Implement SessionService**

```go
type SessionService struct {
	repo repository.SessionRepository
}

func NewSessionService(repo repository.SessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

func (s *SessionService) Create(ctx context.Context, sess *upal.Session) (*upal.Session, error) {
	sess.ID = "sess_" + generateID()
	sess.Status = upal.SessionStatusDraft
	sess.CreatedAt = time.Now()
	sess.UpdatedAt = sess.CreatedAt
	// Generate stage IDs if stages present
	for i := range sess.Stages {
		if sess.Stages[i].ID == "" {
			sess.Stages[i].ID = "stg_" + generateID()
		}
	}
	return sess, s.repo.Create(ctx, sess)
}
// Get, List, Update, Delete — straightforward delegation
```

**Step 4: Run test → PASS**

**Step 5: Write tests for remaining CRUD**

**Step 6: Run all tests → PASS**

**Step 7: Commit**

```
feat: add SessionService with CRUD operations
```

---

## Task 5: RunService

**Files:**
- Create: `internal/services/run_service.go`
- Create: `internal/services/run_service_test.go`
- Delete: `internal/services/content_session_service.go` (after migration)

**Step 1: Write test for RunService.CreateRun**

```go
func TestRunService_CreateRun(t *testing.T) {
	sessionRepo := repository.NewMemorySessionRepository()
	runRepo := repository.NewMemoryRunRepository()
	// ... other repos
	svc := NewRunService(runRepo, sessionRepo, fetchRepo, analysisRepo, publishedRepo, surgeRepo, wfRunRepo)

	// Create a session first
	sess := &upal.Session{ID: "sess_1", Name: "Test", Status: upal.SessionStatusActive}
	sessionRepo.Create(ctx, sess)

	run, err := svc.CreateRun(ctx, "sess_1", "manual")
	require.NoError(t, err)
	assert.HasPrefix(t, run.ID, "run_")
	assert.Equal(t, upal.RunStatusCollecting, run.Status)
}
```

**Step 2: Run test → FAIL**

**Step 3: Implement RunService**

```go
type RunService struct {
	runs       repository.RunRepository
	sessions   repository.SessionRepository
	fetches    repository.SourceFetchRepository
	analyses   repository.LLMAnalysisRepository
	published  repository.PublishedContentRepository
	surges     repository.SurgeEventRepository
	wfRuns     repository.WorkflowRunRepository
}
```

Key methods:
- `CreateRun(ctx, sessionID, triggerType)` — creates Run with `collecting` status
- `GetRunDetail(ctx, runID)` — composes Run + SourceFetch[] + Analysis + WorkflowRun[] + session name
- `ListRuns / ListRunsBySession / ListRunsByStatus` — with detail composition
- `UpdateRunStatus / ApproveRun / RejectRun` — state transitions
- `RecordSourceFetch / GetAnalysis / SetWorkflowRuns` — delegate to sub-repos
- `DeleteRun` — cascade delete fetches, analysis, published, workflow runs

**Step 4: Run test → PASS**

**Step 5: Write tests for GetRunDetail, status transitions, cascade delete**

**Step 6: Run all tests → PASS**

**Step 7: Commit**

```
feat: add RunService with lifecycle management
```

---

## Task 6: Update ContentCollector → Use RunService

**Files:**
- Modify: `internal/services/content_collector.go`

This is the most complex migration — ContentCollector currently uses ContentSessionService and references Pipeline.

**Step 1: Update struct dependencies**

Replace `contentSvc *ContentSessionService` with `runSvc *RunService` and `sessionSvc *SessionService`. Remove `pipelineRepo`.

**Step 2: Update CollectPipeline → CollectSession**

```go
func (c *ContentCollector) CollectSession(ctx context.Context, sessionID string) (*upal.Run, error) {
	sess, err := c.sessionSvc.Get(ctx, sessionID)
	// Create run from session
	run, err := c.runSvc.CreateRun(ctx, sessionID, "manual")
	// Async collect and analyze
	go c.CollectAndAnalyze(ctx, sess, run, false, 0)
	return run, nil
}
```

**Step 3: Update CollectAndAnalyze signature**

Change from `(pipeline, session, isTest, limit)` to `(session, run, isTest, limit)`. Replace all `session.ID` references to use `run.ID` for source fetch / analysis recording. Use `session.Sources` for source configuration (was `session.SessionSources`).

**Step 4: Update ProduceWorkflows**

Change from `sessionID` parameter to `runID`. Load Run instead of ContentSession. Use `run.ID` for workflow result tracking.

**Step 5: Update mapPipelineSources**

Sources now come from `Session.Sources` directly (no change in structure, just the source entity).

**Step 6: Run existing tests (will need updates)**

Run: `go test ./internal/services/... -v -race`
Fix any compilation errors from type changes.

**Step 7: Commit**

```
refactor: update ContentCollector to use Session/Run model
```

---

## Task 7: API Handlers — Session Routes

**Files:**
- Create: `internal/api/sessions.go`
- Modify: `internal/api/server.go` (routes)
- Delete: `internal/api/pipelines.go` (after migration)

**Step 1: Write test for createSession handler**

File: `internal/api/sessions_test.go`

```go
func TestCreateSession(t *testing.T) {
	srv := &Server{sessionSvc: mockSessionSvc}
	body := `{"name":"Test Session","sources":[]}`
	req := httptest.NewRequest("POST", "/api/sessions", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.createSession(rec, req)
	assert.Equal(t, 201, rec.Code)
}
```

**Step 2: Run test → FAIL**

**Step 3: Implement session handlers**

```go
func (s *Server) createSession(w http.ResponseWriter, r *http.Request)
func (s *Server) listSessions(w http.ResponseWriter, r *http.Request)
func (s *Server) getSession(w http.ResponseWriter, r *http.Request)
func (s *Server) updateSession(w http.ResponseWriter, r *http.Request)
func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request)
func (s *Server) activateSession(w http.ResponseWriter, r *http.Request)
func (s *Server) deactivateSession(w http.ResponseWriter, r *http.Request)
```

**Step 4: Run test → PASS**

**Step 5: Commit**

```
feat: add session API handlers
```

---

## Task 8: API Handlers — Run Routes

**Files:**
- Create: `internal/api/runs_session.go` (run management handlers, distinct from existing `runs.go` which handles workflow execution events)
- Modify: `internal/api/server.go` (routes)
- Delete: `internal/api/content.go` (after migration)

**Step 1: Write test for createRun handler**

```go
func TestCreateRun(t *testing.T) {
	// POST /api/sessions/{id}/runs → creates run + triggers collection
}
```

**Step 2: Run test → FAIL**

**Step 3: Implement run handlers**

```go
func (s *Server) createRun(w http.ResponseWriter, r *http.Request)      // POST /api/sessions/{id}/runs
func (s *Server) listSessionRuns(w http.ResponseWriter, r *http.Request) // GET /api/sessions/{id}/runs
func (s *Server) listAllRuns(w http.ResponseWriter, r *http.Request)     // GET /api/runs
func (s *Server) getRunDetail(w http.ResponseWriter, r *http.Request)    // GET /api/runs/{id}
func (s *Server) produceRun(w http.ResponseWriter, r *http.Request)      // POST /api/runs/{id}/produce
func (s *Server) publishRun(w http.ResponseWriter, r *http.Request)      // POST /api/runs/{id}/publish
func (s *Server) rejectRun(w http.ResponseWriter, r *http.Request)       // POST /api/runs/{id}/reject
func (s *Server) listRunSources(w http.ResponseWriter, r *http.Request)  // GET /api/runs/{id}/sources
func (s *Server) getRunAnalysis(w http.ResponseWriter, r *http.Request)  // GET /api/runs/{id}/analysis
func (s *Server) patchRunAnalysis(w http.ResponseWriter, r *http.Request)// PATCH /api/runs/{id}/analysis
```

**Step 4: Run test → PASS**

**Step 5: Commit**

```
feat: add run API handlers
```

---

## Task 9: Wire Routes + Update Server Struct

**Files:**
- Modify: `internal/api/server.go`
- Modify: `cmd/upal/main.go`

**Step 1: Update Server struct**

Replace fields:
```go
// Remove:
pipelineSvc    *services.PipelineService
pipelineRunner *services.PipelineRunner
contentSvc     *services.ContentSessionService
collector      *services.ContentCollector

// Add:
sessionSvc *services.SessionService
runSvc     *services.RunService
collector  *services.ContentCollector  // updated version
```

**Step 2: Update Handler() routes**

Replace `/api/pipelines/` and `/api/content-sessions/` blocks with:

```go
// Sessions
r.Route("/api/sessions", func(r chi.Router) {
    r.Post("/", s.createSession)
    r.Get("/", s.listSessions)
    r.Route("/{id}", func(r chi.Router) {
        r.Get("/", s.getSession)
        r.Put("/", s.updateSession)
        r.Delete("/", s.deleteSession)
        r.Post("/activate", s.activateSession)
        r.Post("/deactivate", s.deactivateSession)
        r.Post("/runs", s.createRun)
        r.Get("/runs", s.listSessionRuns)
        r.Post("/configure", s.configureSession)
        r.Post("/thumbnail", s.generateSessionThumbnail)
    })
})

// Runs
r.Route("/api/runs", func(r chi.Router) {
    // Note: /api/runs already exists for workflow execution events.
    // Add session-run endpoints alongside:
    r.Get("/sessions", s.listAllRuns)  // or nest differently to avoid collision
})
r.Route("/api/session-runs", func(r chi.Router) {
    r.Get("/", s.listAllRuns)
    r.Route("/{id}", func(r chi.Router) {
        r.Get("/", s.getRunDetail)
        r.Post("/produce", s.produceRun)
        r.Post("/publish", s.publishRun)
        r.Post("/reject", s.rejectRun)
        r.Get("/sources", s.listRunSources)
        r.Get("/analysis", s.getRunAnalysis)
        r.Patch("/analysis", s.patchRunAnalysis)
    })
})
```

**Important:** `/api/runs` already handles workflow execution events (SSE streaming). Session-runs need a distinct path. Options:
- `/api/session-runs/` — clear separation
- `/api/runs/` with sub-routes that don't collide (e.g. `/api/runs/{id}` already exists for workflow runs)

Decision: Use `/api/session-runs/` to avoid collision with existing workflow run endpoints. Frontend can alias this.

**Step 3: Update main.go DI wiring**

```go
// Replace pipeline + content-session setup with:
sessionRepo := repository.NewMemorySessionRepository()
runRepo := repository.NewMemoryRunRepository()
// ... source fetch, analysis, published, surge, wfRun repos (reuse existing)

sessionSvc := services.NewSessionService(sessionRepo)
runSvc := services.NewRunService(runRepo, sessionRepo, fetchRepo, analysisRepo, publishedRepo, surgeRepo, wfRunRepo)
collector := services.NewContentCollector(sessionSvc, runSvc, collectExec, workflowSvc, ...)

srv.SetSessionService(sessionSvc)
srv.SetRunService(runSvc)
srv.SetContentCollector(collector)
```

**Step 4: Verify build**

Run: `go build ./cmd/upal/...`
Expected: Compiles successfully.

**Step 5: Run all backend tests**

Run: `go test ./... -v -race`
Expected: All pass.

**Step 6: Commit**

```
feat: wire Session/Run services into server and DI
```

---

## Task 10: Delete Old Pipeline/ContentSession Backend Code

**Files:**
- Delete: `internal/upal/pipeline.go` (move reusable types to session.go first)
- Delete: `internal/upal/ports/pipeline.go`
- Delete: `internal/repository/pipeline.go`
- Delete: `internal/repository/pipeline_memory.go`
- Delete: `internal/repository/pipeline_persistent.go`
- Delete: `internal/services/pipeline_service.go`
- Delete: `internal/services/pipeline_runner.go`
- Delete: `internal/api/pipelines.go`
- Delete: `internal/api/content.go` (session-run handlers now in sessions.go + runs_session.go)
- Delete: `internal/api/configure_pipeline.go`
- Modify: Other files that import deleted packages — update or remove references

**Step 1: Move shared types from pipeline.go to session.go**

Types to keep: `Stage`, `StageConfig`, `CollectSource`, `PipelineSource`, `PipelineWorkflow`, `PipelineContext`. These are data structures used by Session — move them to `session.go`. Consider renaming `PipelineSource` → `Source`, `PipelineWorkflow` → `SessionWorkflow`, `PipelineContext` → `SessionContext` for consistency.

**Step 2: Delete old files**

**Step 3: Fix all compilation errors**

Run: `go build ./...`
Fix imports and references throughout codebase.

**Step 4: Run tests**

Run: `go test ./... -v -race`

**Step 5: Commit**

```
refactor: remove Pipeline and ContentSession code
```

---

## Task 11: Update Scheduler and Other Backend References

**Files:**
- Modify: `internal/services/scheduler.go` — replace pipeline references with session
- Modify: `internal/generate/pipeline.go` — rename to session generation or update references
- Modify: `internal/generate/backfill.go` — update stage description backfill to use Session
- Modify: `internal/api/configure_session.go` — update to use new types
- Modify: `internal/api/generate.go` — update pipeline generation → session generation

**Step 1: Update scheduler**

The scheduler currently calls `CollectPipeline(pipelineID)`. Change to `CollectSession(sessionID)`. Update any schedule-to-pipeline mappings to schedule-to-session.

**Step 2: Update generate module**

Rename pipeline generation functions to session generation. Update skill prompt references if needed.

**Step 3: Update configure endpoints**

`configure_pipeline.go` → integrated into `configure_session.go`.

**Step 4: Run tests**

Run: `go test ./... -v -race`

**Step 5: Commit**

```
refactor: update scheduler, generator, and configure to use Session model
```

---

## Task 12: Update PostgreSQL Repository (if applicable)

**Files:**
- Create: `internal/repository/session_persistent.go`
- Create: `internal/repository/run_persistent.go`
- Modify: `internal/repository/content_persistent.go`
- Delete: `internal/repository/pipeline_persistent.go`
- Modify: `internal/db/` — update SQL schema

**Step 1: Create sessions table schema**

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    sources JSONB,
    schedule TEXT,
    model TEXT,
    workflows JSONB,
    context JSONB,
    stages JSONB,
    status TEXT NOT NULL DEFAULT 'draft',
    thumbnail_svg TEXT,
    last_collected_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Step 2: Create runs table schema**

```sql
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    status TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    source_count INTEGER DEFAULT 0,
    schedule_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);
```

**Step 3: Update source_fetches and llm_analyses tables**

Rename `session_id` column to `run_id`.

**Step 4: Implement persistent repositories**

Follow existing patterns in `pipeline_persistent.go` and `content_persistent.go`.

**Step 5: Update main.go persistent wiring**

**Step 6: Commit**

```
feat: add PostgreSQL implementations for Session and Run repositories
```

---

## Task 13: Frontend — New Types

**Files:**
- Create: `web/src/entities/session/types.ts`
- Create: `web/src/entities/session/index.ts`
- Create: `web/src/entities/run/types.ts`
- Create: `web/src/entities/run/constants.ts`
- Create: `web/src/entities/run/index.ts`

**Step 1: Define Session type**

```typescript
export type SessionStatus = 'draft' | 'active' | 'archived'

export type Session = {
  id: string
  name: string
  description?: string
  sources?: PipelineSource[]
  schedule?: string
  model?: string
  workflows?: PipelineWorkflow[]
  context?: PipelineContext
  stages?: Stage[]
  status: SessionStatus
  thumbnail_svg?: string
  pending_run_count?: number
  last_collected_at?: string
  created_at: string
  updated_at: string
}
```

Note: Reuse `PipelineSource`, `PipelineWorkflow`, `PipelineContext`, `Stage` types — move them from `entities/pipeline/types.ts` to `entities/session/types.ts`. Rename if desired (`Source`, `SessionWorkflow`, `SessionContext`).

**Step 2: Define Run type**

```typescript
export type RunStatus =
  | 'collecting' | 'analyzing' | 'pending_review'
  | 'approved' | 'rejected' | 'producing' | 'published' | 'error'

export type WorkflowRun = {
  workflow_name: string
  run_id: string
  status: string
  channel_id?: string
  output_url?: string
  completed_at?: string
  error_message?: string
  failed_node_id?: string
}

export type Run = {
  id: string
  session_id: string
  session_name?: string
  status: RunStatus
  trigger_type: 'schedule' | 'manual' | 'surge'
  source_count?: number
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_runs?: WorkflowRun[]
  created_at: string
  reviewed_at?: string
}
```

**Step 3: Move status constants from `content-session/constants.ts` to `run/constants.ts`**

Adapt `SESSION_STATUS_DOT`, `SESSION_FILTER_TABS`, `matchesSessionFilter` to use `RunStatus`.

**Step 4: Commit**

```
feat: add frontend Session and Run types
```

---

## Task 14: Frontend — API Client

**Files:**
- Create: `web/src/entities/session/api.ts`
- Create: `web/src/entities/run/api.ts`
- Delete: `web/src/entities/pipeline/api/index.ts` (after migration)
- Delete: `web/src/entities/content-session/api.ts` (after migration)

**Step 1: Implement Session API**

```typescript
export async function fetchSessions(): Promise<Session[]> {
  const res = await fetch('/api/sessions')
  return res.json()
}
export async function fetchSession(id: string): Promise<Session> { ... }
export async function createSession(data: Partial<Session>): Promise<Session> { ... }
export async function updateSession(id: string, data: Partial<Session>): Promise<Session> { ... }
export async function deleteSession(id: string): Promise<void> { ... }
export async function activateSession(id: string): Promise<void> { ... }
export async function deactivateSession(id: string): Promise<void> { ... }
```

**Step 2: Implement Run API**

```typescript
export async function createRun(sessionId: string): Promise<Run> {
  const res = await fetch(`/api/sessions/${sessionId}/runs`, { method: 'POST' })
  return res.json()
}
export async function fetchRuns(params?: { sessionId?: string; status?: string }): Promise<Run[]> { ... }
export async function fetchRun(id: string): Promise<Run> {
  const res = await fetch(`/api/session-runs/${id}`)
  return res.json()
}
export async function produceRun(id: string, workflows: any[]): Promise<void> { ... }
export async function publishRun(id: string, runIds: string[]): Promise<void> { ... }
export async function rejectRun(id: string): Promise<void> { ... }
// ... sources, analysis endpoints
```

**Step 3: Commit**

```
feat: add frontend Session and Run API clients
```

---

## Task 15: Frontend — Sessions Page (Replacing Pipelines Page)

**Files:**
- Create: `web/src/pages/sessions/` directory
- Move + refactor components from `web/src/pages/pipelines/`
- Modify: `web/src/app/router.tsx`

**Step 1: Create Sessions page structure**

```
pages/sessions/
  ├── index.tsx              ← Main page (2-panel: sidebar + detail)
  ├── SessionSidebar.tsx     ← Session list (replaces PipelineSidebar + SessionListPanel)
  ├── SessionSetupView.tsx   ← Session configuration (from pipelines/SessionSetupView.tsx)
  ├── RunHistoryPanel.tsx    ← Run list for selected session (new)
  └── session/
       └── stages/           ← Reuse CollectStage, AnalyzeStage, ProduceStage, PublishStage
```

**Step 2: Simplify to 2-panel layout**

Current Pipelines page: Pipeline sidebar → Session list → Session detail (3 panels)
New Sessions page: Session sidebar → Session detail with tabs (2 panels)

Tab structure for selected session:
- **Setup** tab: sources, schedule, model, workflows, context configuration
- **Runs** tab: list of runs with status, click to expand detail

**Step 3: Update router**

```typescript
// Change:
{ path: '/pipelines', element: <PipelinesPage /> }
// To:
{ path: '/sessions', element: <SessionsPage /> }
```

**Step 4: Verify frontend builds**

Run: `cd web && npm run build`

**Step 5: Commit**

```
feat: add Sessions page replacing Pipelines
```

---

## Task 16: Frontend — Update Inbox Page

**Files:**
- Modify: `web/src/pages/inbox/index.tsx`
- Modify: `web/src/pages/inbox/InboxSidebar.tsx`
- Modify: `web/src/pages/inbox/InboxPreview.tsx`

**Step 1: Update data fetching**

Change from `fetchContentSessions()` to `fetchRuns()`. Update types from `ContentSession` to `Run`.

**Step 2: Update status filtering**

Use new `run/constants.ts` for status dot colors and filter logic.

**Step 3: Update detail preview**

`InboxPreview` now renders `Run` detail instead of `ContentSession` detail.

**Step 4: Verify build**

Run: `cd web && npm run build`

**Step 5: Commit**

```
refactor: update Inbox page to use Run model
```

---

## Task 17: Frontend — Clean Up Old Entities

**Files:**
- Delete: `web/src/entities/pipeline/` directory
- Delete: `web/src/entities/content-session/` directory
- Modify: Any remaining imports across the app
- Delete: `web/src/widgets/pipeline-editor/` (or refactor to session editor)
- Delete/rename: `web/src/features/configure-pipeline-sources/` → `configure-session-sources/`
- Delete/rename: `web/src/features/generate-pipeline/` → `generate-session/`

**Step 1: Delete old entity directories**

**Step 2: Fix all import references**

Run: `cd web && npx tsc -b` to find all broken imports.

**Step 3: Rename feature directories**

**Step 4: Verify build**

Run: `cd web && npm run build`

**Step 5: Commit**

```
refactor: remove old pipeline and content-session frontend code
```

---

## Task 18: Update Navigation and Sidebar

**Files:**
- Modify: `web/src/widgets/` — sidebar/navigation components
- Modify: `web/src/app/router.tsx` — ensure all route references updated

**Step 1: Update navigation labels**

"Pipelines" → "Sessions" in sidebar navigation.

**Step 2: Update URL references**

Any hardcoded `/pipelines` links → `/sessions`.

**Step 3: Verify full app**

Run: `cd web && npm run build && npm run lint`

**Step 4: Commit**

```
refactor: update navigation from Pipelines to Sessions
```

---

## Task 19: End-to-End Verification

**Step 1: Run full backend tests**

Run: `make test`
Expected: All pass.

**Step 2: Run frontend type check + build**

Run: `make test-frontend`
Expected: No type errors.

**Step 3: Run full build**

Run: `make build`
Expected: Produces `bin/upal` successfully.

**Step 4: Manual smoke test**

Start dev environment and verify:
- Sessions page loads, can create/edit session
- Can trigger a run from session
- Inbox shows runs with correct status filtering
- Run detail shows sources, analysis, workflow runs
- Workflow execution events still stream correctly

**Step 5: Final commit**

```
feat: complete Session/Run restructuring
```

---

## Summary

| Task | Description | Estimated Steps |
|------|-------------|-----------------|
| 1 | Domain types (Session, Run, WorkflowRun) | 6 |
| 2 | Port interfaces | 3 |
| 3 | Repository interfaces + memory impl + tests | 10 |
| 4 | SessionService + tests | 7 |
| 5 | RunService + tests | 7 |
| 6 | Update ContentCollector | 7 |
| 7 | Session API handlers + tests | 5 |
| 8 | Run API handlers + tests | 5 |
| 9 | Wire routes + DI | 6 |
| 10 | Delete old pipeline/content-session backend | 5 |
| 11 | Update scheduler, generator, configure | 5 |
| 12 | PostgreSQL repositories | 6 |
| 13 | Frontend types | 4 |
| 14 | Frontend API client | 3 |
| 15 | Sessions page | 5 |
| 16 | Update Inbox page | 5 |
| 17 | Clean up old frontend code | 5 |
| 18 | Update navigation | 4 |
| 19 | E2E verification | 5 |

**Total: 19 tasks, ~107 steps**

**Dependencies:** Tasks 1-2 → 3 → 4,5 (parallel) → 6 → 7,8 (parallel) → 9 → 10,11 (parallel) → 12 → 13 → 14 → 15,16 (parallel) → 17 → 18 → 19
