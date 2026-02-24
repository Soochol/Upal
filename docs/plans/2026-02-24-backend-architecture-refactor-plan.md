# Backend Architecture Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Resolve 15 architecture issues across the Go backend: duplicate definitions, separation of concerns violations, and data flow gaps.

**Architecture:** Bottom-up 4-phase approach. Phase 1 (domain types) → Phase 2 (model resolution) → Phase 3 (separation of concerns) → Phase 4 (data flow). Each phase builds on the previous.

**Tech Stack:** Go 1.24, Chi router, Google ADK, PostgreSQL (optional)

**Design doc:** `docs/plans/2026-02-24-backend-architecture-refactor-design.md`

---

## Phase 1: Domain Type Consolidation

### Task 1: Add PipelineRunStatus and StageStatus typed enums

**Files:**
- Modify: `internal/upal/pipeline.go:116-134`
- Modify: `internal/services/pipeline_runner.go` (status literals)
- Modify: `internal/api/pipelines.go` (status comparisons)
- Test: `internal/services/pipeline_runner_test.go` (if exists)

**Step 1: Add typed enums to `internal/upal/pipeline.go`**

Add before `PipelineRun` struct:

```go
type PipelineRunStatus string

const (
	PipelineRunPending   PipelineRunStatus = "pending"
	PipelineRunRunning   PipelineRunStatus = "running"
	PipelineRunWaiting   PipelineRunStatus = "waiting"
	PipelineRunCompleted PipelineRunStatus = "completed"
	PipelineRunFailed    PipelineRunStatus = "failed"
	PipelineRunRejected  PipelineRunStatus = "rejected"
)

type StageStatus string

const (
	StageStatusPending   StageStatus = "pending"
	StageStatusRunning   StageStatus = "running"
	StageStatusWaiting   StageStatus = "waiting"
	StageStatusCompleted StageStatus = "completed"
	StageStatusFailed    StageStatus = "failed"
	StageStatusSkipped   StageStatus = "skipped"
)
```

Change `PipelineRun.Status` field type from `string` to `PipelineRunStatus`.
Change `StageResult.Status` field type from `string` to `StageStatus`.

**Step 2: Fix all compilation errors**

Replace string literals in:
- `internal/services/pipeline_runner.go`: all `"running"`, `"completed"`, `"failed"`, `"waiting"`, `"pending"`, `"skipped"` → `upal.PipelineRunRunning`, `upal.StageStatusCompleted`, etc.
- `internal/api/pipelines.go:115,117,158,163,215,221,224`: all status string comparisons/assignments → typed constants
- `internal/services/stage_workflow.go:68`: `Status: "completed"` → `Status: upal.StageStatusCompleted`
- All other stage executors that return `StageResult` with string status

**Step 3: Add WorkflowResultStatus and NodeRunStatus typed enums**

Add to `internal/upal/content.go` (after existing `ContentSessionStatus`):

```go
type WorkflowResultStatus string

const (
	WFResultPending WorkflowResultStatus = "pending"
	WFResultRunning WorkflowResultStatus = "running"
	WFResultSuccess WorkflowResultStatus = "success"
	WFResultFailed  WorkflowResultStatus = "failed"
)
```

Change `WorkflowResult.Status` from `string` to `WorkflowResultStatus`.

Add to `internal/upal/scheduler.go`:

```go
type NodeRunStatus string

const (
	NodeRunRunning   NodeRunStatus = "running"
	NodeRunCompleted NodeRunStatus = "completed"
	NodeRunError     NodeRunStatus = "error"
)
```

Change `NodeRunRecord.Status` from `string` to `NodeRunStatus`.

**Step 4: Fix compilation errors for WorkflowResultStatus and NodeRunStatus**

Update:
- `internal/services/content_collector.go`: all `"pending"`, `"running"`, `"success"`, `"failed"` for WorkflowResult → typed constants
- `internal/services/run/publisher.go:114,120`: `Status: "running"`, `Status: "completed"` → `upal.NodeRunRunning`, `upal.NodeRunCompleted`
- `internal/services/run/publisher.go:90`: `"status": "completed"` in donePayload (this is a map value, keep as string)

**Step 5: Run tests**

Run: `make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/upal/pipeline.go internal/upal/content.go internal/upal/scheduler.go \
  internal/services/pipeline_runner.go internal/api/pipelines.go \
  internal/services/stage_workflow.go internal/services/content_collector.go \
  internal/services/run/publisher.go
# Add any other files that needed changes
git commit -m "refactor: add typed status enums for PipelineRun, StageResult, WorkflowResult, NodeRunRecord"
```

---

### Task 2: Unify ErrNotFound across repositories

**Files:**
- Create: `internal/repository/errors.go`
- Modify: `internal/repository/workflow.go:12` (remove ErrNotFound)
- Modify: `internal/repository/connection_memory.go:30-35`
- Modify: `internal/repository/pipeline_memory.go:31-36,73-79`
- Modify: `internal/repository/content_memory.go:31-36,129-131,223-226`
- Modify: `internal/repository/run_memory.go` (if uses ErrNotFound)
- Modify: `internal/repository/trigger_memory.go`
- Modify: `internal/repository/schedule_memory.go`
- Modify: `internal/api/content.go` (strings.Contains → errors.Is)
- Test: existing tests + verify error matching

**Step 1: Create `internal/repository/errors.go`**

```go
package repository

import "errors"

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("not found")
```

**Step 2: Remove ErrNotFound from `internal/repository/workflow.go:12`**

Delete: `var ErrNotFound = errors.New("workflow not found")`

**Step 3: Update all memory repos to wrap ErrNotFound**

