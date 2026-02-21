# Pipeline Approve/Reject Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement approve/reject for paused pipeline runs — run-level URL, async resume via goroutine, immediate 202 response.

**Architecture:** Extract `executeFrom(ctx, pipeline, run, startIdx)` helper from `Start()`, add `Resume()` that finds current stage index and calls `executeFrom` from the next stage. API handlers look up the waiting run, update status, fire a goroutine, and return immediately.

**Tech Stack:** Go 1.23, Chi router, React 19 + TypeScript

---

### Task 1: Refactor `PipelineRunner` — extract `executeFrom`, add `Resume`

**Files:**
- Modify: `internal/services/pipeline_runner.go`
- Test: `internal/services/pipeline_runner_test.go`

**Step 1: Write failing tests for `Resume`**

Add to `internal/services/pipeline_runner_test.go`:

```go
// mockWaitingExecutor returns "waiting" on first call, "completed" never (approval gate).
type mockWaitingExecutor struct {
	stageType string
	calls     []string
}

func (m *mockWaitingExecutor) Type() string { return m.stageType }
func (m *mockWaitingExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	m.calls = append(m.calls, stage.ID)
	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "waiting",
		Output:  map[string]any{"message": "please approve"},
	}, nil
}

func TestPipelineRunner_Resume(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	approvalExec := &mockWaitingExecutor{stageType: "approval"}
	wfExec := &mockStageExecutor{stageType: "workflow", output: map[string]any{"done": true}}

	runner := NewPipelineRunner(runRepo)
	runner.RegisterExecutor(approvalExec)
	runner.RegisterExecutor(wfExec)

	pipeline := &upal.Pipeline{
		ID:   "pipe-resume",
		Name: "Resume Test",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Collect", Type: "workflow"},
			{ID: "s2", Name: "Approve", Type: "approval"},
			{ID: "s3", Name: "Process", Type: "workflow"},
		},
	}

	// Start: should pause at s2
	run, err := runner.Start(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if run.Status != "waiting" {
		t.Fatalf("expected status 'waiting', got %q", run.Status)
	}
	if run.CurrentStage != "s2" {
		t.Errorf("expected current_stage 's2', got %q", run.CurrentStage)
	}
	if len(wfExec.calls) != 1 || wfExec.calls[0] != "s1" {
		t.Errorf("expected s1 executed before pause, got %v", wfExec.calls)
	}

	// Resume: should skip s2 (already done), execute s3
	wfExec.calls = nil
	err = runner.Resume(context.Background(), pipeline, run)
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("expected status 'completed' after resume, got %q", run.Status)
	}
	if len(wfExec.calls) != 1 || wfExec.calls[0] != "s3" {
		t.Errorf("expected only s3 executed after resume, got %v", wfExec.calls)
	}
	// approval executor must NOT be called again
	if len(approvalExec.calls) != 1 {
		t.Errorf("approval executor should have been called exactly once, got %d", len(approvalExec.calls))
	}
}

func TestPipelineRunner_Resume_CurrentStageNotFound(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	runner := NewPipelineRunner(runRepo)

	pipeline := &upal.Pipeline{
		ID:     "pipe-bad",
		Name:   "Bad",
		Stages: []upal.Stage{{ID: "s1", Type: "workflow"}},
	}
	run := &upal.PipelineRun{
		ID:           "prun-bad",
		PipelineID:   "pipe-bad",
		Status:       "waiting",
		CurrentStage: "nonexistent",
		StageResults: make(map[string]*upal.StageResult),
	}
	if err := runRepo.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	err := runner.Resume(context.Background(), pipeline, run)
	if err == nil {
		t.Fatal("expected error for nonexistent current stage")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/services/... -v -race -run TestPipelineRunner_Resume
```
Expected: FAIL — `runner.Resume undefined`

**Step 3: Refactor `pipeline_runner.go`**

Replace the entire file with:

