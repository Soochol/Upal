# Pipeline Two-Phase Generation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Pipeline generation produces high-quality workflows by delegating to the same `Generate()` function used for direct workflow creation, instead of having the pipeline LLM generate workflows inline.

**Architecture:** Two-phase generation — Phase 1: pipeline LLM generates skeleton + `workflow_specs` (name + rich description per new workflow needed); Phase 2: each new spec goes through `Generate()` in parallel (identical to direct workflow generation). API handler saves generated workflows to repo before returning bundle.

**Tech Stack:** Go 1.23, `sync` package for parallel execution, existing `Generator.Generate()` and `Generator.buildModelPrompt()` in `internal/generate/`.

---

### Task 1: Add `WorkflowSpec` type and update `PipelineBundle`

**Files:**
- Modify: `internal/generate/pipeline.go:13-17`

**Step 1: Add the new type and modify PipelineBundle**

In `internal/generate/pipeline.go`, add `WorkflowSpec` and update `PipelineBundle`:

```go
// WorkflowSpec describes a workflow the pipeline LLM wants created.
// Phase 2 calls Generate(Description) and enforces Name on the result.
type WorkflowSpec struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

// PipelineBundle is what the LLM returns — a pipeline + workflow specs to generate.
type PipelineBundle struct {
    Pipeline      upal.Pipeline             `json:"pipeline"`
    WorkflowSpecs []WorkflowSpec            `json:"workflow_specs"`
    Workflows     []upal.WorkflowDefinition `json:"-"` // populated by Phase 2, not from LLM JSON
}
```

**Step 2: Run existing tests to confirm no regressions**

```bash
go test ./internal/generate/... -v -race
```
Expected: all existing tests pass.

**Step 3: Commit**

```bash
git add internal/generate/pipeline.go
git commit -m "feat(generate): add WorkflowSpec type and update PipelineBundle"
```

---

### Task 2: Inject models and tools into pipeline system prompt

**Files:**
- Modify: `internal/generate/pipeline.go:354-369` (`buildPipelineSysPrompt`)

**Step 1: Write the failing test**

In `internal/generate/pipeline_test.go`, add (or create the file if missing):

```go
func TestBuildPipelineSysPromptInjectsModelsAndTools(t *testing.T) {
    g := &Generator{
        models: []ModelOption{
            {ID: "anthropic/claude-sonnet-4-6", Category: "text", Tier: "mid", Hint: "general purpose"},
        },
        toolInfos: []ToolEntry{
            {Name: "web_search", Description: "Search the web"},
        },
        defaultModelID: "anthropic/claude-sonnet-4-6",
    }
    prompt := g.buildPipelineSysPrompt("BASE", nil, nil)

    if !strings.Contains(prompt, "claude-sonnet-4-6") {
        t.Error("expected model ID in pipeline system prompt")
    }
    if !strings.Contains(prompt, "web_search") {
        t.Error("expected tool name in pipeline system prompt")
    }
}
```

**Step 2: Run test to confirm it fails**

```bash
go test ./internal/generate/... -v -race -run TestBuildPipelineSysPromptInjectsModelsAndTools
```
Expected: FAIL — model and tool not in prompt yet.

**Step 3: Modify `buildPipelineSysPrompt`**

Add model and tool injection after the existing workflow/pipeline list injection, before the final reinforcement line.

Current end of `buildPipelineSysPrompt` (around line 361-368):
```go
    if len(existingPipelines) > 0 {
        sysPrompt += "\n\nExisting pipelines ..."
        sysPrompt += formatPipelineList(existingPipelines)
    }

    // Final reinforcement — must be last so it benefits from recency bias.
    sysPrompt += "\n\nIMPORTANT: ..."
    return sysPrompt
```