Change pattern from:
```go
return nil, fmt.Errorf("pipeline %q not found", id)
```
To:
```go
return nil, fmt.Errorf("pipeline %q: %w", id, ErrNotFound)
```

Apply to:
- `connection_memory.go:33` → `fmt.Errorf("connection %q: %w", id, ErrNotFound)`
- `pipeline_memory.go:34` → `fmt.Errorf("pipeline %q: %w", id, ErrNotFound)`
- `pipeline_memory.go:77` → `fmt.Errorf("pipeline run %q: %w", id, ErrNotFound)`
- `content_memory.go:34` → `fmt.Errorf("content session %q: %w", id, ErrNotFound)`
- `content_memory.go:131` → `fmt.Errorf("llm analysis for session %q: %w", sessionID, ErrNotFound)`
- `content_memory.go:226` → `fmt.Errorf("surge event %q: %w", id, ErrNotFound)`

Also check persistent repo implementations for same pattern.

**Step 4: Replace string error checks in `internal/api/content.go`**

Change all `strings.Contains(err.Error(), "not found")` to `errors.Is(err, repository.ErrNotFound)`:

Lines 113, 195, 264, 337, 427, 449, 471 — replace pattern.

Also replace:
- Line 428: `strings.Contains(err.Error(), "already archived")` → define `ErrAlreadyArchived` in services or upal and use `errors.Is`
- Line 450: `strings.Contains(err.Error(), "not archived")` → define `ErrNotArchived`
- Line 472: `strings.Contains(err.Error(), "must be archived")` → define `ErrMustBeArchived`

Add sentinel errors to `internal/upal/errors.go` (or wherever the service defines them):

```go
var (
	ErrAlreadyArchived = errors.New("session is already archived")
	ErrNotArchived     = errors.New("session is not archived")
	ErrMustBeArchived  = errors.New("session must be archived before deletion")
)
```

Update services to return these sentinel errors (wrap with `%w`).

**Step 5: Run tests**

Run: `make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/repository/errors.go internal/repository/workflow.go \
  internal/repository/connection_memory.go internal/repository/pipeline_memory.go \
  internal/repository/content_memory.go internal/api/content.go \
  # + any other modified files
git commit -m "refactor: unify ErrNotFound across repositories, replace string error checks with errors.Is"
```

---

### Task 3: Create shared domain DTOs (ModelSummary, ToolSummary)

**Files:**
- Create: `internal/upal/model_summary.go`
- Modify: `internal/generate/generate.go:19-31` (remove ModelOption, ToolEntry)
- Modify: `cmd/upal/main.go` (simplify conversion)

**Step 1: Create `internal/upal/model_summary.go`**

```go
package upal

// ModelSummary is a lightweight model descriptor for generation contexts.
// Avoids import cycles between api and generate packages.
type ModelSummary struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Tier     string `json:"tier,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// ToolSummary is a lightweight tool descriptor for generation contexts.
type ToolSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
```

**Step 2: Replace `ModelOption` and `ToolEntry` in `internal/generate/generate.go`**

Change:
```go
type Generator struct {
	// ...
	toolInfos []ToolEntry
	models    []ModelOption
}
```
To:
```go
type Generator struct {
	// ...
	toolInfos []upal.ToolSummary
	models    []upal.ModelSummary
}
```

Delete the `ModelOption` and `ToolEntry` type definitions (lines 19-31).

Update `New()` constructor signature and all internal references.

**Step 3: Update `cmd/upal/main.go` conversion code**

Change `generate.ModelOption` → `upal.ModelSummary`, `generate.ToolEntry` → `upal.ToolSummary` in the conversion loops (lines 310-323).

**Step 4: Run tests**

Run: `make test`
Expected: All tests pass

**Step 5: Commit**

```bash
git add internal/upal/model_summary.go internal/generate/generate.go cmd/upal/main.go
git commit -m "refactor: extract ModelSummary and ToolSummary to domain layer, remove generate.ModelOption/ToolEntry"
```

---

### Task 4: Unify SchedulerConfig with ConcurrencyLimits

**Files:**
- Modify: `internal/config/config.go:14-25`
- Modify: `cmd/upal/main.go:177-180`

**Step 1: Replace SchedulerConfig with ConcurrencyLimits in config**

In `internal/config/config.go`, change:
```go
Scheduler SchedulerConfig `yaml:"scheduler"`
```
To:
```go
Scheduler upal.ConcurrencyLimits `yaml:"scheduler"`
```

Delete `SchedulerConfig` type definition (lines 22-25). Add import for `upal` package.

**Step 2: Remove field-by-field copy in main.go**

In `cmd/upal/main.go`, change:
```go
concurrencyLimits := upal.ConcurrencyLimits{
    GlobalMax:   cfg.Scheduler.GlobalMax,
    PerWorkflow: cfg.Scheduler.PerWorkflow,
}
```
To:
```go
concurrencyLimits := cfg.Scheduler
```

**Step 3: Run tests**

Run: `make test`
Expected: All tests pass

**Step 4: Commit**

```bash
git add internal/config/config.go cmd/upal/main.go
git commit -m "refactor: unify SchedulerConfig with upal.ConcurrencyLimits"
```

---

### Task 5: EventRecord embeds WorkflowEvent + dead code cleanup

**Files:**
- Modify: `internal/services/runmanager.go:9-14`
- Modify: `internal/services/run/publisher.go:53-57,75-79`
- Modify: `internal/agents/registry.go:22-24` (delete ConnectionResolver)
- Modify: `internal/api/server.go:264-268` (delete local toolInfo)
- Modify: `internal/tools/registry.go:86-90` (add JSON tags to ToolInfo)

**Step 1: Embed WorkflowEvent in EventRecord**

In `internal/services/runmanager.go`, change:
```go
type EventRecord struct {
	Seq     int            `json:"seq"`
	Type    string         `json:"type"`
	NodeID  string         `json:"node_id,omitempty"`
	Payload map[string]any `json:"payload"`
}
```
To:
```go
type EventRecord struct {
	upal.WorkflowEvent
	Seq int `json:"seq"`
}
```

Note: `WorkflowEvent` fields don't have JSON tags. Add JSON tags to `WorkflowEvent` in `internal/upal/events.go`:
```go
type WorkflowEvent struct {
	Type    string         `json:"type"`
	NodeID  string         `json:"node_id,omitempty"`
	Payload map[string]any `json:"payload"`
}
```

**Step 2: Simplify EventRecord creation in publisher.go**

Change (lines 53-57 and 75-79):
```go
p.runManager.Append(runID, services.EventRecord{
    Type:    ev.Type,
    NodeID:  ev.NodeID,
    Payload: ev.Payload,
})
```
To:
```go
p.runManager.Append(runID, services.EventRecord{
    WorkflowEvent: ev,
})
```

**Step 3: Delete `ConnectionResolver` interface**

Remove lines 22-24 from `internal/agents/registry.go`:
```go
type ConnectionResolver interface {
	Resolve(ctx context.Context, id string) (*upal.Connection, error)
}
```

**Step 4: Replace local `toolInfo` with `tools.ToolInfo`**

Add JSON tags to `internal/tools/registry.go:86-90`:
```go
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Native      bool   `json:"native,omitempty"`
}
```

In `internal/api/server.go`, delete the local `toolInfo` struct (lines 264-268) and update `listAvailableTools` to use `tools.ToolInfo` directly. If the handler needs to exclude the `Native` field from JSON, use a local anonymous struct in the JSON encoding or just keep `Native` in the response (it's harmless).

**Step 5: Run tests**

Run: `make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/upal/events.go internal/services/runmanager.go \
  internal/services/run/publisher.go internal/agents/registry.go \
  internal/api/server.go internal/tools/registry.go