```go
// internal/services/pipeline_runner.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// StageExecutor is the interface for executing a pipeline stage.
// Implement this interface to add new stage types.
type StageExecutor interface {
	Type() string
	Execute(ctx context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error)
}

// PipelineRunner orchestrates sequential execution of pipeline stages.
type PipelineRunner struct {
	executors map[string]StageExecutor
	runRepo   repository.PipelineRunRepository
}

func NewPipelineRunner(runRepo repository.PipelineRunRepository) *PipelineRunner {
	return &PipelineRunner{
		executors: make(map[string]StageExecutor),
		runRepo:   runRepo,
	}
}

func (r *PipelineRunner) RegisterExecutor(exec StageExecutor) {
	r.executors[exec.Type()] = exec
}

func (r *PipelineRunner) Start(ctx context.Context, pipeline *upal.Pipeline) (*upal.PipelineRun, error) {
	run := &upal.PipelineRun{
		ID:           upal.GenerateID("prun"),
		PipelineID:   pipeline.ID,
		Status:       "running",
		StageResults: make(map[string]*upal.StageResult),
		StartedAt:    time.Now(),
	}
	r.runRepo.Create(ctx, run)
	if err := r.executeFrom(ctx, pipeline, run, 0); err != nil {
		return run, err
	}
	return run, nil
}

// Resume continues a paused pipeline run from the stage after run.CurrentStage.
func (r *PipelineRunner) Resume(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun) error {
	currentIdx := -1
	for i, stage := range pipeline.Stages {
		if stage.ID == run.CurrentStage {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		return fmt.Errorf("current stage %q not found in pipeline %q", run.CurrentStage, pipeline.ID)
	}
	return r.executeFrom(ctx, pipeline, run, currentIdx+1)
}

// executeFrom runs pipeline stages sequentially starting from startIdx.
// It updates run in the repository at each transition.
func (r *PipelineRunner) executeFrom(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun, startIdx int) error {
	// Seed prevResult from the last completed stage before startIdx.
	var prevResult *upal.StageResult
	for i := 0; i < startIdx; i++ {
		stage := pipeline.Stages[i]
		if result, ok := run.StageResults[stage.ID]; ok && result.Status == "completed" {
			prevResult = result
		}
	}

	for i := startIdx; i < len(pipeline.Stages); i++ {
		stage := pipeline.Stages[i]

		executor, ok := r.executors[stage.Type]
		if !ok {
			now := time.Now()
			run.Status = "failed"
			run.CompletedAt = &now
			r.runRepo.Update(ctx, run)
			return fmt.Errorf("no executor registered for stage type %q", stage.Type)
		}

		run.CurrentStage = stage.ID
		stageResult := &upal.StageResult{
			StageID:   stage.ID,
			Status:    "running",
			StartedAt: time.Now(),
		}
		run.StageResults[stage.ID] = stageResult
		r.runRepo.Update(ctx, run)

		result, err := executor.Execute(ctx, stage, prevResult)
		if err != nil {
			now := time.Now()
			stageResult.Status = "failed"
			stageResult.Error = err.Error()
			stageResult.CompletedAt = &now
			run.Status = "failed"
			run.CompletedAt = &now
			r.runRepo.Update(ctx, run)
			return fmt.Errorf("stage %q failed: %w", stage.ID, err)
		}

		if result.Status == "waiting" {
			run.Status = "waiting"
			run.StageResults[stage.ID] = result
			r.runRepo.Update(ctx, run)
			return nil
		}

		now := time.Now()
		result.CompletedAt = &now
		run.StageResults[stage.ID] = result
		r.runRepo.Update(ctx, run)

		prevResult = result
	}

	now := time.Now()
	run.Status = "completed"
	run.CompletedAt = &now
	r.runRepo.Update(ctx, run)
	return nil
}
```

**Step 4: Run all runner tests**

```bash
go test ./internal/services/... -v -race -run TestPipelineRunner
```
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/services/pipeline_runner.go internal/services/pipeline_runner_test.go
git commit -m "refactor: extract executeFrom, add Resume to PipelineRunner"
```

---

### Task 2: Implement approve/reject API handlers + update routes

**Files:**
- Modify: `internal/api/pipelines.go`
- Modify: `internal/api/server.go`

**Step 1: Write failing handler tests**

Check if `internal/api/pipeline_test.go` (or similar) exists. If not, create `internal/api/pipelines_test.go`:

```go
// internal/api/pipelines_test.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// noopStageExecutor completes any stage immediately.
type noopStageExecutor struct{ t string }

func (n *noopStageExecutor) Type() string { return n.t }
func (n *noopStageExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	return &upal.StageResult{StageID: stage.ID, Status: "completed"}, nil
}

func newTestPipelineServer(t *testing.T) (*Server, *repository.MemoryPipelineRepository, *repository.MemoryPipelineRunRepository) {
	t.Helper()
	pipelineRepo := repository.NewMemoryPipelineRepository()
	runRepo := repository.NewMemoryPipelineRunRepository()
	pipelineSvc := services.NewPipelineService(pipelineRepo, runRepo)
	runner := services.NewPipelineRunner(runRepo)
	runner.RegisterExecutor(&noopStageExecutor{"workflow"})

	srv := &Server{}
	srv.SetPipelineService(pipelineSvc)
	srv.SetPipelineRunner(runner)
	return srv, pipelineRepo, runRepo
}

