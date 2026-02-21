# Pipeline DB Persistence Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Persist pipelines and pipeline_runs to PostgreSQL so the UI can load them after server restart.

**Architecture:** Follow the existing two-tier pattern — `MemoryRepo` is always active; when a DB connection is configured, a `PersistentRepo` wrapper writes to both stores and reads from memory first (DB as fallback + cache-warmer). Startup migration automatically creates the new tables.

**Tech Stack:** Go 1.23, PostgreSQL (`database/sql` + `github.com/lib/pq`), JSONB for complex fields

---

### Task 1: Add migration SQL for pipelines and pipeline_runs tables

**Files:**
- Modify: `internal/db/db.go:129-130`

**Step 1: Add the two new tables to `migrationSQL`**

In `internal/db/db.go`, find the closing backtick of `migrationSQL` (currently line 130). Insert the new tables before the closing backtick:

```go
// Replace the closing of migrationSQL:
);
`

// With:
);

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
`
```

The full end of `migrationSQL` should look like:

```
);

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
`
```

**Step 2: Verify the file compiles**

```bash
cd /home/dev/code/Upal && go build ./internal/db/...
```
Expected: no output (success)

**Step 3: Commit**

```bash
git add internal/db/db.go
git commit -m "feat: add pipelines and pipeline_runs migration tables"
```

---

### Task 2: Create `internal/db/pipeline.go` — DB CRUD layer

**Files:**
- Create: `internal/db/pipeline.go`

**Step 1: Write the file**

```go
// internal/db/pipeline.go
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// CreatePipeline inserts a new pipeline. stages is stored as JSONB.
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

// GetPipeline retrieves a pipeline by ID.
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

// ListPipelines returns all pipelines ordered by updated_at descending.
func (d *DB) ListPipelines(ctx context.Context) ([]*upal.Pipeline, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, name, description, stages, created_at, updated_at
		 FROM pipelines ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}
	defer rows.Close()

	var result []*upal.Pipeline
	for rows.Next() {
		var p upal.Pipeline
		var stagesJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		if err := json.Unmarshal(stagesJSON, &p.Stages); err != nil {
			return nil, fmt.Errorf("unmarshal stages: %w", err)
		}
		result = append(result, &p)
	}
	return result, nil
}

// UpdatePipeline updates an existing pipeline's name, description, stages, and updated_at.
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

