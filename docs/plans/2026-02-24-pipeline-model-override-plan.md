# Pipeline Model Override — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow each pipeline to specify an LLM model for the analysis stage, overriding the system default.

**Architecture:** Add `LLMResolver` port interface + map-based implementation. `Pipeline.Model` field flows through existing CRUD. `ContentCollector` resolves the model at analysis time. Frontend adds a model dropdown to the Settings panel.

**Tech Stack:** Go 1.23, PostgreSQL, React 19, TypeScript, Zustand, TanStack Query

---

### Task 1: Add `Model` field to Pipeline domain type

**Files:**
- Modify: `internal/upal/pipeline.go:6-21`

**Step 1: Add the field**

In `internal/upal/pipeline.go`, add `Model` to the `Pipeline` struct after `Schedule`:

```go
Schedule             string     `json:"schedule,omitempty"`
Model                string     `json:"model,omitempty"`                 // "provider/model" format, empty = system default
LastCollectedAt      *time.Time `json:"last_collected_at,omitempty"`
```

**Step 2: Verify compilation**

Run: `go build ./internal/upal/...`
Expected: PASS (no other files reference the new field yet)

**Step 3: Commit**

```bash
git add internal/upal/pipeline.go
git commit -m "feat(domain): add Model field to Pipeline struct"
```

---

### Task 2: Add `LLMResolver` port interface

**Files:**
- Create: `internal/upal/ports/llm.go`

**Step 1: Create the port**

```go
package ports

import adkmodel "google.golang.org/adk/model"

// LLMResolver resolves a "provider/model" ID string to an LLM instance
// and the model name to use in LLMRequest. Empty modelID returns the
// system default.
type LLMResolver interface {
	Resolve(modelID string) (adkmodel.LLM, string, error)
}
```

**Step 2: Verify compilation**

Run: `go build ./internal/upal/...`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/upal/ports/llm.go
git commit -m "feat(ports): add LLMResolver interface"
```

---

### Task 3: Implement `MapResolver`

**Files:**
- Create: `internal/llmutil/resolver.go`

**Step 1: Write the implementation**

```go
package llmutil

import (
	"fmt"
	"strings"

	adkmodel "google.golang.org/adk/model"
)

// MapResolver implements ports.LLMResolver using a map of provider name → LLM.
type MapResolver struct {
	llms         map[string]adkmodel.LLM
	defaultLLM   adkmodel.LLM
	defaultModel string
}

// NewMapResolver creates a resolver. llms keys are provider names (e.g. "anthropic").
func NewMapResolver(llms map[string]adkmodel.LLM, defaultLLM adkmodel.LLM, defaultModel string) *MapResolver {
	return &MapResolver{llms: llms, defaultLLM: defaultLLM, defaultModel: defaultModel}
}

// Resolve parses "provider/model" and returns the matching LLM + model name.
// Empty modelID returns the system default.
func (r *MapResolver) Resolve(modelID string) (adkmodel.LLM, string, error) {
	if modelID == "" {
		return r.defaultLLM, r.defaultModel, nil
	}
	provider, modelName, ok := strings.Cut(modelID, "/")
	if !ok {
		return nil, "", fmt.Errorf("invalid model ID %q: expected provider/model format", modelID)
	}
	llm, found := r.llms[provider]
	if !found {
		return nil, "", fmt.Errorf("unknown provider %q in model ID %q", provider, modelID)
	}
	return llm, modelName, nil
}
```

**Step 2: Write test**

Create `internal/llmutil/resolver_test.go`:

```go
package llmutil

import (
	"testing"
)

func TestMapResolver_EmptyReturnsDefault(t *testing.T) {
	r := NewMapResolver(nil, nil, "default-model")
	_, model, err := r.Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "default-model" {
		t.Errorf("expected default-model, got %s", model)
	}
}