git commit -m "refactor: EventRecord embeds WorkflowEvent, delete ConnectionResolver, unify toolInfo"
```

---

## Phase 2: Model Resolution Unification

### Task 6: Inject LLMResolver into BuildDeps and agents

**Files:**
- Modify: `internal/agents/registry.go:28-33` (add LLMResolver to BuildDeps)
- Modify: `internal/agents/builders.go:41-61` (remove resolveLLM, use deps.LLMResolver)
- Modify: `internal/agents/builders.go` (all node builders that call resolveLLM)
- Modify: `internal/services/workflow.go:29-56` (pass LLMResolver in BuildDeps)
- Modify: `cmd/upal/main.go` (wire LLMResolver into BuildDeps)
- Test: `internal/agents/builders_test.go`

**Step 1: Add LLMResolver to BuildDeps**

In `internal/agents/registry.go`:
```go
type BuildDeps struct {
	LLMs             map[string]adkmodel.LLM
	LLMResolver      ports.LLMResolver  // new
	ToolReg          *tools.Registry
	OutputDir        string
	HTMLLayoutPrompt string
}
```

**Step 2: Update node builders to use LLMResolver**

Find all calls to `resolveLLM(modelID, deps.LLMs)` in builders and replace with:
```go
llm, modelName, err := deps.LLMResolver.Resolve(modelID)
if err != nil {
    return nil, fmt.Errorf("resolve model %q: %w", modelID, err)
}
```

The `namedLLM` wrapper may still be needed if the resolver doesn't handle model naming. Check if `MapResolver.Resolve()` returns the bare model name (it does — returns `modelName` from the split).

**Step 3: Delete `resolveLLM` function**

Remove `resolveLLM` (lines 41-61) and `namedLLM` struct if no longer needed.

Actually, check if `namedLLM` is still needed: it wraps an LLM to override `Name()`. The `MapResolver` doesn't do this. We may need to keep `namedLLM` or integrate naming into the resolver. Decision: keep `namedLLM` for now, create it from resolver output:

```go
llm, modelName, err := deps.LLMResolver.Resolve(modelID)
if err != nil {
    return nil, err
}
named := &namedLLM{LLM: llm, name: modelName}
```

**Step 4: Update WorkflowService to pass LLMResolver in BuildDeps**

In `internal/services/workflow.go`, add `LLMResolver` to the struct and constructor. Set `buildDeps.LLMResolver` when constructing `BuildDeps`.

**Step 5: Wire in main.go**

In `cmd/upal/main.go`, create `MapResolver` earlier and pass to `WorkflowService`:
```go
resolver := llmutil.NewMapResolver(llms, defaultLLM, defaultModelName)
workflowSvc := services.NewWorkflowService(repo, llms, sessionService, toolReg, nodeReg, outputDir, skillReg.GetPrompt("html-layout"), resolver)
```

**Step 6: Update tests**

Update `internal/agents/builders_test.go` to provide a `LLMResolver` in `BuildDeps`. Use `llmutil.NewMapResolver` or a test mock.

**Step 7: Run tests**

Run: `make test`
Expected: All tests pass

**Step 8: Commit**

```bash
git add internal/agents/registry.go internal/agents/builders.go \
  internal/services/workflow.go cmd/upal/main.go internal/agents/builders_test.go