// DeletePipeline removes a pipeline by ID. Cascade deletes pipeline_runs.
func (d *DB) DeletePipeline(ctx context.Context, id string) error {
	res, err := d.Pool.ExecContext(ctx, `DELETE FROM pipelines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline %q not found", id)
	}
	return nil
}

// CreatePipelineRun inserts a new pipeline run.
func (d *DB) CreatePipelineRun(ctx context.Context, run *upal.PipelineRun) error {
	stageResultsJSON, err := json.Marshal(run.StageResults)
	if err != nil {
		return fmt.Errorf("marshal stage_results: %w", err)
	}
	_, err = d.Pool.ExecContext(ctx,
		`INSERT INTO pipeline_runs (id, pipeline_id, status, current_stage, stage_results, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		run.ID, run.PipelineID, run.Status, run.CurrentStage, stageResultsJSON, run.StartedAt, run.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline_run: %w", err)
	}
	return nil
}

// GetPipelineRun retrieves a pipeline run by ID.
func (d *DB) GetPipelineRun(ctx context.Context, id string) (*upal.PipelineRun, error) {
	var run upal.PipelineRun
	var stageResultsJSON []byte
	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, pipeline_id, status, current_stage, stage_results, started_at, completed_at
		 FROM pipeline_runs WHERE id = $1`, id,
	).Scan(&run.ID, &run.PipelineID, &run.Status, &run.CurrentStage, &stageResultsJSON, &run.StartedAt, &run.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("pipeline run %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get pipeline_run: %w", err)
	}
	if err := json.Unmarshal(stageResultsJSON, &run.StageResults); err != nil {
		return nil, fmt.Errorf("unmarshal stage_results: %w", err)
	}
	return &run, nil
}

// ListPipelineRunsByPipeline returns all runs for a pipeline ordered by started_at descending.
func (d *DB) ListPipelineRunsByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	rows, err := d.Pool.QueryContext(ctx,
		`SELECT id, pipeline_id, status, current_stage, stage_results, started_at, completed_at
		 FROM pipeline_runs WHERE pipeline_id = $1 ORDER BY started_at DESC`,
		pipelineID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline_runs: %w", err)
	}
	defer rows.Close()

	var result []*upal.PipelineRun
	for rows.Next() {
		var run upal.PipelineRun
		var stageResultsJSON []byte
		if err := rows.Scan(&run.ID, &run.PipelineID, &run.Status, &run.CurrentStage, &stageResultsJSON, &run.StartedAt, &run.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan pipeline_run: %w", err)
		}
		if err := json.Unmarshal(stageResultsJSON, &run.StageResults); err != nil {
			return nil, fmt.Errorf("unmarshal stage_results: %w", err)
		}
		result = append(result, &run)
	}
	return result, nil
}

// UpdatePipelineRun updates an existing pipeline run's mutable fields.
func (d *DB) UpdatePipelineRun(ctx context.Context, run *upal.PipelineRun) error {
	stageResultsJSON, err := json.Marshal(run.StageResults)
	if err != nil {
		return fmt.Errorf("marshal stage_results: %w", err)
	}
	res, err := d.Pool.ExecContext(ctx,
		`UPDATE pipeline_runs
		 SET status = $1, current_stage = $2, stage_results = $3, completed_at = $4
		 WHERE id = $5`,
		run.Status, run.CurrentStage, stageResultsJSON, run.CompletedAt, run.ID,
	)
	if err != nil {
		return fmt.Errorf("update pipeline_run: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("pipeline run %q not found", run.ID)
	}
	return nil
}

// Ensure time.Time is used (suppress unused import warning if needed).
var _ = time.Now
```

**Step 2: Build to verify**

```bash
cd /home/dev/code/Upal && go build ./internal/db/...
```
Expected: no output

**Step 3: Commit**

```bash
git add internal/db/pipeline.go
git commit -m "feat: add DB CRUD layer for pipelines and pipeline_runs"
```

---

### Task 3: Create `internal/repository/pipeline_persistent.go`

**Files:**
- Create: `internal/repository/pipeline_persistent.go`

**Step 1: Write failing tests**

Create `internal/repository/pipeline_persistent_test.go`:

```go
// internal/repository/pipeline_persistent_test.go
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// stubPipelineDB is a fake DB that records calls and returns canned data.
type stubPipelineDB struct {
	pipelines []*upal.Pipeline
	runs      []*upal.PipelineRun
	createErr error
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
}

func (s *stubPipelineDB) CreatePipeline(_ context.Context, p *upal.Pipeline) error {
	s.pipelines = append(s.pipelines, p)
	return s.createErr
}
func (s *stubPipelineDB) GetPipeline(_ context.Context, id string) (*upal.Pipeline, error) {
	for _, p := range s.pipelines {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, s.getErr
}
func (s *stubPipelineDB) ListPipelines(_ context.Context) ([]*upal.Pipeline, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.pipelines, nil
}
func (s *stubPipelineDB) UpdatePipeline(_ context.Context, p *upal.Pipeline) error {
	return s.updateErr
}
func (s *stubPipelineDB) DeletePipeline(_ context.Context, _ string) error {
	return s.deleteErr
}
func (s *stubPipelineDB) CreatePipelineRun(_ context.Context, r *upal.PipelineRun) error {
	s.runs = append(s.runs, r)
	return s.createErr
}
func (s *stubPipelineDB) GetPipelineRun(_ context.Context, id string) (*upal.PipelineRun, error) {
	for _, r := range s.runs {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, s.getErr
}
func (s *stubPipelineDB) ListPipelineRunsByPipeline(_ context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var out []*upal.PipelineRun
	for _, r := range s.runs {
		if r.PipelineID == pipelineID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (s *stubPipelineDB) UpdatePipelineRun(_ context.Context, _ *upal.PipelineRun) error {
	return s.updateErr
}

func newTestPipeline(id string) *upal.Pipeline {
	return &upal.Pipeline{
		ID:        id,
		Name:      "Pipeline " + id,
		Stages:    []upal.Stage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestPersistentPipelineRepository_CreateAndGet(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	stub := &stubPipelineDB{}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	p := newTestPipeline("pipe-1")
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Should be in memory
	got, err := repo.Get(context.Background(), "pipe-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != "pipe-1" {
		t.Errorf("expected pipe-1, got %s", got.ID)
	}

	// DB should have been called
	if len(stub.pipelines) != 1 {
		t.Errorf("expected 1 pipeline in DB stub, got %d", len(stub.pipelines))
	}
}

func TestPersistentPipelineRepository_GetFallsBackToDb(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	p := newTestPipeline("pipe-db")
	stub := &stubPipelineDB{pipelines: []*upal.Pipeline{p}}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	// Memory is empty, DB has the pipeline
	got, err := repo.Get(context.Background(), "pipe-db")
	if err != nil {
		t.Fatalf("Get fallback failed: %v", err)
	}
	if got.ID != "pipe-db" {
		t.Errorf("expected pipe-db, got %s", got.ID)
	}
}

func TestPersistentPipelineRepository_ListPrefersDb(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	p1 := newTestPipeline("pipe-1")
	p2 := newTestPipeline("pipe-2")
	stub := &stubPipelineDB{pipelines: []*upal.Pipeline{p1, p2}}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 pipelines from DB, got %d", len(list))
	}
}

func TestPersistentPipelineRepository_ListFallsBackToMemory(t *testing.T) {
	mem := repository.NewMemoryPipelineRepository()
	p := newTestPipeline("pipe-mem")
	_ = mem.Create(context.Background(), p)
	stub := &stubPipelineDB{listErr: errFake}
	repo := repository.NewPersistentPipelineRepository(mem, stub)

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List memory fallback failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != "pipe-mem" {
		t.Errorf("expected memory fallback with pipe-mem, got %v", list)
	}
}

func TestPersistentPipelineRunRepository_CreateAndListByPipeline(t *testing.T) {
	mem := repository.NewMemoryPipelineRunRepository()
	stub := &stubPipelineDB{}
	repo := repository.NewPersistentPipelineRunRepository(mem, stub)

	run := &upal.PipelineRun{
		ID:           "prun-1",
		PipelineID:   "pipe-1",
		Status:       "running",
		StageResults: map[string]*upal.StageResult{},
		StartedAt:    time.Now(),
	}
	if err := repo.Create(context.Background(), run); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	list, err := repo.ListByPipeline(context.Background(), "pipe-1")
	if err != nil {
		t.Fatalf("ListByPipeline failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 run, got %d", len(list))
	}
}
```

Note: `errFake` needs to be defined. Add to the test file:
```go
import "errors"
var errFake = errors.New("fake db error")
```

**Step 2: Run tests to verify they fail**

```bash
cd /home/dev/code/Upal && go test ./internal/repository/... -v -race -run "TestPersistentPipeline"
```
Expected: FAIL — `repository.NewPersistentPipelineRepository undefined`

**Step 3: Create `internal/repository/pipeline_persistent.go`**

```go
// internal/repository/pipeline_persistent.go
package repository

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// PipelineDB defines the DB-layer methods needed by the persistent pipeline repo.
// *db.DB satisfies this interface.
type PipelineDB interface {
	CreatePipeline(ctx context.Context, p *upal.Pipeline) error
	GetPipeline(ctx context.Context, id string) (*upal.Pipeline, error)
	ListPipelines(ctx context.Context) ([]*upal.Pipeline, error)
	UpdatePipeline(ctx context.Context, p *upal.Pipeline) error
	DeletePipeline(ctx context.Context, id string) error
}

// PipelineRunDB defines the DB-layer methods needed by the persistent run repo.
type PipelineRunDB interface {
	CreatePipelineRun(ctx context.Context, run *upal.PipelineRun) error
	GetPipelineRun(ctx context.Context, id string) (*upal.PipelineRun, error)
	ListPipelineRunsByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error)
	UpdatePipelineRun(ctx context.Context, run *upal.PipelineRun) error
}

// PersistentPipelineRepository wraps MemoryPipelineRepository with a PostgreSQL backend.
// Writes go to both. Reads try memory first; on miss, fall back to DB and cache.
type PersistentPipelineRepository struct {
	mem *MemoryPipelineRepository
	db  PipelineDB
}

func NewPersistentPipelineRepository(mem *MemoryPipelineRepository, db PipelineDB) *PersistentPipelineRepository {
	return &PersistentPipelineRepository{mem: mem, db: db}
}

func (r *PersistentPipelineRepository) Create(ctx context.Context, p *upal.Pipeline) error {
	_ = r.mem.Create(ctx, p)
	return r.db.CreatePipeline(ctx, p)
}

func (r *PersistentPipelineRepository) Get(ctx context.Context, id string) (*upal.Pipeline, error) {
	if p, err := r.mem.Get(ctx, id); err == nil {
		return p, nil
	}
	p, err := r.db.GetPipeline(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, p)
	return p, nil
}

func (r *PersistentPipelineRepository) List(ctx context.Context) ([]*upal.Pipeline, error) {
	pipelines, err := r.db.ListPipelines(ctx)
	if err == nil {
		return pipelines, nil
	}
	slog.Warn("db list pipelines failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentPipelineRepository) Update(ctx context.Context, p *upal.Pipeline) error {
	_ = r.mem.Update(ctx, p)
	return r.db.UpdatePipeline(ctx, p)
}

func (r *PersistentPipelineRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	return r.db.DeletePipeline(ctx, id)
}

// PersistentPipelineRunRepository wraps MemoryPipelineRunRepository with a PostgreSQL backend.
type PersistentPipelineRunRepository struct {
	mem *MemoryPipelineRunRepository
	db  PipelineRunDB
}

func NewPersistentPipelineRunRepository(mem *MemoryPipelineRunRepository, db PipelineRunDB) *PersistentPipelineRunRepository {
	return &PersistentPipelineRunRepository{mem: mem, db: db}
}

func (r *PersistentPipelineRunRepository) Create(ctx context.Context, run *upal.PipelineRun) error {
	_ = r.mem.Create(ctx, run)
	return r.db.CreatePipelineRun(ctx, run)
}

func (r *PersistentPipelineRunRepository) Get(ctx context.Context, id string) (*upal.PipelineRun, error) {
	if run, err := r.mem.Get(ctx, id); err == nil {
		return run, nil
	}
	run, err := r.db.GetPipelineRun(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, run)
	return run, nil
}

func (r *PersistentPipelineRunRepository) ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.PipelineRun, error) {
	runs, err := r.db.ListPipelineRunsByPipeline(ctx, pipelineID)
	if err == nil {
		return runs, nil
	}
	slog.Warn("db list pipeline_runs failed, falling back to in-memory", "err", err)
	return r.mem.ListByPipeline(ctx, pipelineID)
}

func (r *PersistentPipelineRunRepository) Update(ctx context.Context, run *upal.PipelineRun) error {
	_ = r.mem.Update(ctx, run)
	return r.db.UpdatePipelineRun(ctx, run)
}
```

**Step 4: Run tests**

```bash
cd /home/dev/code/Upal && go test ./internal/repository/... -v -race -run "TestPersistentPipeline"
```
Expected: All PASS

**Step 5: Run full suite**

```bash
cd /home/dev/code/Upal && go test ./... -race
```
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/repository/pipeline_persistent.go internal/repository/pipeline_persistent_test.go
git commit -m "feat: add PersistentPipelineRepository and PersistentPipelineRunRepository"
```

---

### Task 4: Wire persistent repos in `cmd/upal/main.go`

**Files:**
- Modify: `cmd/upal/main.go:238-247`

**Step 1: No test needed** — this is wiring code; verified by integration compile + smoke test

**Step 2: Replace the pipeline section**

Find (lines 238-247):
```go
// Pipeline
pipelineRepo := repository.NewMemoryPipelineRepository()
pipelineRunRepo := repository.NewMemoryPipelineRunRepository()
pipelineSvc := services.NewPipelineService(pipelineRepo, pipelineRunRepo)
pipelineRunner := services.NewPipelineRunner(pipelineRunRepo)
pipelineRunner.RegisterExecutor(services.NewWorkflowStageExecutor(workflowSvc))
pipelineRunner.RegisterExecutor(services.NewApprovalStageExecutor())
pipelineRunner.RegisterExecutor(&services.TransformStageExecutor{})
srv.SetPipelineService(pipelineSvc)
srv.SetPipelineRunner(pipelineRunner)
```

Replace with:
```go
// Pipeline
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

pipelineSvc := services.NewPipelineService(pipelineRepo, pipelineRunRepo)
pipelineRunner := services.NewPipelineRunner(pipelineRunRepo)
pipelineRunner.RegisterExecutor(services.NewWorkflowStageExecutor(workflowSvc))
pipelineRunner.RegisterExecutor(services.NewApprovalStageExecutor())
pipelineRunner.RegisterExecutor(&services.TransformStageExecutor{})
srv.SetPipelineService(pipelineSvc)
srv.SetPipelineRunner(pipelineRunner)
```

**Step 3: Build to verify**

```bash
cd /home/dev/code/Upal && go build ./cmd/upal/...
```
Expected: no output (success)

**Step 4: Full test suite**

```bash
cd /home/dev/code/Upal && go test ./... -race
```
Expected: All PASS

**Step 5: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat: wire persistent pipeline repositories when DB is configured"
```
