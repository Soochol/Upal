# Pipeline Two-Phase Generation Design

**Date:** 2026-02-23
**Status:** Approved

## Problem

Pipeline generation quality is significantly lower than direct workflow generation because:
1. `pipeline-create.md` has no tool/model context — LLM writes vague workflow descriptions
2. `bundle.Workflows = nil` forces pipeline LLM to generate workflows AND manage their quality in one shot
3. `filterWorkflowStages()` strips any workflow stage referencing a non-existent workflow — LLM can only use what already exists

## Solution: Two-Phase Orchestration

Pipeline generation is split into two phases that share the same workflow generation procedure.

```
[Phase 1] Pipeline LLM
  Input:  user description + available workflows + models + tools
  Output: pipeline skeleton + workflow_specs[] (name + description for new workflows)

[Phase 2] Workflow LLM × N  (parallel, only for new workflows)
  Input:  spec.description (from Phase 1)  →  Generate()  (identical to direct generation)
  Output: WorkflowDefinition[]  with wf.Name = spec.Name enforced

[Final Bundle]
  pipeline + workflows[]  →  API saves workflows to repo  →  returns full bundle
```

## Data Flow

```
POST /api/generate-pipeline
  ↓
generatePipelineCreate()
  Phase 1: LLM → PipelineBundle{ pipeline, workflow_specs[] }
  Filter:  specs for already-existing workflows → skipped (Phase 2 not called)
  Phase 2: parallel Generate(spec.description) for each new spec
           enforce wf.Name = spec.Name
  Result:  bundle.Workflows = generated WorkflowDefinition[]
  ↓
generatePipeline() handler
  Save each bundle.Workflows[i] to repo
  Generate pipeline thumbnail
  Return bundle (pipeline + workflows)
```

## Changes

### 1. `internal/skills/prompts/pipeline-create.md`

- **Remove** the rule "Return ONLY the pipeline JSON — do NOT generate workflow definitions"
- **Remove** the rule "NEVER include a `workflows` key"
- **Remove** the rule "workflow_name MUST match Available workflows list — NEVER invent"
- **Add** `workflow_specs` output schema — LLM may propose new workflows with a description
- **Add** instruction: if the needed workflow doesn't exist in the Available list, add a spec entry instead of omitting the stage
- Tool and model lists are injected **programmatically** (not in the .md file)

New output schema:
```json
{
  "pipeline": { "name": "...", "description": "...", "stages": [...] },
  "workflow_specs": [
    { "name": "exact-slug", "description": "Rich description for Generate() — mention tools and data flow" }
  ]
}
```

`workflow_specs` entries:
- Only for **new** workflows (not in the Available workflows list)
- `name` must be a valid English slug (used as the workflow's `Name` after generation)
- `description` should be rich enough for the workflow LLM to make good decisions (mention data inputs, expected output, tools to use, model tier)
- Workflow stages in `pipeline.stages` MUST use `workflow_name` matching a spec name or an existing workflow name

### 2. `internal/generate/pipeline.go`

**New type:**
```go
type WorkflowSpec struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}
```

**Modified `PipelineBundle`:**
```go
type PipelineBundle struct {
    Pipeline      upal.Pipeline             `json:"pipeline"`
    WorkflowSpecs []WorkflowSpec            `json:"workflow_specs"`  // Phase 1 output
    Workflows     []upal.WorkflowDefinition `json:"-"`               // Phase 2 output (not in JSON)
}
```

**Modified `generatePipelineCreate()`:**
1. Parse Phase 1 output (pipeline + workflow_specs)
2. Filter specs: skip names already in `availableWorkflows`
3. `filterWorkflowStages()` updated: allow names that appear in workflow_specs OR in availableWorkflows
4. Phase 2: parallel `g.Generate()` for each new spec, force `wf.Name = spec.Name`
5. Remove `bundle.Workflows = nil`

**Modified `buildPipelineSysPrompt()`:**
- Inject model list (same as `Generate()` does via `g.buildModelPrompt()`)
- Inject tool list (same as `Generate()` does)

### 3. `internal/api/generate.go`

**Modified `generatePipeline()` handler:**
After `GeneratePipelineBundle()` returns:
```go
for i := range bundle.Workflows {
    if err := s.repo.Save(ctx, &bundle.Workflows[i]); err != nil {
        log.Printf("generatePipeline: save workflow %q: %v", bundle.Workflows[i].Name, err)
        // non-fatal — pipeline still returned
    }
}
```

## Invariant: Name Enforcement

Phase 1 outputs `workflow_specs[i].Name` = `"article-summarizer"`.
Phase 2 calls `Generate(spec.description)` which may choose a different name internally.
After generation: `wf.Name = spec.Name` is enforced before adding to bundle.
This guarantees pipeline stage `workflow_name: "article-summarizer"` resolves correctly.

## Edit Mode

`generatePipelineEdit()` follows the same pattern:
- Phase 1 delta may include new `workflow_specs`
- Phase 2 generates them using `Generate()`
- New workflows saved to repo in handler
- `delta.Workflows = nil` guard updated to `delta.Workflows = nil` only (specs handled separately)

## What Does NOT Change

- `generateWithSkills()` — shared multi-turn LLM loop, unchanged
- `Generate()` — direct workflow generation, completely unchanged
- `workflow-create.md` — unchanged, Phase 2 uses it as-is
- All other stage types (collect, notification, approval, etc.) — unchanged
- Frontend — no changes needed; bundle already has `workflows` field

## Quality Guarantee

After this change, pipeline-generated workflows go through the exact same code path as directly-generated workflows:
- Same `Generate()` function
- Same `workflow-create.md` prompt
- Same model/tool/framework injections
- Same post-processing (stripInvalidNodeTypes, fixInvalidModels, etc.)