git commit -m "refactor: inject LLMResolver into BuildDeps, remove duplicate resolveLLM"
```

---

### Task 7: Replace api.Server.resolveModel with LLMResolver

**Files:**
- Modify: `internal/api/server.go:23-47,281-298` (add llmResolver field, delete resolveModel)
- Modify: `internal/api/configure.go:70-78` (use llmResolver)
- Modify: `cmd/upal/main.go` (wire llmResolver to server)
- Test: `internal/api/edge_cases_test.go`, `internal/api/generate_test.go`

**Step 1: Add LLMResolver to Server struct**

In `internal/api/server.go`, add to Server struct:
```go
llmResolver ports.LLMResolver
```

Add a setter method or wire via constructor.

**Step 2: Replace resolveModel usage in configure.go**

Change (lines 70-78):
```go
resolved, ok := s.resolveModel(req.Model)
if !ok { ... }
llm := resolved.llm
model := resolved.model
```
To:
```go
llm, model, err := s.llmResolver.Resolve(req.Model)
if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
}
```

**Step 3: Delete resolveModel and resolvedModel from server.go**

Remove `resolvedModel` struct (lines 281-284) and `resolveModel` method (lines 287-298).

**Step 4: Wire in main.go**

```go
srv.SetLLMResolver(resolver)
```

**Step 5: Update tests**

Update test helpers to provide an LLMResolver (mock or MapResolver).

**Step 6: Run tests**

Run: `make test`
Expected: All tests pass

**Step 7: Commit**

```bash
git add internal/api/server.go internal/api/configure.go cmd/upal/main.go \
  internal/api/edge_cases_test.go internal/api/generate_test.go
git commit -m "refactor: replace api.resolveModel with injected LLMResolver"
```

---

### Task 8: Remove inline model validation from WorkflowService.Validate

**Files:**
- Modify: `internal/services/workflow.go:64-86`

**Step 1: Use LLMResolver in Validate**

Change the inline `strings.SplitN` validation (lines 77-83) to use `LLMResolver`:

```go
if _, _, err := s.llmResolver.Resolve(modelID); err != nil {
    return fmt.Errorf("node %q: %w", n.ID, err)
}
```

This replaces the manual split + provider lookup.

**Step 2: Run tests**

Run: `make test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/services/workflow.go
git commit -m "refactor: use LLMResolver in WorkflowService.Validate"
```

---

### Task 9: Move model catalog from api to domain/model layer

**Files:**
- Create: `internal/model/catalog.go`
- Create: `internal/upal/models.go` (ModelInfo, ModelCategory, ModelTier, OptionSchema)
- Modify: `internal/api/models.go` (remove type defs and data, keep handlers)
- Modify: `cmd/upal/main.go` (import changes)

**Step 1: Create `internal/upal/models.go`**

Move from `internal/api/models.go` to `internal/upal/models.go`:
- `ModelCategory` type + constants (lines 14-19)
- `ModelTier` type + constants (lines 41-47)
- `OptionSchema` struct (lines 22-32)
- `OptionChoice` struct (lines 35-38)
- `ModelInfo` struct (lines 49-58)

**Step 2: Create `internal/model/catalog.go`**

Move the model list data (knownModels map, `allModels()` or equivalent) from `api/models.go` to `model/catalog.go`. Function returns `[]upal.ModelInfo`.

```go
package model

func AllStaticModels() []upal.ModelInfo { ... }
```

Move Ollama discovery logic to `model/ollama_discovery.go` (or keep in catalog.go), returning `[]upal.ModelInfo`.

**Step 3: Update api/models.go**

- Remove all type definitions (now in `upal/models.go`)
- Remove model list data (now in `model/catalog.go`)
- Import from `upal` and `model` packages
- Handler functions call `model.AllStaticModels()` etc.

**Step 4: Update imports across codebase**

Replace `api.ModelInfo` → `upal.ModelInfo`, `api.ModelCategory` → `upal.ModelCategory`, etc. in any files that reference these types.

Update `internal/generate/generate.go` to reference `upal.ModelInfo` where needed (for constructing the model list tool).

**Step 5: Simplify main.go model conversion**

With `upal.ModelSummary` (from Task 3) and `upal.ModelInfo` (this task), the conversion in `main.go` simplifies — can add a method `ModelInfo.Summary() ModelSummary` if desired.

**Step 6: Run tests**

Run: `make test`
Expected: All tests pass

**Step 7: Commit**

```bash
git add internal/upal/models.go internal/model/catalog.go internal/api/models.go \
  cmd/upal/main.go internal/generate/generate.go