func TestApprovePipelineRun(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	// Seed a pipeline
	pipeline := &upal.Pipeline{
		ID:   "pipe-1",
		Name: "Test",
		Stages: []upal.Stage{
			{ID: "s1", Type: "workflow"},
			{ID: "s2", Type: "approval"},
			{ID: "s3", Type: "workflow"},
		},
	}
	pipelineRepo.Create(context.Background(), pipeline)

	// Seed a waiting run
	run := &upal.PipelineRun{
		ID:           "prun-1",
		PipelineID:   "pipe-1",
		Status:       "waiting",
		CurrentStage: "s2",
		StageResults: map[string]*upal.StageResult{
			"s1": {StageID: "s1", Status: "completed", StartedAt: time.Now()},
			"s2": {StageID: "s2", Status: "waiting", StartedAt: time.Now()},
		},
		StartedAt: time.Now(),
	}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/api/pipelines/pipe-1/runs/prun-1/approve", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-1")
	rctx.URLParams.Add("runId", "prun-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.approvePipelineRun(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp upal.PipelineRun
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "running" {
		t.Errorf("expected run status 'running' in response, got %q", resp.Status)
	}
}

func TestRejectPipelineRun(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	pipeline := &upal.Pipeline{
		ID:   "pipe-2",
		Name: "Reject Test",
		Stages: []upal.Stage{
			{ID: "s1", Type: "approval"},
		},
	}
	pipelineRepo.Create(context.Background(), pipeline)

	run := &upal.PipelineRun{
		ID:           "prun-2",
		PipelineID:   "pipe-2",
		Status:       "waiting",
		CurrentStage: "s1",
		StageResults: map[string]*upal.StageResult{
			"s1": {StageID: "s1", Status: "waiting", StartedAt: time.Now()},
		},
		StartedAt: time.Now(),
	}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/api/pipelines/pipe-2/runs/prun-2/reject", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-2")
	rctx.URLParams.Add("runId", "prun-2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.rejectPipelineRun(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// Verify persisted as failed
	updated, _ := runRepo.Get(context.Background(), "prun-2")
	if updated.Status != "failed" {
		t.Errorf("expected persisted status 'failed', got %q", updated.Status)
	}
}

func TestApprovePipelineRun_NotWaiting(t *testing.T) {
	srv, pipelineRepo, runRepo := newTestPipelineServer(t)

	pipeline := &upal.Pipeline{ID: "pipe-3", Name: "P3", Stages: []upal.Stage{{ID: "s1", Type: "workflow"}}}
	pipelineRepo.Create(context.Background(), pipeline)

	run := &upal.PipelineRun{ID: "prun-3", PipelineID: "pipe-3", Status: "completed", StageResults: map[string]*upal.StageResult{}}
	runRepo.Create(context.Background(), run)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "pipe-3")
	rctx.URLParams.Add("runId", "prun-3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	srv.approvePipelineRun(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/api/... -v -race -run TestApprovePipeline
```
Expected: FAIL — `srv.approvePipelineRun undefined`

**Step 3: Implement handlers in `internal/api/pipelines.go`**

Replace lines 120–130 (the two TODO stubs) with:

```go
func (s *Server) approvePipelineRun(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	runID := chi.URLParam(r, "runId")

	p, err := s.pipelineSvc.Get(r.Context(), pipelineID)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	run, err := s.pipelineSvc.GetRun(r.Context(), runID)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.Status != "waiting" {
		http.Error(w, "run is not waiting for approval", http.StatusBadRequest)
		return
	}

	run.Status = "running"
	if err := s.pipelineSvc.UpdateRun(r.Context(), run); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		if err := s.pipelineRunner.Resume(context.Background(), p, run); err != nil {
			// Resume persists failure to the run itself; log if needed.
			_ = err
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(run)
}

func (s *Server) rejectPipelineRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")

	run, err := s.pipelineSvc.GetRun(r.Context(), runID)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.Status != "waiting" {
		http.Error(w, "run is not waiting for approval", http.StatusBadRequest)
		return
	}

	now := time.Now()
	run.Status = "failed"
	run.CompletedAt = &now
	if result, ok := run.StageResults[run.CurrentStage]; ok {
		result.Status = "failed"
		result.Error = "rejected by user"
		result.CompletedAt = &now
	}
	if err := s.pipelineSvc.UpdateRun(r.Context(), run); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}
```

Also add `"context"` and `"time"` to the import block at the top of `pipelines.go`:

```go
import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/soochol/upal/internal/upal"
)
```

**Step 4: Update routes in `internal/api/server.go`**

Find the two old routes (lines ~109–110):
```go
r.Post("/{id}/stages/{stageId}/approve", s.approvePipelineStage)
r.Post("/{id}/stages/{stageId}/reject", s.rejectPipelineStage)
```

Replace with:
```go
r.Post("/{id}/runs/{runId}/approve", s.approvePipelineRun)
r.Post("/{id}/runs/{runId}/reject", s.rejectPipelineRun)
```

**Step 5: Run all API tests**

```bash
go test ./internal/api/... -v -race -run TestApprovePipeline
go test ./internal/api/... -v -race -run TestRejectPipeline
go test ./internal/api/... -v -race -run TestApprovePipelineRun_NotWaiting
```
Expected: All PASS

**Step 6: Run full test suite**

```bash
go test ./... -race
```
Expected: All PASS (no compilation errors)

**Step 7: Commit**

```bash
git add internal/api/pipelines.go internal/api/server.go internal/api/pipelines_test.go
git commit -m "feat: implement approve/reject for pipeline runs with async resume"
```

---

### Task 3: Update frontend API client

**Files:**
- Modify: `web/src/lib/api/pipelines.ts`

**Step 1: Replace stage-based functions with run-based**

In `web/src/lib/api/pipelines.ts`, find and remove:

```ts
export async function approvePipelineStage(pipelineId: string, stageId: string): Promise<void> {
  await apiFetch(`/api/pipelines/${encodeURIComponent(pipelineId)}/stages/${encodeURIComponent(stageId)}/approve`, {
    method: 'POST',
  })
}