Replace that section with:
```go
    if len(existingPipelines) > 0 {
        sysPrompt += "\n\nExisting pipelines (for reference — avoid duplicating; understand patterns from stage summaries):\n"
        sysPrompt += formatPipelineList(existingPipelines)
    }

    // Inject available models — used when writing workflow_specs descriptions.
    if len(g.models) > 0 {
        sysPrompt += g.buildModelPrompt()
    }

    // Inject available tools — used when writing workflow_specs descriptions.
    if len(g.toolInfos) > 0 {
        sysPrompt += "\n\nAvailable tools (for use when describing workflow_specs — mention by name in descriptions):\n"
        for _, t := range g.toolInfos {
            sysPrompt += fmt.Sprintf("- %q — %s\n", t.Name, t.Description)
        }
        sysPrompt += "ONLY reference tools from this list in workflow_specs descriptions.\n"
    }

    // Final reinforcement — must be last so it benefits from recency bias.
    sysPrompt += "\n\nIMPORTANT: Your entire response must be ONLY the raw JSON object. No markdown fences, no explanation, no commentary before or after the JSON."
    return sysPrompt
```

**Step 4: Run test to confirm it passes**

```bash
go test ./internal/generate/... -v -race -run TestBuildPipelineSysPromptInjectsModelsAndTools
```
Expected: PASS.

**Step 5: Run all generate tests**

```bash
go test ./internal/generate/... -v -race
```
Expected: all pass.

**Step 6: Commit**

```bash
git add internal/generate/pipeline.go internal/generate/pipeline_test.go
git commit -m "feat(generate): inject models and tools into pipeline system prompt"
```

---

### Task 3: Modify `pipeline-create.md` prompt

**Files:**
- Modify: `internal/skills/prompts/pipeline-create.md`

**Step 1: Rewrite the prompt**

Replace the entire file content with the updated version that:
- Allows `workflow_specs` for new workflows
- Removes the prohibition on workflow definitions
- Updates workflow stage guidance to allow new names

```markdown
You are a pipeline generator for the Upal platform. Given a user's natural language description, produce a valid pipeline JSON.

For any `workflow` stage that references a workflow not in the "Available workflows" list, add a `workflow_specs` entry describing what it should do. The system will generate that workflow automatically.

---

## Before generating

Use `get_skill(skill_name)` to load the detailed configuration guide for each stage type you will use:

| Stage type | skill_name |
|-----------|-----------|
| collect | `"stage-collect"` |
| notification | `"stage-notification"` |
| approval | `"stage-approval"` |
| schedule | `"stage-schedule"` |
| trigger | `"stage-trigger"` |
| transform | `"stage-transform"` |

Load only the skill guides for stage types you will actually use. After loading, generate the JSON.

---

## Output Schema

```json
{
  "pipeline": {
    "name": "english-slug",
    "description": "한국어 설명",
    "stages": [ ...Stage[] ]
  },
  "workflow_specs": [
    {
      "name": "exact-slug-matching-stage-workflow_name",
      "description": "Rich description: what inputs it takes, what tools to use, what model tier, what output format"
    }
  ]
}
```

`workflow_specs` rules:
- Include an entry for every `workflow` stage whose `workflow_name` is NOT in the "Available workflows" list.
- If all workflow stages reference existing workflows, omit `workflow_specs` entirely (or set `[]`).
- `name` MUST exactly match the `workflow_name` used in the stage config.
- `description` should be detailed: mention input variables, tools from the Available tools list, output format, and model tier.

### Stage base fields (required on every stage)

```json
{
  "id":          "stage-N",
  "name":        "한국어 이름",
  "description": "한국어 설명",
  "type":        "...",
  "config":      { ... },
  "depends_on":  ["stage-N"]
}
```

`depends_on` establishes execution order. Every stage (except stage-1) MUST declare which stage(s) must complete before it runs.

---

## Stage Types

Only these seven types exist: `workflow`, `collect`, `notification`, `approval`, `schedule`, `trigger`, `transform`.

### "workflow" — run a workflow by name
```json
"config": {
  "workflow_name": "exact-name",
  "input_mapping": { "workflow_input_var": "static value or {{field}}" }
}
```
- `workflow_name`: MUST exactly match a name from the "Available workflows" list, OR a name you define in `workflow_specs`.
- `input_mapping`: optional. Maps the workflow's input variable names to values or `{{field}}` references.

For all other stage types, call `get_skill("stage-TYPE")` to load the full configuration guide.

---

## Output Fields — for use in `input_mapping`

| Stage type | Available `{{field}}` references |
|------------|----------------------------------|
| collect | `{{text}}`, `{{sources}}` |
| workflow | `{{output}}` |
| transform | `{{output}}` |
| notification | `{{sent}}`, `{{channel}}` |
| schedule / trigger / approval | (no meaningful output) |

---

## Rules

**Structure:**
- Stage IDs MUST be `"stage-1"`, `"stage-2"`, etc. in sequential order.
- Every stage MUST include a `"description"` field (one Korean sentence). No exceptions.
- Every stage except `stage-1` MUST include `"depends_on"` listing the preceding stage id(s).
- Pipeline `"name"` MUST be an English slug (lowercase, hyphens only, no spaces).

**Language:**
- ALL user-facing text — pipeline description, stage names, stage descriptions, messages, subject — MUST be written in Korean (한국어).

**Workflow stages:**
- If `workflow_name` matches a name in "Available workflows", use it directly. No `workflow_specs` entry needed.
- If `workflow_name` does NOT match any available workflow, add a `workflow_specs` entry with the same name and a rich description.
- NEVER use a workflow name that is neither in Available workflows nor in your workflow_specs.

**Connection-based stages (notification, approval):**
- Always set `connection_id` to `""`. Never invent a connection ID.

**Trigger stages:**
- Always set `trigger_id` to `""`. Never invent a trigger ID.

**Type safety:**
- NEVER use any stage type other than the seven defined above.

**Output:**
- Your entire response MUST be ONLY the raw JSON object.
- No markdown fences, no explanation, no text before or after the JSON.
```

