# Pipeline Session Setup Redesign — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move pipeline settings (sources, schedule, brief, model, workflows) to session level, simplify inbox to approve/reject only, and replace the session detail stage view with a session setup view.

**Architecture:** Backend-first approach. Add settings fields to ContentSession domain type, update ContentCollector to read from session instead of pipeline, then rebuild the frontend pipeline page as a 2-panel layout (pipeline list + session setup), and simplify inbox AnalyzeStage by removing angle/workflow selection.

**Tech Stack:** Go 1.24 (Chi router), React 19, TypeScript, TanStack Query, Tailwind CSS v4, Zustand

---

## Phase 1: Backend Domain & API

### Task 1: Add settings fields to ContentSession domain type

**Files:**
- Modify: `internal/upal/content.go:22-31`

**Step 1: Add settings fields to ContentSession struct**

Add new fields after `SourceCount` (line 27):

```go
type ContentSession struct {
	ID          string               `json:"id"`
	PipelineID  string               `json:"pipeline_id"`
	Status      ContentSessionStatus `json:"status"`
	TriggerType string               `json:"trigger_type"`
	SourceCount int                  `json:"source_count"`
	// Session-level settings (moved from Pipeline)
	Sources   []PipelineSource   `json:"sources,omitempty"`
	Schedule  string             `json:"schedule,omitempty"`
	Model     string             `json:"model,omitempty"`
	Workflows []PipelineWorkflow `json:"workflows,omitempty"`
	Context   *PipelineContext   `json:"context,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	ReviewedAt  *time.Time         `json:"reviewed_at,omitempty"`
	ArchivedAt  *time.Time         `json:"archived_at,omitempty"`
}
```

**Step 2: Add same fields to ContentSessionDetail**

Update `ContentSessionDetail` (line 142-156) to include the settings:

```go
type ContentSessionDetail struct {
	ID              string               `json:"id"`
	PipelineID      string               `json:"pipeline_id"`
	PipelineName    string               `json:"pipeline_name,omitempty"`
	SessionNumber   int                  `json:"session_number,omitempty"`
	Status          ContentSessionStatus `json:"status"`
	TriggerType     string               `json:"trigger_type"`
	SourceCount     int                  `json:"source_count"`
	// Session-level settings
	Sources         []PipelineSource     `json:"session_sources,omitempty"`
	Schedule        string               `json:"schedule,omitempty"`
	Model           string               `json:"model,omitempty"`
	Workflows       []PipelineWorkflow   `json:"session_workflows,omitempty"`
	Context         *PipelineContext     `json:"context,omitempty"`
	// Related data
	FetchedSources  []*SourceFetch       `json:"sources,omitempty"`
	Analysis        *LLMAnalysis         `json:"analysis,omitempty"`
	WorkflowResults []WorkflowResult     `json:"workflow_results,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	ReviewedAt      *time.Time           `json:"reviewed_at,omitempty"`
	ArchivedAt      *time.Time           `json:"archived_at,omitempty"`
}
```

Note: The existing `Sources []*SourceFetch` field (fetched data) conflicts with the new `Sources []PipelineSource` (settings). Rename to avoid collision:
- Settings sources → `json:"session_sources"` in Detail
- Fetched sources → `json:"sources"` in Detail (keep for backward compat)

**Step 3: Run tests**

Run: `go test ./internal/upal/... -v -race`
Expected: PASS (no logic changes, only struct additions)

**Step 4: Commit**

```bash
git add internal/upal/content.go
git commit -m "feat: add session-level settings fields to ContentSession"
```

---

### Task 2: Add session settings API endpoints

**Files:**
- Modify: `internal/api/content.go`
- Modify: `internal/api/server.go` (add route)
- Modify: `internal/services/content_session_service.go`

**Step 1: Add UpdateSessionSettings to ContentSessionService**

In `content_session_service.go`, add after `UpdateSessionStatus` (~line 181):

```go
// UpdateSessionSettings updates the settings fields of a session.
func (s *ContentSessionService) UpdateSessionSettings(ctx context.Context, id string, update *upal.ContentSession) error {
	sess, err := s.sessions.Get(ctx, id)
	if err != nil {
		return err
	}
	if update.Sources != nil {
		sess.Sources = update.Sources
	}
	if update.Schedule != "" {
		sess.Schedule = update.Schedule
	}
	if update.Model != "" {
		sess.Model = update.Model
	}
	if update.Workflows != nil {
		sess.Workflows = update.Workflows
	}
	if update.Context != nil {
		sess.Context = update.Context
	}
	return s.sessions.Update(ctx, sess)
}
```

**Step 2: Add patchSessionSettings API handler**

In `content.go`, add a new handler:

```go
// PATCH /api/content-sessions/{id}/settings
func (s *Server) patchSessionSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Sources   []upal.PipelineSource   `json:"sources,omitempty"`
		Schedule  string                  `json:"schedule,omitempty"`
		Model     string                  `json:"model,omitempty"`
		Workflows []upal.PipelineWorkflow `json:"workflows,omitempty"`
		Context   *upal.PipelineContext   `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	update := &upal.ContentSession{
		Sources:   body.Sources,
		Schedule:  body.Schedule,
		Model:     body.Model,
		Workflows: body.Workflows,
		Context:   body.Context,
	}
	if err := s.contentSvc.UpdateSessionSettings(r.Context(), id, update); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	sess, _ := s.contentSvc.GetSession(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}
```

**Step 3: Register the route in server.go**

Find the content-sessions route group and add:
```go
r.Patch("/{id}/settings", s.patchSessionSettings)
```

**Step 4: Update GetSessionDetail to include settings fields**

In `content_session_service.go` `GetSessionDetail` (~line 480), copy settings from session to detail:

```go
// Inside GetSessionDetail, after setting basic fields:
detail.SessionSources = sess.Sources  // use the renamed field
detail.Schedule = sess.Schedule
detail.Model = sess.Model
detail.SessionWorkflows = sess.Workflows  // use the renamed field
detail.Context = sess.Context
```

**Step 5: Run tests**

Run: `go test ./internal/services/... ./internal/api/... -v -race`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/api/content.go internal/api/server.go internal/services/content_session_service.go
git commit -m "feat: add session settings CRUD endpoint"
```

---

### Task 3: Update ContentCollector to read from session

**Files:**
- Modify: `internal/services/content_collector.go`

The collector currently reads `pipeline.Sources`, `pipeline.Model`, `pipeline.Context`, `pipeline.Workflows`. Change it to read from session.

**Step 1: Update CollectAndAnalyze signature**

Change the function to also accept session settings. Since session already has the settings, update `CollectAndAnalyze` to read from session rather than pipeline:

At line 91, change:
```go
func (c *ContentCollector) CollectAndAnalyze(ctx context.Context, pipeline *upal.Pipeline, session *upal.ContentSession, isTest bool, limit int) {
	// Map session sources (was pipeline.Sources)
	sources := mapPipelineSources(session.Sources, isTest, limit)
```

At line 106, change:
```go
pipelineSrc := session.Sources[mapped.pipelineIndex]
```

**Step 2: Update runAnalysis to read model/context from session**

At line 282 `runAnalysis`, change to accept session:
```go
func (c *ContentCollector) runAnalysis(ctx context.Context, session *upal.ContentSession, sess *upal.ContentSession) {
```

Actually simpler: pass session instead of pipeline. Key changes:
- Line 296: `c.resolver.Resolve(pipeline.Model)` → `c.resolver.Resolve(session.Model)`
- Line 302: `buildAnalysisPrompt(..., pipeline, ...)` → pass session data instead
- Update `buildAnalysisPrompt` to accept session fields

**Step 3: Update buildAnalysisPrompt**

Change signature from `pipeline *upal.Pipeline` to session fields:
```go
func buildAnalysisPrompt(
	basePrompt string,
	context *upal.PipelineContext,
	workflows []upal.PipelineWorkflow,
	fetches []*upal.SourceFetch,
	allWorkflows []*upal.WorkflowDefinition,
) (string, string)
```

Update calls:
- `pipeline.Context.*` → `context.*`
- `pipeline.Workflows` → `workflows`

**Step 4: Update CollectPipeline (scheduler entry point)**

At line 69, `CollectPipeline` creates a session for scheduled runs. It should copy settings from an existing session (the "template" session) or pass them. For now, this function should look up the session's saved settings:

```go
func (c *ContentCollector) CollectPipeline(ctx context.Context, sessionID string) error {
	session, err := c.contentSvc.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session %s: %w", sessionID, err)
	}
	go c.CollectAndAnalyze(context.Background(), session, false, 0)
	return nil
}
```

Wait — the session concept changes. Previously one session = one run. Now one session = recurring config. Each scheduled run creates "what"? We need to think about this.

**Design decision:** A session is now a configured recurring entity. Each "Start" or scheduled trigger creates a **new execution** within the session. The simplest approach: keep creating new ContentSessions as "child runs" but copy settings from the parent session.

Actually, re-reading the design: "세션에만 설정 존재" + "세션 레벨 스케줄" means the session IS the recurring entity. Each scheduled run of the session should:
1. Read the session's sources
2. Collect + analyze
3. Send results to inbox

The session's status cycles: `draft` (being configured) → `active` (running/scheduled) → and each collection produces an "execution" that goes to inbox.

**Revised approach:** Add a new status `draft` for sessions being configured. When "Start" is pressed, status → `active`. Scheduler triggers collection using the session's settings. Each collection creates a "ContentSessionRun" or reuses the session's status cycling.

**Simpler approach for Phase 1:** Keep the current session model where each session = one execution. The "template" is just the session settings. When schedule fires, create a NEW session copying settings from the template session. The template session has a special status like `configured` or `active`.

Let's add a new status:

```go
SessionConfigured ContentSessionStatus = "configured" // session setup complete, not yet started
SessionActive     ContentSessionStatus = "active"      // recurring, scheduled for auto-collection
```

**Step 5: Update collectPipeline API handler**

In `content.go` line 469, the `collectPipeline` handler creates a session and starts collection. Change to accept `session_id` parameter — start collection for an existing configured session:

Add new endpoint: `POST /api/content-sessions/{id}/collect`

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

	// Create a child execution session copying settings
	execSess := &upal.ContentSession{
		PipelineID:  session.PipelineID,
		TriggerType: "manual",
		Sources:     session.Sources,
		Model:       session.Model,
		Workflows:   session.Workflows,
		Context:     session.Context,
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

**Step 6: Run tests**

Run: `go test ./internal/services/... -v -race`

**Step 7: Commit**

```bash
git add internal/services/content_collector.go internal/api/content.go internal/api/server.go internal/upal/content.go
git commit -m "feat: collector reads settings from session instead of pipeline"
```

---

### Task 4: Auto-produce on approve (backend)

**Files:**
- Modify: `internal/api/content.go:108-139` (patchContentSession)
- Modify: `internal/services/content_session_service.go`

Currently approve just changes status. Now it should also auto-trigger produce using the session's (or parent session's) pre-configured workflows.

**Step 1: Update approve flow**

In `patchContentSession`, after approve, look up the session's workflows and auto-trigger production:

```go
case "approve":
	err = s.contentSvc.ApproveSession(ctx, id)
	if err == nil && s.collector != nil {
		// Auto-produce using session's pre-configured workflows
		sess, _ := s.contentSvc.GetSession(ctx, id)
		if sess != nil && len(sess.Workflows) > 0 {
			var requests []services.WorkflowRequest
			for _, pw := range sess.Workflows {
				requests = append(requests, services.WorkflowRequest{
					Name:      pw.WorkflowName,
					ChannelID: pw.ChannelID,
				})
			}
			go s.collector.ProduceWorkflows(context.Background(), id, requests)
		}
	}
```

**Step 2: Run tests**

Run: `go test ./internal/api/... -v -race`

**Step 3: Commit**

```bash
git add internal/api/content.go
git commit -m "feat: auto-produce using session workflows on approve"
```

---

## Phase 2: Frontend — Session Setup View

### Task 5: Add session settings fields to frontend types

**Files:**
- Modify: `web/src/entities/content-session/types.ts`
- Modify: `web/src/entities/content-session/api.ts`

**Step 1: Update ContentSession type**

Add settings fields to `ContentSession` in `types.ts`:

```typescript
export type ContentSession = {
  id: string
  pipeline_id: string
  pipeline_name?: string
  session_number?: number
  trigger_type: 'schedule' | 'manual' | 'surge'
  status: ContentSessionStatus
  // Session-level settings
  session_sources?: PipelineSource[]
  schedule?: string
  model?: string
  session_workflows?: PipelineWorkflow[]
  context?: PipelineContext
  // Related data
  sources?: SourceFetch[]
  analysis?: LLMAnalysis
  workflow_results?: WorkflowResult[]
  created_at: string
  updated_at?: string
  archived_at?: string
}
```

Add import for `PipelineSource`, `PipelineWorkflow`, `PipelineContext` from `@/shared/types`.

**Step 2: Add updateSessionSettings API function**

In `api.ts`, add:

```typescript
export async function updateSessionSettings(
  id: string,
  settings: {
    sources?: PipelineSource[]
    schedule?: string
    model?: string
    workflows?: PipelineWorkflow[]
    context?: PipelineContext
  },
): Promise<ContentSession> {
  return apiFetch<ContentSession>(`${BASE}/${encodeURIComponent(id)}/settings`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(settings),
  })
}
```

Add `collectSession` API function:

```typescript
export async function collectSession(
  id: string,
  config?: { isTest?: boolean; limit?: number },
): Promise<{ session_id: string }> {
  return apiFetch(`${BASE}/${encodeURIComponent(id)}/collect`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config ?? {}),
  })
}
```

**Step 3: Run type check**

Run: `cd web && npx tsc -b`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/entities/content-session/types.ts web/src/entities/content-session/api.ts
git commit -m "feat: add session settings types and API functions"
```

---

### Task 6: Create SessionSetupView component

**Files:**
- Create: `web/src/pages/pipelines/session/SessionSetupView.tsx`

This component replaces the session detail (stages) view. It renders session settings as scrollable sections, reusing existing components (`AddSourceModal`, `EditorialBriefForm`, `WorkflowPicker`, `ModelSelector`).

**Step 1: Create SessionSetupView**

Build a section-based layout component:
- Header: Session name/number + status badge
- Section 1: Sources & Schedule (reuse `AddSourceModal` for adding, schedule picker from `PipelineSettingsPanel`)
- Section 2: Editorial Brief (reuse `EditorialBriefForm`)
- Section 3: Analysis Model (reuse `ModelSelector`)
- Section 4: Workflows (reuse `WorkflowPicker`)
- Bottom sticky: Start Session button

Props:
```typescript
interface SessionSetupViewProps {
  session: ContentSession
  onSettingsChange: (settings: Partial<SessionSettings>) => void
  onStartSession: () => void
  autoSaveStatus: 'idle' | 'saving' | 'saved'
}
```

Follow the same auto-save pattern as `PipelineSettingsPanel` (debounced, local state → API call).

**Step 2: Wire up auto-save**

Use the same debounce pattern from `Pipelines.tsx` (800ms debounce, dirty detection, save on change).

**Step 3: Run type check**

Run: `cd web && npx tsc -b`

**Step 4: Commit**

```bash
git add web/src/pages/pipelines/session/SessionSetupView.tsx
git commit -m "feat: create SessionSetupView component with section-based layout"
```

---

### Task 7: Refactor Pipelines.tsx layout

**Files:**
- Modify: `web/src/pages/Pipelines.tsx`

**Step 1: Remove session list panel and right sidebar**

Change from 3-panel layout to 2-panel:
- Left: `PipelineSidebar` (unchanged)
- Main: `SessionSetupView` (when session selected) or session list

Actually, per the design: pipeline list | session list | session setup. Keep 3 panels but replace session detail + settings sidebar with session setup view.

Remove:
- Right sidebar (`PipelineSettingsPanel` and its resizable panel)
- `SessionDetailPreview` in main area
- All the local settings state (`localSources`, `localSchedule`, etc.) that was for pipeline-level auto-save

Replace with:
- `SessionSetupView` when a session is selected
- The setup view handles its own state and auto-save

**Step 2: Update NewSessionModal**

The "+ New Session" now creates a session with status `configured` (no immediate collection). Remove test mode options from the modal.

**Step 3: Remove Start Session button pipeline-level collect**

The Start Session button should call `collectSession(sessionId)` instead of `collectPipeline(pipelineId)`.

**Step 4: Run lint and type check**

Run: `cd web && npm run lint && npx tsc -b`

**Step 5: Commit**

```bash
git add web/src/pages/Pipelines.tsx
git commit -m "feat: refactor pipeline page to 2-panel layout with session setup view"
```

---

### Task 8: Simplify AnalyzeStage for inbox

**Files:**
- Modify: `web/src/pages/pipelines/session/stages/AnalyzeStage.tsx`

**Step 1: Remove angle/workflow selection UI**

Remove:
- Lines ~417-513: Entire "Recommended Workflows" / angle grid section
- `selectedAngles`, `overriddenWorkflows`, `generatingAngles` state variables
- `handleWorkflowChange`, `handleGenerateWorkflow` functions
- `WorkflowPicker` modal at bottom
- The `handleApprove` function that collects selected workflow names

Simplify `onApprove` prop: no longer needs `selectedWorkflows` parameter. Change to `onApprove: () => void`.

Keep:
- Score bar
- Two-column layout (sources left, AI summary/insights right)
- Editable summary/insights (if pending_review)
- Approve/Reject buttons (simplified — just approve, no workflow names)

**Step 2: Update InboxPreview handleApprove**

In `InboxPreview.tsx`, simplify `handleApprove` to not pass workflow names:

```typescript
const handleApprove = useCallback(async () => {
    if (!sessionId) return
    setIsApproving(true)
    try {
        await approveSession(sessionId, [])  // empty — backend auto-produces
        await queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
        await queryClient.invalidateQueries({ queryKey: ['inbox-sessions'] })
        navigate('/publish-inbox')
    } finally {
        setIsApproving(false)
    }
}, [sessionId, approveSession, queryClient, navigate])
```

**Step 3: Update store approveSession**

In `store.ts`, remove the produce chain since backend now handles it:

```typescript
approveSession: async (id) => {
    const updated = await approveSession(id, [])
    set((state) => {
        const sessions = state.sessions.map((s) => (s.id === id ? updated : s))
        return {
            sessions,
            pendingCount: sessions.filter((s) => s.status === 'pending_review').length,
            publishReadyCount: sessions.filter((s) => s.status === 'approved' || s.status === 'producing').length,
        }
    })
    // No manual produce call — backend auto-triggers on approve
},
```

**Step 4: Run type check and lint**

Run: `cd web && npx tsc -b && npm run lint`

**Step 5: Commit**

```bash
git add web/src/pages/pipelines/session/stages/AnalyzeStage.tsx web/src/pages/inbox/InboxPreview.tsx web/src/entities/content-session/store.ts
git commit -m "feat: simplify inbox to approve/reject only, remove angle/workflow selection"
```

---

## Phase 3: Scheduler & Cleanup

### Task 9: Session-level scheduling

**Files:**
- Modify: `internal/services/scheduler/dispatch.go`
- Modify: `internal/services/scheduler/sync.go`
- Modify: `internal/services/scheduler/scheduler.go`

Currently the scheduler dispatches based on pipeline.Sources existence. Change to dispatch based on session schedule:

**Step 1: Add session-based scheduling**

The scheduler's `ContentCollector` interface currently takes `pipelineID`. Change to take `sessionID`:

```go
type ContentCollector interface {
	CollectSession(ctx context.Context, sessionID string) error
}
```

Update `dispatch.go` to look for sessions with active schedules instead of pipeline sources:

```go
// In executeScheduledRun, change content pipeline branch:
if schedule.SessionID != "" && s.contentCollector != nil {
	if err := s.contentCollector.CollectSession(ctx, schedule.SessionID); err != nil {
		slog.Error("scheduler: session collection failed", ...)
	}
}
```

**Step 2: Add session schedule sync**

Add `SyncSessionSchedule` to sync a session's cron schedule:

```go
func (s *SchedulerService) SyncSessionSchedule(ctx context.Context, session *upal.ContentSession) error {
	// Remove existing schedule for this session
	// Create new schedule if session.Schedule is non-empty
}
```

**Step 3: Update Schedule domain type**

In `internal/upal/schedule.go`, add `SessionID` field:
```go
type Schedule struct {
	// ... existing fields
	SessionID string `json:"session_id,omitempty"`
}
```

**Step 4: Run tests**

Run: `go test ./internal/services/scheduler/... -v -race`

**Step 5: Commit**

```bash
git add internal/services/scheduler/ internal/upal/
git commit -m "feat: session-level scheduling support"
```

---

### Task 10: Simplify PipelineNew wizard

**Files:**
- Modify: `web/src/pages/pipelines/PipelineNew.tsx`

**Step 1: Reduce to 1-step creation**

Pipeline is now just a group. Remove steps 2-4 (editorial brief, sources/schedule, workflows). Keep only step 1 (name + description).

Remove:
- Steps 2, 3, 4 and their state
- Right sidebar pipeline summary
- Step indicator (only 1 step)

Simplify to a simple form: name + description + "Create Pipeline" button.

**Step 2: Run type check**

Run: `cd web && npx tsc -b`

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/PipelineNew.tsx
git commit -m "feat: simplify pipeline creation to name and description only"
```

---

### Task 11: Floating AI Assistant widget

**Files:**
- Create: `web/src/widgets/floating-chat/FloatingChat.tsx`
- Modify: `web/src/pages/Pipelines.tsx`

**Step 1: Extract AI assistant from PipelineSettingsPanel**

Look at how `PipelineChatEditor` is used in the settings panel and create a floating version:
- Fixed position bottom-right
- Collapsed: small chat bubble icon
- Expanded: chat panel overlay (similar to workflow canvas chat)

**Step 2: Add to Pipelines page**

Render `FloatingChat` at the page level, passing the current pipeline/session context.

**Step 3: Run type check**

Run: `cd web && npx tsc -b`

**Step 4: Commit**

```bash
git add web/src/widgets/floating-chat/ web/src/pages/Pipelines.tsx
git commit -m "feat: add floating AI assistant widget"
```

---

### Task 12: Final cleanup and verification

**Step 1: Remove unused components**

If `PipelineSettingsPanel` is no longer imported anywhere, mark it or remove it.
If `SessionDetailPreview` is no longer imported, same.

**Step 2: Run full test suite**

Run: `make test && make test-frontend`

**Step 3: Manual verification**

1. Create a pipeline (name + description only)
2. Create a session within the pipeline
3. Configure session: add sources, set schedule, set editorial brief, select model, add workflows
4. Start session → verify collection + analysis runs
5. Check inbox → verify session appears with analysis (no workflow selection UI)
6. Approve → verify auto-produce starts with configured workflows
7. Verify scheduled collection creates new inbox items

**Step 4: Commit**

```bash
git add -A
git commit -m "chore: cleanup unused components after session setup redesign"
```