export async function rejectPipelineStage(pipelineId: string, stageId: string): Promise<void> {
  await apiFetch(`/api/pipelines/${encodeURIComponent(pipelineId)}/stages/${encodeURIComponent(stageId)}/reject`, {
    method: 'POST',
  })
}
```

Replace with:

```ts
export async function approvePipelineRun(pipelineId: string, runId: string): Promise<void> {
  await apiFetch(`/api/pipelines/${encodeURIComponent(pipelineId)}/runs/${encodeURIComponent(runId)}/approve`, {
    method: 'POST',
  })
}

export async function rejectPipelineRun(pipelineId: string, runId: string): Promise<void> {
  await apiFetch(`/api/pipelines/${encodeURIComponent(pipelineId)}/runs/${encodeURIComponent(runId)}/reject`, {
    method: 'POST',
  })
}
```

**Step 2: Type-check**

```bash
cd web && npx tsc -b 2>&1 | head -30
```
Expected: Errors referencing `approvePipelineStage`/`rejectPipelineStage` usage in `PipelineRunHistory.tsx` (to be fixed in Task 4)

**Step 3: Commit after Task 4 passes type check** (defer to Task 4 step 4)

---

### Task 4: Update `PipelineRunHistory` component

**Files:**
- Modify: `web/src/components/pipelines/PipelineRunHistory.tsx`

**Step 1: Update import and handler calls**

In `PipelineRunHistory.tsx`:

Change the import line from:
```ts
import { fetchPipelineRuns, approvePipelineStage, rejectPipelineStage } from '@/lib/api/pipelines'
```
To:
```ts
import { fetchPipelineRuns, approvePipelineRun, rejectPipelineRun } from '@/lib/api/pipelines'
```

**Step 2: Update `handleApprove` and `handleReject`**

Find (around lines 46–54):
```ts
const handleApprove = async (run: PipelineRun, stageId: string) => {
  await approvePipelineStage(pipeline.id, stageId)
  reload()
}

const handleReject = async (run: PipelineRun, stageId: string) => {
  await rejectPipelineStage(pipeline.id, stageId)
  reload()
}
```

Replace with:
```ts
const handleApprove = async (run: PipelineRun) => {
  await approvePipelineRun(pipeline.id, run.id)
  reload()
}

const handleReject = async (run: PipelineRun) => {
  await rejectPipelineRun(pipeline.id, run.id)
  reload()
}
```

**Step 3: Update call sites in the render**

Find (around lines 111–122) the Approve/Reject buttons:
```tsx
onClick={() => handleApprove(run, run.current_stage!)}
...
onClick={() => handleReject(run, run.current_stage!)}
```

Replace with:
```tsx
onClick={() => handleApprove(run)}
...
onClick={() => handleReject(run)}
```

**Step 4: Type-check frontend**

```bash
cd web && npx tsc -b
```
Expected: No errors

**Step 5: Commit both frontend files**

```bash
git add web/src/lib/api/pipelines.ts web/src/components/pipelines/PipelineRunHistory.tsx
git commit -m "feat: update pipeline frontend to use run-level approve/reject URLs"
```

---

### Task 5: Final verification

**Step 1: Full Go test suite**

```bash
go test ./... -race
```
Expected: All PASS

**Step 2: Frontend build**

```bash
cd web && npm run build
```
Expected: Build succeeds, no TypeScript errors

**Step 3: Smoke test (manual)**

```bash
make dev-backend   # terminal 1
make dev-frontend  # terminal 2
```

1. Create a pipeline with stages: `workflow → approval → workflow`
2. Start it — should return `202 Accepted`, status `waiting`
3. Open the Pipelines page, see the run paused at the approval stage
4. Click **Approve** — run resumes, eventually status becomes `completed`
5. Create another run, click **Reject** — status becomes `failed`