git commit -m "refactor: move model catalog from api to domain/model layer"
```

---

## Phase 3: Separation of Concerns

### Task 10: Extract configureNode to generate package

**Files:**
- Create: `internal/generate/configure.go`
- Create: `internal/generate/configure_test.go`
- Modify: `internal/api/configure.go` (thin handler)

**Step 1: Write test for Generator.ConfigureNode**

In `internal/generate/configure_test.go`:

```go
func TestConfigureNode_ReturnsConfig(t *testing.T) {
	// Setup mock LLM that returns a JSON config response
	// Call generator.ConfigureNode(ctx, wf, nodeID, model)
	// Assert result contains expected config fields
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/generate/... -v -race -run TestConfigureNode`
Expected: FAIL (method doesn't exist)

**Step 3: Extract business logic to `internal/generate/configure.go`**

Move the LLM orchestration from `api/configure.go:50-190` into:

```go
type ConfigureResult struct {
	Config map[string]any `json:"config"`
}

func (g *Generator) ConfigureNode(ctx context.Context, wf *upal.WorkflowDefinition, nodeID string, modelID string, resolver ports.LLMResolver, skills skills.Provider) (*ConfigureResult, error) {
	// ... all LLM orchestration logic from the handler ...
}
```

The generator needs access to `skills.Provider` and `LLMResolver` — these can be passed as parameters or stored on the Generator struct. Since Generator already has `skills`, use that. For LLMResolver, add as parameter or struct field.

**Step 4: Thin out the API handler**

`internal/api/configure.go` becomes:

```go
func (s *Server) configureNode(w http.ResponseWriter, r *http.Request) {
	var req ConfigureNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := s.generator.ConfigureNode(r.Context(), req.Workflow, req.NodeID, req.Model, ...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}
```

**Step 5: Run tests**

Run: `make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/generate/configure.go internal/generate/configure_test.go internal/api/configure.go
git commit -m "refactor: extract configureNode business logic to generate.ConfigureNode"
```

---

### Task 11: Extract suggestWorkflowName to generate package

**Files:**
- Create: `internal/generate/name.go` (or add to existing)
- Modify: `internal/api/name.go` (thin handler)

**Step 1: Extract to Generator.SuggestName**

Move LLM call logic from `api/name.go:28-101` to:

```go
// internal/generate/name.go
func (g *Generator) SuggestName(ctx context.Context, wf *upal.WorkflowDefinition) (string, error) {
	// ... LLM call + response parsing ...
}
```

**Step 2: Thin out handler**

```go
func (s *Server) suggestWorkflowName(w http.ResponseWriter, r *http.Request) {
	var req suggestNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
	name, err := s.generator.SuggestName(r.Context(), req.Workflow)
	if err != nil { ... }
	json.NewEncoder(w).Encode(map[string]string{"name": name})
}
```

**Step 3: Run tests & commit**

Run: `make test`

```bash
git add internal/generate/name.go internal/api/name.go
git commit -m "refactor: extract suggestWorkflowName to generate.SuggestName"
```

---

### Task 12: Extract publishContentSession to service

**Files:**
- Modify: `internal/services/content_session_service.go`
- Modify: `internal/api/content.go:222-280`

**Step 1: Add PublishSession to ContentSessionService**

```go
func (s *ContentSessionService) PublishSession(ctx context.Context, sessionID string, selectedResults []string, storage storage.Storage) error {
	// ... multi-step publish orchestration from handler ...
}
```

**Step 2: Thin out handler**

**Step 3: Run tests & commit**

```bash
git commit -m "refactor: extract publishContentSession to ContentSessionService.PublishSession"
```

---

### Task 13: Extract rejectPipelineRun to service

**Files:**
- Modify: `internal/services/pipeline_service.go` (or create)
- Modify: `internal/api/pipelines.go:202-235`

**Step 1: Add RejectRun to PipelineService**

```go
func (s *PipelineService) RejectRun(ctx context.Context, pipelineID, runID, reason string) error {
	// ... domain state mutation from handler ...
}
```

Use typed status enums from Task 1.

**Step 2: Thin out handler & commit**

```bash
git commit -m "refactor: extract rejectPipelineRun to PipelineService.RejectRun"
```

---

### Task 14: Extract remaining inline prompts to skills

**Files:**
- Create: `internal/skills/prompts/thumbnail-user.md`
- Create: `internal/skills/_frameworks/output-extract.md`
- Modify: `internal/generate/thumbnail.go:56-102,143-165`
- Modify: `internal/agents/output_extract.go:45-59`

**Step 1: Create `internal/skills/prompts/thumbnail-user.md`**

```markdown
## Workflow Thumbnail
Create a thumbnail for an AI workflow named "{{workflow_name}}".{{description}}
Nodes: {{nodes}}.{{agent_context}}

Design a bold, conceptual visual that captures the specific semantic purpose and objective of this workflow. DO NOT just draw a generic graph of nodes and edges — illustrate what it actually does.

## Pipeline Thumbnail
Create a thumbnail for a pipeline named "{{pipeline_name}}".
Stages: {{stages}}.{{description}}
Design a visual that captures this pipeline's purpose.
```

**Step 2: Create `internal/skills/_frameworks/output-extract.md`**

```markdown
## JSON Mode
Respond ONLY with valid JSON in this exact format:
{"{{key}}": <your output here>}
Do not include any other text outside the JSON object.

## Tagged Mode
Wrap your final output in <{{tag}}> tags:
<{{tag}}>your output here</{{tag}}>
Do not include any other text outside the tags.
```

**Step 3: Update Go code to load from skills**

In `thumbnail.go`, replace `fmt.Sprintf(...)` with:
```go
tmpl := g.skills.GetPrompt("thumbnail-user")
// substitute placeholders
```

In `output_extract.go`, load from skills and substitute.

**Step 4: Run tests & commit**

```bash
git commit -m "refactor: extract remaining inline prompts to skill files"
```

---

### Task 15: Extract hardcoded config values

**Files:**
- Modify: `internal/config/config.go` (add RunsConfig, GeneratorConfig fields)
- Modify: `cmd/upal/main.go` (pass config values)
- Modify: `internal/services/runmanager.go:150` (use config)
- Modify: `internal/api/generate.go:74,118,149,234` (use config)
- Modify: `internal/api/upload.go:18` (use config)

**Step 1: Add config sections**

In `internal/config/config.go`:

```go
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Database  DatabaseConfig            `yaml:"database"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Scheduler upal.ConcurrencyLimits   `yaml:"scheduler"`
	Runs      RunsConfig               `yaml:"runs"`
	Generator GeneratorConfig          `yaml:"generator"`
}

type ServerConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	UploadMaxSize  int64  `yaml:"upload_max_size"`
}

type RunsConfig struct {
	TTL        duration `yaml:"ttl"`
	GCInterval duration `yaml:"gc_interval"`
}

type GeneratorConfig struct {
	ThumbnailTimeout duration `yaml:"thumbnail_timeout"`
	AnalysisTimeout  duration `yaml:"analysis_timeout"`
}
```

Note: `time.Duration` doesn't unmarshal from YAML strings by default. Either use a custom `duration` type or parse in `defaults()`.

**Step 2: Set defaults**

```go
func defaults() *Config {
	return &Config{
		Server: ServerConfig{Host: "0.0.0.0", Port: 8080, UploadMaxSize: 50 << 20},
		Runs: RunsConfig{TTL: duration(5 * time.Minute), GCInterval: duration(time.Minute)},
		Generator: GeneratorConfig{
			ThumbnailTimeout: duration(60 * time.Second),
			AnalysisTimeout:  duration(5 * time.Minute),
		},
	}
}
```

**Step 3: Thread config values to consumers**

- `RunManager` constructor: pass `cfg.Runs.TTL` and `cfg.Runs.GCInterval`
- API handlers: extract thumbnail timeout from `cfg.Generator.ThumbnailTimeout`
- `api/upload.go`: use `cfg.Server.UploadMaxSize`

**Step 4: Extract `60*time.Second` literal repeated 4 times**

At minimum, define a constant in `api/generate.go`:
```go
const thumbnailTimeout = 60 * time.Second
```
Then use `cfg.Generator.ThumbnailTimeout` from config (or the constant as default).

**Step 5: Run tests & commit**

```bash
git commit -m "refactor: extract hardcoded config values to config.yaml sections"
```

---

## Phase 4: Data Flow Restoration

### Task 16: Wire ExecutionRegistry into RunPublisher

**Files:**
- Modify: `internal/services/run/publisher.go:16-20` (add executionReg)
- Modify: `internal/services/run/publisher.go:37-100` (Register/Unregister in Launch)
- Modify: `cmd/upal/main.go` (pass executionReg to RunPublisher)

**Step 1: Add ExecutionRegistry to RunPublisher**

```go
type RunPublisher struct {
	workflowExec  ports.WorkflowExecutor
	runManager    *services.RunManager
	runHistorySvc ports.RunHistoryPort
	executionReg  *services.ExecutionRegistry
}
```

Update constructor to accept `*services.ExecutionRegistry`.

**Step 2: Register/Unregister in Launch**

At the start of `Launch()`:
```go
if p.executionReg != nil {
	p.executionReg.Register(runID)
	defer p.executionReg.Unregister(runID)
}
```

**Step 3: Wire in main.go**

Pass `execReg` to `NewRunPublisher(...)`.

**Step 4: Run tests & commit**

```bash
git commit -m "feat: wire ExecutionRegistry into RunPublisher for pause/resume support"
```

---

### Task 17: Record token usage in RunPublisher

**Files:**
- Modify: `internal/services/run/publisher.go:103-125` (trackNodeRun)
- Modify: `internal/services/run/publisher.go:37-100` (aggregate in Launch)
- Modify: `internal/upal/ports/runhistory.go` (CompleteRun signature if needed)

**Step 1: Extract token usage in trackNodeRun**

In the `EventNodeCompleted` case of `trackNodeRun`:

```go
case upal.EventNodeCompleted:
	rec.Status = upal.NodeRunCompleted
	now := time.Now()
	rec.CompletedAt = &now
	if tokens, ok := ev.Payload["tokens"].(map[string]any); ok {
		rec.Usage = &upal.TokenUsage{
			PromptTokens:     int32(toInt(tokens["prompt_token_count"])),
			CompletionTokens: int32(toInt(tokens["candidates_token_count"])),
			TotalTokens:      int32(toInt(tokens["total_token_count"])),
		}
	}
```

Add a helper:
```go
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	default:
		return 0
	}
}
```

**Step 2: Aggregate in Launch and pass to CompleteRun**

After draining events, sum up all node token usage:
```go
var totalUsage upal.TokenUsage
for _, nr := range nodeRecords {
	if nr.Usage != nil {
		totalUsage.PromptTokens += nr.Usage.PromptTokens
		totalUsage.CompletionTokens += nr.Usage.CompletionTokens
		totalUsage.TotalTokens += nr.Usage.TotalTokens
	}
}
```

If `CompleteRun` doesn't accept usage, add it or use a separate `UpdateRunUsage` method.

**Step 3: Run tests & commit**

```bash
git commit -m "feat: record token usage from ADK events into RunRecord and NodeRunRecord"
```

---

### Task 18: Pass webhook inputs to PipelineRunner.Start

**Files:**
- Modify: `internal/upal/ports/scheduler.go` (PipelineRunner.Start signature)
- Modify: `internal/services/pipeline_runner.go:40-55` (Start signature + executeFrom)
- Modify: `internal/api/webhooks.go:74-81` (pass inputs)
- Modify: `internal/api/pipelines.go` (update Start calls)
- Modify: all callers of PipelineRunner.Start

**Step 1: Update PipelineRunner port interface**

In `internal/upal/ports/scheduler.go`:
```go
type PipelineRunner interface {
	Start(ctx context.Context, pipeline *upal.Pipeline, inputs map[string]any) (*upal.PipelineRun, error)
	Resume(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun) error
}
```

**Step 2: Update implementation**

In `internal/services/pipeline_runner.go`:
```go
func (r *PipelineRunner) Start(ctx context.Context, pipeline *upal.Pipeline, inputs map[string]any) (*upal.PipelineRun, error) {
	// ... create run ...
	if err := r.executeFrom(ctx, pipeline, run, 0, inputs); err != nil {
		return run, err
	}
	return run, nil
}

func (r *PipelineRunner) executeFrom(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun, startIdx int, inputs map[string]any) error {
	var prevResult *upal.StageResult
	if inputs != nil {
		prevResult = &upal.StageResult{Output: inputs, Status: upal.StageStatusCompleted}
	}
	// ... rest of execution ...
}
```

**Step 3: Update all callers**

- `api/webhooks.go:75`: `s.pipelineRunner.Start(ctx, pipeline, inputs)` (was `s.pipelineRunner.Start(ctx, pipeline)`)
- `api/pipelines.go`: `s.pipelineRunner.Start(ctx, pipeline, nil)`
- Scheduler: `pipelineRunner.Start(ctx, pipeline, nil)`

**Step 4: Run tests & commit**

```bash
git commit -m "feat: pass webhook inputs through to PipelineRunner.Start"
```

---

### Task 19: Expand StageExecutor interface with pipeline context

**Files:**
- Modify: `internal/services/pipeline_runner.go:18-21` (StageExecutor interface)
- Modify: All stage executor implementations
- Modify: `internal/services/pipeline_runner.go:107` (executor.Execute call)

**Step 1: Expand StageExecutor interface**

```go
type StageExecutor interface {
	Type() string
	Execute(ctx context.Context, pipeline *upal.Pipeline, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error)
}
```

**Step 2: Update all executor implementations**

Add `pipeline *upal.Pipeline` parameter to `Execute` in:
- `WorkflowStageExecutor`
- `ApprovalStageExecutor`
- `NotificationStageExecutor`
- `TransformStageExecutor`
- `CollectStageExecutor`
- `PassthroughStageExecutor`

Most executors will just ignore the `pipeline` parameter for now. `WorkflowStageExecutor` can use `pipeline.Model` for model selection.

**Step 3: Update executeFrom call site**

```go
result, err := executor.Execute(ctx, pipeline, stage, prevResult)
```

**Step 4: Run tests & commit**

```bash
git commit -m "refactor: expand StageExecutor.Execute with pipeline context parameter"
```

---

### Task 20: Pipeline workflow stages use RunPublisher

**Files:**
- Modify: `internal/services/stage_workflow.go` (inject RunPublisher)
- Modify: `internal/services/pipeline_runner.go` (wire RunPublisher)
- Modify: `cmd/upal/main.go` (pass RunPublisher to WorkflowStageExecutor)

**Step 1: Inject RunPublisher into WorkflowStageExecutor**

```go
type WorkflowStageExecutor struct {
	workflowSvc ports.WorkflowExecutor
	publisher   *run.RunPublisher
}

func NewWorkflowStageExecutor(workflowSvc ports.WorkflowExecutor, publisher *run.RunPublisher) *WorkflowStageExecutor {
	return &WorkflowStageExecutor{workflowSvc: workflowSvc, publisher: publisher}
}
```

**Step 2: Use RunPublisher.Launch in Execute**

Replace direct `workflowSvc.Run()` + event draining with `publisher.Launch()`:

```go
func (e *WorkflowStageExecutor) Execute(ctx context.Context, pipeline *upal.Pipeline, stage upal.Stage, prev *upal.StageResult) (*upal.StageResult, error) {
	// ... resolve workflow, map inputs ...
	if e.publisher != nil {
		runID := upal.GenerateID("run")
		e.publisher.Launch(ctx, runID, wf, inputs)
		// Need to wait for completion and get result
		// May need a synchronous variant of Launch, or a channel-based approach
	} else {
		// fallback to direct execution
		eventCh, resultCh, err := e.workflowSvc.Run(ctx, wf, inputs)
		for range eventCh {}
		result := <-resultCh
		return &upal.StageResult{...}, nil
	}
}
```

Note: `RunPublisher.Launch` runs in the background. For stage execution, we need a synchronous version that blocks until completion and returns the result. Consider adding `RunPublisher.LaunchSync(ctx, runID, wf, inputs) (*upal.RunResult, error)` that returns the result after the run completes.

**Step 3: Wire in main.go**

```go
pipelineRunner.RegisterExecutor(services.NewWorkflowStageExecutor(workflowSvc, runPublisher))
```

**Step 4: Run tests & commit**

```bash
git commit -m "feat: pipeline workflow stages tracked via RunPublisher with run history"
```

---

### Task 21: Persist WorkflowResults

**Files:**
- Create: `internal/repository/workflow_result_memory.go`
- Create: `internal/repository/workflow_result_persistent.go` (if DB available)
- Add to: `internal/upal/ports/content.go` or create `ports/workflow_result.go`
- Modify: `internal/services/content_session_service.go:22-23` (remove in-memory map)
- Modify: `cmd/upal/main.go` (wire repository)

**Step 1: Define repository interface**

```go
// internal/upal/ports/content.go (or new file)
type WorkflowResultRepository interface {
	Save(ctx context.Context, sessionID string, results []upal.WorkflowResult) error
	GetBySession(ctx context.Context, sessionID string) ([]upal.WorkflowResult, error)
	DeleteBySession(ctx context.Context, sessionID string) error
}
```

**Step 2: Implement memory repository**

```go
// internal/repository/workflow_result_memory.go
type MemoryWorkflowResultRepository struct {
	mu      sync.RWMutex
	results map[string][]upal.WorkflowResult
}
```

**Step 3: Implement persistent repository**

If using PostgreSQL, add SQL schema:
```sql
CREATE TABLE IF NOT EXISTS workflow_results (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    run_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    output_url TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_workflow_results_session ON workflow_results(session_id);
```

**Step 4: Replace in-memory map in ContentSessionService**

Remove:
```go
workflowResults map[string][]upal.WorkflowResult
```

Replace `SetWorkflowResults` and `GetWorkflowResults` with calls to the repository.

**Step 5: Wire in main.go**

Create the repository, inject into `ContentSessionService`.

**Step 6: Run tests & commit**

```bash
git commit -m "feat: persist WorkflowResults via repository, survive server restarts"
```

---

### Task 22: Fill missing fields (PipelineName, PendingSessionCount)

**Files:**
- Modify: `internal/services/content_session_service.go:368-411,415-459`
- Modify: `internal/api/pipelines.go` (or service)

**Step 1: Inject PipelineRepository into ContentSessionService**

Add `pipelineRepo` field. In `GetSessionDetail()`:
```go
if sess.PipelineID != "" {
    if p, err := s.pipelineRepo.Get(ctx, sess.PipelineID); err == nil {
        detail.PipelineName = p.Name
    }
}
```

**Step 2: Add CountByStatus method**

```go
func (s *ContentSessionService) CountByStatus(ctx context.Context, pipelineID string, status upal.ContentSessionStatus) (int, error) {
	// count sessions matching pipeline + status
}
```

Use in pipeline list/get to set `PendingSessionCount`.

**Step 3: Run tests & commit**

```bash
git commit -m "feat: populate PipelineName and PendingSessionCount in API responses"
```

---

### Task 23: Prepare userID context propagation

**Files:**
- Create: `internal/upal/context.go`
- Modify: `internal/services/workflow.go:110` (use context)

**Step 1: Create context utilities**

```go
// internal/upal/context.go
package upal

import "context"

type contextKey string

const userIDKey contextKey = "userID"

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok && v != "" {
		return v
	}
	return "default"
}
```

**Step 2: Use in WorkflowService.Run**

Replace:
```go
userID := "default"
```
With:
```go
userID := upal.UserIDFromContext(ctx)
```

**Step 3: Run tests & commit**

```bash
git commit -m "feat: add userID context propagation, prepare for multi-user support"
```

---

### Task 24: Integrate scheduler with ContentCollector

**Files:**
- Modify: scheduler service (add ContentCollector dependency)
- Modify: `cmd/upal/main.go` (wire)

**Step 1: Add ContentCollector to scheduler**

Add a `SetContentCollector` method to the scheduler service.

**Step 2: Dispatch logic**

In the scheduled pipeline dispatch path:
```go
if len(pipeline.Sources) > 0 && s.contentCollector != nil {
    // Content pipeline: create session, collect, analyze
    sess, _ := s.contentCollector.CreateSession(ctx, pipeline)
    go s.contentCollector.CollectAndAnalyze(ctx, pipeline, sess, false, 0)
} else {
    // Generic pipeline: stage execution
    go s.pipelineRunner.Start(ctx, pipeline, nil)
}
```

**Step 3: Wire in main.go**

```go
if collector != nil {
    schedulerSvc.SetContentCollector(collector)
}
```

**Step 4: Run tests & commit**

```bash
git commit -m "feat: scheduler dispatches content pipelines through ContentCollector lifecycle"
```

---

### Task 25: Final integration test and cleanup

**Step 1: Run full test suite**

```bash
make test
```

**Step 2: Run frontend type check**

```bash
make test-frontend
```

If any API response types changed (e.g. status fields now have different serialization), update frontend types.

**Step 3: Build**

```bash
make build
```

**Step 4: Manual smoke test**

```bash
make dev-backend
# In another terminal:
make dev-frontend
# Verify: create workflow, run, check run history
# Verify: create pipeline, trigger webhook, check stage results
# Verify: content pipeline collect → analyze → approve → produce
```

**Step 5: Final commit**

```bash
git commit -m "test: verify architecture refactor integration"
```

---

## Task Dependency Graph

```
Task 1 (typed enums) ─────────────────────────────────────────────┐
Task 2 (ErrNotFound) ─────────────────────────────────────────────┤
Task 3 (shared DTOs) ──┬──────────────────────────────────────────┤
Task 4 (config unify) ─┤                                          │
Task 5 (embed+cleanup) ┘                                          │
                        │                                          │
                Task 6 (LLMResolver → BuildDeps) ──┐               │
                Task 7 (LLMResolver → api.Server) ─┤               │
                Task 8 (LLMResolver → Validate) ───┤               │
                Task 9 (model catalog move) ────────┘              │
                                                    │              │
                        Task 10 (extract configureNode) ──┐        │
                        Task 11 (extract suggestName) ────┤        │
                        Task 12 (extract publishSession) ─┤        │
                        Task 13 (extract rejectRun) ──────┤        │
                        Task 14 (extract prompts) ────────┤        │
                        Task 15 (extract config values) ──┘        │
                                                           │       │
                                Task 16 (ExecutionRegistry) ──┐    │
                                Task 17 (token usage) ────────┤    │
                                Task 18 (webhook inputs) ─────┤    │
                                Task 19 (StageExecutor sig) ──┤    │
                                Task 20 (workflow→RunPub) ────┤    │
                                Task 21 (persist WFResults) ──┤    │
                                Task 22 (fill fields) ────────┤    │
                                Task 23 (userID context) ─────┤    │
                                Task 24 (scheduler+collector)─┤    │
                                Task 25 (integration test) ───┘────┘
```

**Parallelizable within phases:**
- Phase 1: Tasks 1-5 are independent, can run in parallel
- Phase 2: Tasks 6-8 sequential (6→7→8), Task 9 independent
- Phase 3: Tasks 10-14 are mostly independent, Task 15 independent
- Phase 4: Tasks 16-17 sequential, 18-19 sequential, 20 depends on 19, 21-24 independent