func TestMapResolver_InvalidFormat(t *testing.T) {
	r := NewMapResolver(nil, nil, "")
	_, _, err := r.Resolve("no-slash")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestMapResolver_UnknownProvider(t *testing.T) {
	r := NewMapResolver(map[string]adkmodel.LLM{}, nil, "")
	_, _, err := r.Resolve("unknown/model")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
```

Note: `TestMapResolver_EmptyReturnsDefault` tests with `nil` LLM since we only check the model string. The actual LLM instance is opaque to this test.

**Step 3: Run tests**

Run: `go test ./internal/llmutil/... -v -race`
Expected: PASS (3 tests)

**Step 4: Commit**

```bash
git add internal/llmutil/resolver.go internal/llmutil/resolver_test.go
git commit -m "feat(llmutil): add MapResolver implementing LLMResolver port"
```

---

### Task 4: Refactor `ContentCollector` to use `LLMResolver`

**Files:**
- Modify: `internal/services/content_collector.go:22-48` (struct + constructor)
- Modify: `internal/services/content_collector.go:200-232` (`runAnalysis`)

**Step 1: Update struct and constructor**

Replace the `llm` and `model` fields with `resolver`:

```go
type ContentCollector struct {
	contentSvc   *ContentSessionService
	collectExec  *CollectStageExecutor
	workflowSvc  *WorkflowService
	workflowRepo repository.WorkflowRepository
	resolver     ports.LLMResolver
}

func NewContentCollector(
	contentSvc *ContentSessionService,
	collectExec *CollectStageExecutor,
	workflowSvc *WorkflowService,
	workflowRepo repository.WorkflowRepository,
	resolver ports.LLMResolver,
) *ContentCollector {
	return &ContentCollector{
		contentSvc:   contentSvc,
		collectExec:  collectExec,
		workflowSvc:  workflowSvc,
		workflowRepo: workflowRepo,
		resolver:     resolver,
	}
}
```

Add import: `"github.com/soochol/upal/internal/upal/ports"`

**Step 2: Update `runAnalysis` to resolve from pipeline**

In `runAnalysis()`, replace direct `c.llm` / `c.model` usage:

```go
func (c *ContentCollector) runAnalysis(ctx context.Context, pipeline *upal.Pipeline, session *upal.ContentSession) {
	fetches, err := c.contentSvc.ListSourceFetches(ctx, session.ID)
	if err != nil {
		log.Printf("content_collector: failed to list source fetches for analysis: %v", err)
		return
	}

	llm, modelName, err := c.resolver.Resolve(pipeline.Model)
	if err != nil {
		log.Printf("content_collector: failed to resolve model %q: %v", pipeline.Model, err)
		return
	}

	systemPrompt, userPrompt := buildAnalysisPrompt(pipeline, fetches)

	req := &adkmodel.LLMRequest{
		Model: modelName,
		// ... rest unchanged, but use llm variable below
	}

	// ... existing code, but replace c.llm.GenerateContent with llm.GenerateContent
```

**Step 3: Verify compilation**

Run: `go build ./internal/services/...`
Expected: FAIL — `cmd/upal/main.go` still passes old args. That's expected; Task 5 fixes it.

**Step 4: Commit (allow build to temporarily break across tasks)**

```bash
git add internal/services/content_collector.go
git commit -m "refactor(services): ContentCollector uses LLMResolver instead of hardwired LLM"
```

---

### Task 5: Wire resolver in `main.go`

**Files:**
- Modify: `cmd/upal/main.go:286-296`

**Step 1: Create resolver and pass to collector**

Replace the existing `NewContentCollector` call:

```go
	// Wire content collector for actual source fetching and workflow execution.
	if defaultLLM != nil {
		resolver := llmutil.NewMapResolver(llms, defaultLLM, defaultModelName)
		collector := services.NewContentCollector(
			contentSvc,
			services.NewCollectStageExecutor(),
			workflowSvc,
			repo,
			resolver,
		)
		srv.SetContentCollector(collector)
	}
```

Add import: `"github.com/soochol/upal/internal/llmutil"`

**Step 2: Verify full build**

Run: `go build ./cmd/upal/...`
Expected: PASS

**Step 3: Run all tests**

Run: `go test ./... -v -race -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/upal/main.go
git commit -m "feat(wiring): create MapResolver and inject into ContentCollector"
```

---

### Task 6: Add `model` column to database

**Files:**
- Modify: `internal/db/db.go:240` (append migration)
- Modify: `internal/db/pipeline.go` (all CRUD queries)

**Step 1: Add migration**

Append to `migrationSQL` in `internal/db/db.go`, before the closing backtick:

```sql
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS model TEXT NOT NULL DEFAULT '';
```

**Step 2: Update pipeline CRUD queries**

In `internal/db/pipeline.go`:

**CreatePipeline** — add `model` to INSERT:
```go
_, err = d.Pool.ExecContext(ctx,
    `INSERT INTO pipelines (id, name, description, stages, context, sources, schedule, model, created_at, updated_at)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
    p.ID, p.Name, p.Description, stagesJSON, ctxJSON, sourcesJSON, p.Schedule, p.Model, p.CreatedAt, p.UpdatedAt,
)
```

**GetPipeline** — add `model` to SELECT and Scan:
```go
err := d.Pool.QueryRowContext(ctx,
    `SELECT id, name, description, stages, context, sources, schedule, model, created_at, updated_at
     FROM pipelines WHERE id = $1`, id,
).Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &ctxJSON, &sourcesJSON, &p.Schedule, &p.Model, &p.CreatedAt, &p.UpdatedAt)
```

**ListPipelines** — same pattern, add `model` to SELECT and Scan:
```go
rows, err := d.Pool.QueryContext(ctx,
    `SELECT id, name, description, stages, context, sources, schedule, model, created_at, updated_at
     FROM pipelines ORDER BY updated_at DESC`,
)
// In scan:
rows.Scan(&p.ID, &p.Name, &p.Description, &stagesJSON, &ctxJSON, &sourcesJSON, &p.Schedule, &p.Model, &p.CreatedAt, &p.UpdatedAt)
```

**UpdatePipeline** — add `model` to UPDATE:
```go
res, err := d.Pool.ExecContext(ctx,
    `UPDATE pipelines SET name = $1, description = $2, stages = $3, context = $4, sources = $5, schedule = $6, model = $7, updated_at = $8
     WHERE id = $9`,
    p.Name, p.Description, stagesJSON, ctxJSON, sourcesJSON, p.Schedule, p.Model, p.UpdatedAt, p.ID,
)
```

**Step 3: Verify build**

Run: `go build ./internal/db/...`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/db/db.go internal/db/pipeline.go
git commit -m "feat(db): add model column to pipelines table"
```

---

### Task 7: Add `model` to frontend Pipeline type

**Files:**
- Modify: `web/src/shared/types/index.ts:226-241`

**Step 1: Add model field**

In the `Pipeline` type, add after `schedule`:

```typescript
export type Pipeline = {
  id: string
  name: string
  description?: string
  stages: Stage[]
  thumbnail_svg?: string
  // Content pipeline extensions
  sources?: PipelineSource[]
  workflows?: PipelineWorkflow[]
  context?: PipelineContext
  schedule?: string
  model?: string              // "provider/model" format, empty = system default
  last_collected_at?: string
  pending_session_count?: number
  created_at: string
  updated_at: string
}
```

**Step 2: Verify frontend type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS (no consumers of the field yet)

**Step 3: Commit**

```bash
git add web/src/shared/types/index.ts
git commit -m "feat(types): add model field to Pipeline type"
```

---

### Task 8: Add Model dropdown to Settings panel

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

**Step 1: Add `localModel` state alongside existing local states**

In `PipelineDetailPage`, after `const [localWorkflows, setLocalWorkflows] = ...`:

```typescript
const [localModel, setLocalModel] = useState('')
```

In the `useEffect` that syncs from pipeline:

```typescript
useEffect(() => {
  if (pipeline) {
    setLocalSources(pipeline.sources ?? [])
    setLocalSchedule(pipeline.schedule ?? '')
    setLocalWorkflows(pipeline.workflows ?? [])
    setLocalModel(pipeline.model ?? '')
  }
}, [pipeline?.id])
```

Add ref:

```typescript
const localModelRef = useRef(localModel)
localModelRef.current = localModel
```

**Step 2: Add model to dirty check and save**

Update `isDirty` memo:

```typescript
const isDirty = useMemo(() => {
  if (!pipeline) return false
  return (
    JSON.stringify(localSources) !== JSON.stringify(pipeline.sources ?? []) ||
    localSchedule !== (pipeline.schedule ?? '') ||
    JSON.stringify(localWorkflows) !== JSON.stringify(pipeline.workflows ?? []) ||
    localModel !== (pipeline.model ?? '')
  )
}, [localSources, localSchedule, localWorkflows, localModel, pipeline])
```

Update `doSave`:

```typescript
await updatePipeline(id!, {
  ...p,
  sources: localSourcesRef.current,
  schedule: localScheduleRef.current,
  workflows: localWorkflowsRef.current,
  model: localModelRef.current,
})
```

Update the auto-save effect dependency array:

```typescript
useEffect(() => {
  if (!isDirty) return
  const timer = setTimeout(() => { void doSaveRef.current() }, 800)
  return () => clearTimeout(timer)
}, [localSources, localSchedule, localWorkflows, localModel, isDirty])
```

**Step 3: Pass model props to `PipelineSettingsPanel`**

Add to the `PipelineSettingsPanel` component call:

```tsx
<PipelineSettingsPanel
  sources={localSources}
  schedule={localSchedule}
  context={pipeline.context}
  workflows={localWorkflows}
  model={localModel}
  onSourcesChange={setLocalSources}
  onScheduleChange={setLocalSchedule}
  onContextSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
  onWorkflowsChange={setLocalWorkflows}
  onModelChange={setLocalModel}
  autoSaveStatus={autoSaveStatus}
/>
```

**Step 4: Update `PipelineSettingsPanel` to accept and render model dropdown**

Add props:

```typescript
function PipelineSettingsPanel({
  sources, schedule, context, workflows, model,
  onSourcesChange, onScheduleChange, onContextSave, onWorkflowsChange, onModelChange, autoSaveStatus,
}: {
  sources: PipelineSource[]
  schedule: string
  context: PipelineContext | undefined
  workflows: PipelineWorkflow[]
  model: string
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onContextSave: (ctx: PipelineContext) => Promise<void>
  onWorkflowsChange: (w: PipelineWorkflow[]) => void
  onModelChange: (model: string) => void
  autoSaveStatus: 'idle' | 'saving' | 'saved'
}) {
```

Add state:

```typescript
const [modelOpen, setModelOpen] = useState(false)
```

Add section before the Editorial Brief section (after Workflows):

```tsx
<section>
  <button
    onClick={() => setModelOpen(v => !v)}
    className="w-full flex items-center justify-between mb-2 cursor-pointer"
  >
    <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
      Analysis Model
    </span>
    <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${modelOpen ? 'rotate-180' : ''}`} />
  </button>
  {modelOpen && (
    <div className="space-y-2">
      <p className="text-[11px] text-muted-foreground">
        Model used for content analysis. Leave empty for system default.
      </p>
      <ModelSelector
        value={model}
        onChange={onModelChange}
        placeholder="System Default"
      />
      {model && (
        <button
          onClick={() => onModelChange('')}
          className="text-[11px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        >
          Reset to default
        </button>
      )}
    </div>
  )}
</section>
```

Add import at top of file:

```typescript
import { ModelSelector } from '@/shared/ui/ModelSelector'
```

**Step 5: Verify frontend type-check and dev server**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 6: Commit**

```bash
git add web/src/pages/pipelines/PipelineDetail.tsx web/src/shared/types/index.ts
git commit -m "feat(frontend): add model dropdown to pipeline settings panel"
```

---

### Task 9: End-to-end verification

**Step 1: Start dev servers**

Run: `make dev-backend` (terminal 1) and `make dev-frontend` (terminal 2)

**Step 2: Manual test checklist**

1. Open pipeline detail page → Settings panel → "Analysis Model" section
2. Expand and verify model dropdown loads models from `/api/models`
3. Select a model → verify auto-save triggers (Saved indicator)
4. Refresh page → verify model selection persists
5. Click "Reset to default" → verify it clears back to "System Default"
6. Start a new session → verify analysis runs with the selected model (check backend logs for model name)

**Step 3: Run full test suite**

Run: `go test ./... -v -race -count=1`
Expected: PASS

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS
