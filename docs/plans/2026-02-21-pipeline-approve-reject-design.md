# Pipeline Approve/Reject — Design Doc

**Date**: 2026-02-21
**Status**: Approved
**Scope**: Complete unimplemented approve/reject flow for pipeline runs

---

## Problem

The pipeline runner supports `approval` stages that pause execution and return `status="waiting"`. However, the approve and reject API endpoints are stubs. The current URL design is also semantically incorrect — approval is an action on a **run**, not a stage.

Additionally, the existing `Start()` method contains the execution loop inline, making resume impossible without duplication.

---

## Solution: Option C — Run-level Action + Async Resume

### API Changes

**Remove** (from `internal/api/pipelines.go` and `internal/api/server.go`):
```
POST /api/pipelines/{id}/stages/{stageId}/approve
POST /api/pipelines/{id}/stages/{stageId}/reject
```

**Add**:
```
POST /api/pipelines/{id}/runs/{runId}/approve   → 202 Accepted (async)
POST /api/pipelines/{id}/runs/{runId}/reject    → 200 OK (sync)
```

**Approve handler**:
1. Look up run by `runId`, validate `status == "waiting"`
2. Set `run.Status = "running"`, persist immediately
3. Spawn goroutine: `runner.Resume(ctx, pipeline, run)`
4. Return `202 Accepted` with the run object

**Reject handler**:
1. Look up run by `runId`, validate `status == "waiting"`
2. Set `run.Status = "failed"`, set error message `"rejected by user"`, persist
3. Return `200 OK` with the run object

---

### Runner Changes (`internal/services/pipeline_runner.go`)

Extract inner loop from `Start()` into a shared private helper:

```go
func (r *PipelineRunner) executeFrom(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun, startIdx int) error
```

Refactor `Start()` to call `executeFrom(ctx, pipeline, run, 0)`.

Add public `Resume()`:

```go
func (r *PipelineRunner) Resume(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun) error
```

`Resume()` finds `currentStage` index in `pipeline.Stages`, then calls `executeFrom(ctx, pipeline, run, currentStageIdx+1)`.

---

### Frontend Changes

**`web/src/lib/api/pipelines.ts`**:
- Remove `approvePipelineStage(pipelineId, stageId)`
- Remove `rejectPipelineStage(pipelineId, stageId)`
- Add `approvePipelineRun(pipelineId, runId)` → `POST /api/pipelines/{pipelineId}/runs/{runId}/approve`
- Add `rejectPipelineRun(pipelineId, runId)` → `POST /api/pipelines/{pipelineId}/runs/{runId}/reject`

**`web/src/components/pipelines/PipelineRunHistory.tsx`**:
- Replace `approvePipelineStage(pipeline.id, run.current_stage)` with `approvePipelineRun(pipeline.id, run.id)`
- Replace `rejectPipelineStage(pipeline.id, run.current_stage)` with `rejectPipelineRun(pipeline.id, run.id)`

---

## Data Flow

```
User clicks "Approve"
  → POST /api/pipelines/{id}/runs/{runId}/approve
  → Handler: run.status = "running" (persisted), return 202
  → goroutine: Resume(ctx, pipeline, run)
      → find currentStageIdx in pipeline.Stages
      → executeFrom(ctx, pipeline, run, currentStageIdx + 1)
          → continue remaining stages sequentially
          → run.status = "completed" on finish
  → Frontend polls: fetchPipelineRuns() via reload()
      → shows updated status

User clicks "Reject"
  → POST /api/pipelines/{id}/runs/{runId}/reject
  → Handler: run.status = "failed", error = "rejected by user"
  → return 200
  → Frontend: reload() shows failed status
```

---

## Error Handling

- Run not found → 404
- Run not in `waiting` status → 400 with message "run is not waiting for approval"
- Resume goroutine error → persisted as `run.Status = "failed"`, `run.Error = err.Error()`

---

## Files Changed

| File | Change |
|------|--------|
| `internal/api/pipelines.go` | Implement approve/reject handlers, update routes |
| `internal/api/server.go` | Update route paths |
| `internal/services/pipeline_runner.go` | Extract `executeFrom`, add `Resume` |
| `web/src/lib/api/pipelines.ts` | Replace stage-based functions with run-based |
| `web/src/components/pipelines/PipelineRunHistory.tsx` | Use run.id instead of run.current_stage |
