# Pipeline DB Persistence — Design Doc

**Date**: 2026-02-21
**Status**: Approved
**Scope**: Persist pipelines and pipeline_runs to PostgreSQL, following existing graceful-fallback pattern

---

## Problem

Pipelines and pipeline runs are stored in memory only (`MemoryPipelineRepository`, `MemoryPipelineRunRepository`). All data is lost on server restart. The UI cannot load previously created pipelines or run history after restart.

---

## Solution: Option A — Graceful Fallback (matches existing pattern)

Follow the identical pattern used for workflows, runs, schedules, and triggers:
- DB absent → memory only (graceful fallback, same as today)
- DB present → writes go to memory + DB simultaneously; reads hit memory first, fall back to DB with caching

---

## New Files

### `internal/db/pipeline.go`

DB-layer CRUD methods on `*DB`:

**Pipelines:**
- `CreatePipeline(ctx, *upal.Pipeline) error`
- `GetPipeline(ctx, id string) (*upal.Pipeline, error)`
- `ListPipelines(ctx) ([]*upal.Pipeline, error)`
- `UpdatePipeline(ctx, *upal.Pipeline) error`
- `DeletePipeline(ctx, id string) error`

**Pipeline Runs:**
- `CreatePipelineRun(ctx, *upal.PipelineRun) error`
- `GetPipelineRun(ctx, id string) (*upal.PipelineRun, error)`
- `ListPipelineRunsByPipeline(ctx, pipelineID string) ([]*upal.PipelineRun, error)`
- `UpdatePipelineRun(ctx, *upal.PipelineRun) error`

Storage: `stages` and `stage_results` stored as JSONB.

### `internal/repository/pipeline_persistent.go`

Two persistent wrappers:

**`PersistentPipelineRepository`** implements `PipelineRepository`:
- `Create`: write to memory (ignore error) + write to DB
- `Get`: try memory → on miss, query DB + cache result in memory
- `List`: query DB (prefer durable source) → on DB failure, fall back to memory with `slog.Warn`
- `Update`: write to memory + write to DB
- `Delete`: delete from memory + delete from DB

**`PersistentPipelineRunRepository`** implements `PipelineRunRepository`:
- Same read/write pattern
- `ListByPipeline`: query DB → on DB failure, fall back to memory

---

## Modified Files

### `internal/db/db.go` — add to `migrationSQL`

```sql
CREATE TABLE IF NOT EXISTS pipelines (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    stages      JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id            TEXT PRIMARY KEY,
    pipeline_id   TEXT NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending',
    current_stage TEXT NOT NULL DEFAULT '',
    stage_results JSONB NOT NULL DEFAULT '{}',
    started_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_pipeline_id ON pipeline_runs(pipeline_id);
```

### `cmd/upal/main.go` — wrap repos when DB is available

```go
// Before (current):
pipelineRepo := repository.NewMemoryPipelineRepository()
pipelineRunRepo := repository.NewMemoryPipelineRunRepository()

// After:
memPipelineRepo := repository.NewMemoryPipelineRepository()
var pipelineRepo repository.PipelineRepository = memPipelineRepo
if database != nil {
    pipelineRepo = repository.NewPersistentPipelineRepository(memPipelineRepo, database)
}

memPipelineRunRepo := repository.NewMemoryPipelineRunRepository()
var pipelineRunRepo repository.PipelineRunRepository = memPipelineRunRepo
if database != nil {
    pipelineRunRepo = repository.NewPersistentPipelineRunRepository(memPipelineRunRepo, database)
}
```

---

## Data Flow on Server Start

1. `main.go` runs `database.Migrate()` → creates `pipelines` and `pipeline_runs` tables if not exist
2. `PersistentPipelineRepository` initialized with empty memory store
3. First `List()` call (from UI loading `/api/pipelines`) hits DB → results cached in memory
4. Subsequent reads hit memory (fast path)

---

## Files Changed

| File | Type | Change |
|------|------|--------|
| `internal/db/pipeline.go` | New | DB CRUD for pipelines + pipeline_runs |
| `internal/repository/pipeline_persistent.go` | New | Persistent wrappers for both repos |
| `internal/db/db.go` | Modify | Add 2 tables + 1 index to migrationSQL |
| `cmd/upal/main.go` | Modify | Wrap repos with persistent when DB available |