**Step 2: Run tests**

```bash
go test ./internal/skills/... -v -race
```
Expected: pass (skills tests mostly check file loading).

**Step 3: Commit**

```bash
git add internal/skills/prompts/pipeline-create.md
git commit -m "feat(skills): update pipeline-create prompt for two-phase generation"
```

---

### Task 4: Implement Phase 2 in `generatePipelineCreate`

**Files:**
- Modify: `internal/generate/pipeline.go:246-284`

**Step 1: Write the failing test**

In `internal/generate/pipeline_test.go`:

```go
func TestGeneratePipelineCreateCallsGenerateForNewWorkflows(t *testing.T) {
    // A mock LLM that returns a bundle with a new workflow spec.
    // This test verifies that generatePipelineCreate calls Generate() for new specs
    // and enforces the name from the spec.
    //
    // This is an integration-style test using the existing mock LLM pattern
    // already in the test file. Check the existing tests for mock LLM setup.
    //
    // The key assertion: bundle.Workflows[0].Name == spec.Name
    t.Skip("implement after reviewing existing mock patterns in generate_test.go")
}
```

Run to see it skip:
```bash
go test ./internal/generate/... -v -race -run TestGeneratePipelineCreateCallsGenerateForNewWorkflows
```

**Step 2: Implement Phase 2 in `generatePipelineCreate`**

Replace the current `generatePipelineCreate` function body (lines 246-284) with:

```go
func (g *Generator) generatePipelineCreate(ctx context.Context, description string, availableWorkflows []WorkflowSummary, existingPipelines []PipelineSummary) (*PipelineBundle, error) {
    sysPrompt := g.buildPipelineSysPrompt(g.skills.GetPrompt("pipeline-create"), availableWorkflows, existingPipelines)

    ctx = upalmodel.WithEffort(ctx, "high")

    text, err := g.generateWithSkills(ctx, sysPrompt, description, "generate pipeline bundle")
    if err != nil {
        return nil, err
    }

    var bundle PipelineBundle
    if err := json.NewDecoder(strings.NewReader(text)).Decode(&bundle); err != nil {
        return nil, fmt.Errorf("parse generated pipeline bundle (model output may be malformed): %w\nraw output: %s", err, text)
    }

    if bundle.Pipeline.Name == "" {
        bundle.Pipeline.Name = "generated-pipeline"
    }

    // Build the set of names that are valid for workflow stages:
    // existing workflows + new specs from this generation.
    existingNames := workflowNameSet(availableWorkflows)
    specNames := make(map[string]bool, len(bundle.WorkflowSpecs))
    for _, spec := range bundle.WorkflowSpecs {
        specNames[spec.Name] = true
    }
    validNames := make(map[string]bool, len(existingNames)+len(specNames))
    for k := range existingNames {
        validNames[k] = true
    }
    for k := range specNames {
        validNames[k] = true
    }

    // Keep workflow stages only if their name is in existingNames or specNames.
    bundle.Pipeline.Stages = filterWorkflowStages(bundle.Pipeline.Stages, validNames)

    originalStageCount := len(bundle.Pipeline.Stages)
    bundle.Pipeline.Stages = stripInvalidStageTypes(bundle.Pipeline.Stages)
    if len(bundle.Pipeline.Stages) == 0 && originalStageCount > 0 {
        return nil, fmt.Errorf("generated pipeline has no valid stages (LLM used unsupported stage types or unknown workflow names)")
    }

    for i := range bundle.Pipeline.Stages {
        if bundle.Pipeline.Stages[i].ID == "" {
            bundle.Pipeline.Stages[i].ID = fmt.Sprintf("stage-%d", i+1)
        }
    }

    // Phase 2: generate new workflows for specs not already in availableWorkflows.
    newSpecs := make([]WorkflowSpec, 0, len(bundle.WorkflowSpecs))
    for _, spec := range bundle.WorkflowSpecs {
        if !existingNames[spec.Name] && spec.Name != "" && spec.Description != "" {
            newSpecs = append(newSpecs, spec)
        }
    }

    if len(newSpecs) > 0 {
        // Build updated workflow summaries including the ones we already have.
        allSummaries := availableWorkflows

        type result struct {
            wf  *upal.WorkflowDefinition
            err error
            idx int
        }

        results := make([]result, len(newSpecs))
        var wg sync.WaitGroup
        var mu sync.Mutex

        for i, spec := range newSpecs {
            wg.Add(1)
            go func(i int, spec WorkflowSpec) {
                defer wg.Done()
                wf, err := g.Generate(ctx, spec.Description, nil, allSummaries)
                mu.Lock()
                results[i] = result{wf: wf, err: err, idx: i}
                mu.Unlock()
            }(i, spec)
        }
        wg.Wait()

        for i, res := range results {
            if res.err != nil {
                // Non-fatal: log and skip the failed workflow.
                // The pipeline stage will reference a workflow that doesn't exist yet;
                // the user can generate it separately.
                continue
            }
            // Enforce the name from Phase 1 — Generate() may have chosen a different name.
            res.wf.Name = newSpecs[i].Name
            bundle.Workflows = append(bundle.Workflows, *res.wf)
        }
    }

    return &bundle, nil
}
```

Add `"sync"` to the import block at the top of `pipeline.go`.

**Step 3: Run all generate tests**

```bash
go test ./internal/generate/... -v -race
```
Expected: all pass.

**Step 4: Commit**

```bash
git add internal/generate/pipeline.go
git commit -m "feat(generate): implement Phase 2 workflow generation in generatePipelineCreate"
```

---

### Task 5: Update edit mode (`generatePipelineEdit`)

**Files:**
- Modify: `internal/generate/pipeline.go:151-159` (`PipelineEditDelta`)
- Modify: `internal/generate/pipeline.go:286-350` (`generatePipelineEdit`)

**Step 1: Add `WorkflowSpecs` to `PipelineEditDelta`**

In `PipelineEditDelta` (line 151), add the new field:

```go
type PipelineEditDelta struct {
    Name         string               `json:"name,omitempty"`
    Description  string               `json:"description,omitempty"`
    StageChanges []PipelineStageDelta `json:"stage_changes"`
    StageOrder   []string             `json:"stage_order,omitempty"`
    WorkflowSpecs []WorkflowSpec      `json:"workflow_specs,omitempty"` // new workflows needed by added stages
    Workflows    []upal.WorkflowDefinition `json:"workflows"`            // ignored — managed via WorkflowSpecs
}
```

**Step 2: Update `generatePipelineEdit` to run Phase 2**

After `applyDelta` is called and before `return`, add:

```go
    merged := applyDelta(existing, &delta)

    // Backfill missing stage IDs ... (existing code)

    // Phase 2: generate new workflows for specs in the delta.
    if len(delta.WorkflowSpecs) > 0 {
        existingNames := workflowNameSet(availableWorkflows)
        var wg sync.WaitGroup
        var mu sync.Mutex
        var generatedWorkflows []upal.WorkflowDefinition

        for _, spec := range delta.WorkflowSpecs {
            if existingNames[spec.Name] || spec.Name == "" || spec.Description == "" {
                continue
            }
            spec := spec // capture loop variable
            wg.Add(1)
            go func() {
                defer wg.Done()
                wf, err := g.Generate(ctx, spec.Description, nil, availableWorkflows)
                if err != nil {
                    return
                }
                wf.Name = spec.Name
                mu.Lock()
                generatedWorkflows = append(generatedWorkflows, *wf)
                mu.Unlock()
            }()
        }
        wg.Wait()

        return &PipelineBundle{Pipeline: *merged, Workflows: generatedWorkflows}, nil
    }

    return &PipelineBundle{Pipeline: *merged}, nil
```

Also update the stage validation in edit mode to allow spec names. After `validWF := workflowNameSet(availableWorkflows)`:

```go
    validWF := workflowNameSet(availableWorkflows)
    // Also allow names defined in the delta's workflow_specs.
    for _, spec := range delta.WorkflowSpecs {
        validWF[spec.Name] = true
    }
```

**Step 3: Run tests**

```bash
go test ./internal/generate/... -v -race
```
Expected: all pass.

**Step 4: Commit**

```bash
git add internal/generate/pipeline.go
git commit -m "feat(generate): extend edit mode with Phase 2 workflow generation"
```

---

### Task 6: Save generated workflows in the API handler

**Files:**
- Modify: `internal/api/generate.go:32-75` (`generatePipeline`)

**Step 1: Write the failing test**

In `internal/api/generate_test.go` (check if file exists — if not, look at other `*_test.go` files in `internal/api/` for the test pattern):

```go
func TestGeneratePipelineHandlerSavesGeneratedWorkflows(t *testing.T) {
    // Verify that when bundle.Workflows is non-empty,
    // the handler saves each workflow to the repo.
    // Use the existing httptest + minimal Server pattern from other api tests.
    t.Skip("implement after checking api test patterns")
}
```

**Step 2: Modify `generatePipeline` handler**

After `bundle, err := s.generator.GeneratePipelineBundle(...)` succeeds, add workflow saving:

```go
    bundle, err := s.generator.GeneratePipelineBundle(r.Context(), req.Description, req.ExistingPipeline, workflowSummaries, pipelineSummaries)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Save any workflows generated in Phase 2.
    // Non-fatal: log failures but continue — the pipeline is still returned.
    for i := range bundle.Workflows {
        if err := s.repo.Create(r.Context(), &bundle.Workflows[i]); err != nil {
            log.Printf("generatePipeline: save workflow %q: %v", bundle.Workflows[i].Name, err)
        }
    }
```

> **Note:** Check the exact method name on `s.repo` — it may be `Create`, `Save`, or `Upsert`. Look at how other handlers in `internal/api/workflow.go` create new workflows and use the same method.

**Step 3: Run all API tests**

```bash
go test ./internal/api/... -v -race
```
Expected: all pass.

**Step 4: Run full test suite**

```bash
make test
```
Expected: all pass.

**Step 5: Commit**

```bash
git add internal/api/generate.go
git commit -m "feat(api): save Phase 2 generated workflows in generatePipeline handler"
```

---

### Task 7: End-to-end verification

**Step 1: Build and run**

```bash
make build && ./bin/upal serve
```

**Step 2: Generate a pipeline that needs new workflows**

```bash
curl -s -X POST http://localhost:8080/api/generate-pipeline \
  -H "Content-Type: application/json" \
  -d '{"description": "뉴스 RSS를 매일 수집해서 요약하고 슬랙으로 보내는 파이프라인"}' \
  | jq '{pipeline_name: .pipeline.name, stage_count: (.pipeline.stages | length), workflow_count: (.workflows | length)}'
```

Expected: `workflow_count > 0` when workflows were needed and generated.

**Step 3: Verify generated workflows are saved**

```bash
curl -s http://localhost:8080/api/workflows | jq '[.[] | .name]'
```

Expected: the new workflow names appear in the list.

**Step 4: Commit any final cleanup**

```bash
git add -A
git commit -m "feat: pipeline two-phase generation complete"
```
