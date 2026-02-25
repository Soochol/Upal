# Research Source Type Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an LLM-powered "research" source type to the pipeline collect stage, supporting light (single search) and deep (agent loop) modes.

**Architecture:** New `researchFetcher` implements the existing `SourceFetcher` interface. Light mode does a single LLM call with web_search. Deep mode runs an ADK agent loop with web_search + get_webpage tools, capped by `max_searches`. Both modes return results in the same `(text, data, error)` format as existing fetchers.

**Tech Stack:** Go (ADK agent, genai), React/TypeScript (AddSourceModal), Tailwind CSS, skill markdown files.

**Design doc:** `docs/plans/2026-02-25-research-source-design.md`

---

### Task 1: Data Model — Add research fields to CollectSource and PipelineSource

**Files:**
- Modify: `internal/upal/pipeline.go:34-49` (CollectSource struct)
- Modify: `internal/upal/pipeline.go:94-111` (PipelineSource struct)

**Step 1: Add fields to CollectSource**

In `internal/upal/pipeline.go`, add after the `Geo` field (line 48):

```go
	Topic       string `json:"topic,omitempty"`        // Research: subject to investigate
	Depth       string `json:"depth,omitempty"`         // Research: "light" | "deep"
	MaxSearches int    `json:"max_searches,omitempty"`  // Research: max search iterations for deep (default 10)
```

**Step 2: Add fields to PipelineSource**

In the same file, add after the `Limit` field (line 111):

```go
	Topic string `json:"topic,omitempty"` // research: subject to investigate
	Depth string `json:"depth,omitempty"` // research: "light" | "deep"
```

**Step 3: Add frontend type**

In `web/src/shared/types.ts` (or wherever `PipelineSourceType` is defined), add `'research'` to the union type.

**Step 4: Commit**

```bash
git add internal/upal/pipeline.go web/src/shared/types.ts
git commit -m "feat: add research fields to CollectSource and PipelineSource"
```

---

### Task 2: Source Mapping — Handle research type in mapPipelineSources and convertToSourceItems

**Files:**
- Modify: `internal/services/content_collector.go:874-943` (mapPipelineSources)
- Modify: `internal/services/content_collector.go:237-303` (convertToSourceItems)

**Step 1: Add research case to mapPipelineSources**

In the `switch ps.Type` block, add before the `default:` case:

```go
		case "research":
			cs.Type = "research"
			cs.Topic = ps.Topic
			cs.Depth = ps.Depth
			if cs.Depth == "" {
				cs.Depth = "light"
			}
```

**Step 2: Add research case to convertToSourceItems**

Add a new case in the `switch sourceType` block:

```go
	case "research":
		// Research fetcher returns []map[string]any with title, url, summary.
		items, ok := data.([]map[string]any)
		if !ok {
			return nil
		}
		result := make([]upal.SourceItem, 0, len(items))
		for _, item := range items {
			result = append(result, upal.SourceItem{
				Title:   stringVal(item, "title"),
				URL:     stringVal(item, "url"),
				Content: stringVal(item, "summary"),
			})
		}
		return result
```

**Step 3: Commit**

```bash
git add internal/services/content_collector.go
git commit -m "feat: add research source mapping and conversion"
```

---

### Task 3: Skill File — Create stage-research.md

**Files:**
- Create: `internal/skills/stages/stage-research.md`
- Modify: `internal/skills/stages/stage-collect.md` (add research type)

**Step 1: Create the research skill file**

Create `internal/skills/stages/stage-research.md`:

```markdown
---
name: stage-research
description: LLM-powered topic research — light (single search) and deep (agent loop) modes
---

## Light Mode — System Prompt

You are a research analyst. Your task is to find current, relevant information about a given topic.

**Tools available:** `web_search`, `get_webpage`

**Process:**
1. Generate 1-2 precise search queries for the topic
2. Execute web_search
3. Use get_webpage to read the 3-5 most relevant results
4. Synthesize findings into a concise markdown report

**Output format:**

```
## Research: {topic}

### Summary
3-5 sentence overview of key findings.

### Key Findings
- Finding 1 with supporting detail
- Finding 2 with supporting detail
- ...

### Sources
- [Title](URL) — one-line description
- ...
```

**Constraints:**
- Focus on recent, factual information
- Cite all sources with URLs
- Do not speculate beyond what sources state
- Keep report under 1000 words

## Deep Mode — System Prompt

You are an expert research analyst conducting deep investigation on a topic. You have access to web search and can read full web pages.

**Tools available:** `web_search`, `get_webpage`

**Process:**
1. Decompose the topic into 3-5 sub-questions that together provide comprehensive coverage
2. For each sub-question:
   a. Generate a targeted search query
   b. Execute web_search
   c. Read the 2-3 most relevant results with get_webpage
   d. Record key findings
3. After each round, evaluate: "Do I have enough information for a comprehensive report?"
   - If NO: generate additional sub-questions for gaps and continue
   - If YES: proceed to synthesis
4. Write a structured research report

**Output format:**

```
## Deep Research: {topic}

### Executive Summary
5-8 sentence overview covering all major findings.

### Detailed Findings

#### {Sub-topic 1}
Findings with supporting evidence...

#### {Sub-topic 2}
Findings with supporting evidence...

...

### Sources
- [Title](URL) — one-line description
- ...
```

**Constraints:**
- Be thorough but efficient — don't search for the same thing twice
- Stop when additional searches yield diminishing returns
- Cite all sources with URLs
- Cross-reference claims across multiple sources when possible
- Keep report under 3000 words
```

**Step 2: Update stage-collect.md**

Add `research` to the source types section in `internal/skills/stages/stage-collect.md`:

```markdown
- **research**: `topic` (required), `depth` ("light" | "deep", default "light"). LLM-powered topic research using web search. Light mode does a single search pass; deep mode runs an iterative agent loop. Requires a model that supports native web_search tool (not available on Ollama).
```

**Step 3: Commit**

```bash
git add internal/skills/stages/stage-research.md internal/skills/stages/stage-collect.md
git commit -m "feat: add research stage skill and update collect docs"
```

---

### Task 4: Research Fetcher — Implement researchFetcher (light mode first)

**Files:**
- Create: `internal/services/fetcher_research.go`

**Step 1: Create the fetcher file**

Create `internal/services/fetcher_research.go`. The fetcher needs access to an LLM resolver and skills provider to run LLM calls with tools.

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type researchFetcher struct {
	resolver ports.LLMResolver
	skills   skills.Provider
	modelID  string // "provider/model" format
}

func NewResearchFetcher(resolver ports.LLMResolver, skills skills.Provider, modelID string) *researchFetcher {
	return &researchFetcher{resolver: resolver, skills: skills, modelID: modelID}
}

func (f *researchFetcher) Type() string { return "research" }

func (f *researchFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	if src.Topic == "" {
		return "", nil, fmt.Errorf("research source requires a topic")
	}

	// Validate model supports native tools (web_search).
	llm, err := f.resolver.Resolve(f.modelID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve model %q: %w", f.modelID, err)
	}
	if _, ok := llm.(model.NativeToolProvider); !ok {
		return "", nil, fmt.Errorf("model %q does not support web search (native tools required)", f.modelID)
	}

	depth := src.Depth
	if depth == "" {
		depth = "light"
	}

	switch depth {
	case "light":
		return f.fetchLight(ctx, src, llm)
	case "deep":
		return f.fetchDeep(ctx, src, llm)
	default:
		return "", nil, fmt.Errorf("unknown research depth %q", depth)
	}
}
```

The `fetchLight` and `fetchDeep` methods create temporary ADK agents with web_search + get_webpage tools and run them. The system prompts come from the `stage-research` skill file.

Key implementation details:
- Light: single agent run with 30s timeout, system prompt from skill light section
- Deep: agent run with 5min timeout, max_searches enforcement via tool call counting
- Both return `(markdownText, []map[string]any{title, url, summary}, error)`
- `model.LookupNativeTool("web_search")` provides the genai.Tool spec for web_search
- `get_webpage` tool is resolved from the custom tool registry

**Step 2: Commit**

```bash
git add internal/services/fetcher_research.go
git commit -m "feat: implement researchFetcher with light and deep modes"
```

---

### Task 5: Register Fetcher — Wire researchFetcher into the system

**Files:**
- Modify: `internal/services/stage_collect.go:32-42` (NewCollectStageExecutor)
- Modify: `internal/services/content_collector.go:27-37` (ContentCollector struct)
- Modify: `cmd/upal/main.go` (wiring)

**Step 1: Add resolver and skills to CollectStageExecutor**

The `researchFetcher` requires `LLMResolver` and `skills.Provider` — these must be passed to `NewCollectStageExecutor` or the fetcher must be registered separately after construction.

Preferred approach: register the research fetcher in `ContentCollector` construction, since it already has access to `resolver` and `skills`:

In `NewContentCollector()`, after creating the collector, register the research fetcher:

```go
// Register research fetcher with access to LLM resolver and skills.
// modelID will be set per-session at fetch time via a wrapper.
collectExec.RegisterFetcher(NewResearchFetcher(resolver, skills, ""))
```

However, the `modelID` varies per pipeline. The fetcher needs to know which model to use at fetch time. Two options:

**Option A**: Pass modelID via CollectSource field (add a `Model` field).
**Option B**: Create the research fetcher per-collection with the correct modelID.

Option A is simpler — add a `Model string` field to CollectSource, set it during `mapPipelineSources` from the pipeline's model config. The fetcher reads `src.Model` at fetch time instead of using a stored modelID.

Add to CollectSource:
```go
Model string `json:"model,omitempty"` // Research: LLM model ID for research
```

In mapPipelineSources, the model ID comes from the pipeline. This requires passing the pipeline model to the mapping function. Check how the pipeline model is accessed in `CollectAndAnalyze` and thread it through.

**Step 2: Commit**

```bash
git add internal/services/stage_collect.go internal/services/content_collector.go internal/upal/pipeline.go cmd/upal/main.go
git commit -m "feat: wire researchFetcher into collect stage"
```

---

### Task 6: Frontend — Add Research source type to AddSourceModal

**Files:**
- Modify: `web/src/features/configure-pipeline-sources/AddSourceModal.tsx`

**Step 1: Add RESEARCH_SOURCES array**

After the `SIGNAL_SOURCES` array (line 27), add:

```typescript
import { Search } from 'lucide-react'

export const RESEARCH_SOURCES: SourceTypeDef[] = [
  {
    type: 'research',
    source_type: 'research',
    label: 'AI Research',
    description: 'LLM-powered topic research',
    icon: <Search className="h-4 w-4" />,
    accent: 'bg-[oklch(0.7_0.15_280)]/12',
    accentText: 'text-[oklch(0.6_0.15_280)]',
  },
]
```

**Step 2: Add Research category to the selection step**

In the source type selection grid (step === 'select'), add a third section after Signal:

```tsx
{/* Research */}
<div>
  <p className="text-xs font-medium text-muted-foreground mb-2">Research</p>
  <div className="space-y-1.5">
    {RESEARCH_SOURCES.map(src => (
      // same card rendering as Static/Signal
    ))}
  </div>
</div>
```

**Step 3: Add Research config fields**

In the config step (step === 'config'), add conditional fields for `draft.type === 'research'`:

```tsx
{draft.type === 'research' && (
  <>
    {/* Topic */}
    <div>
      <label className="text-sm font-medium">Topic</label>
      <input
        className="..."
        placeholder="e.g. EV market trends 2026"
        value={draft.topic ?? ''}
        onChange={e => setDraft(d => ({ ...d, topic: e.target.value }))}
      />
    </div>
    {/* Depth */}
    <div>
      <label className="text-sm font-medium">Depth</label>
      <div className="flex gap-3 mt-1">
        <label className="flex items-center gap-1.5 text-sm">
          <input
            type="radio"
            name="depth"
            value="light"
            checked={(draft.depth ?? 'light') === 'light'}
            onChange={() => setDraft(d => ({ ...d, depth: 'light' }))}
          />
          Light — quick search + summary
        </label>
        <label className="flex items-center gap-1.5 text-sm">
          <input
            type="radio"
            name="depth"
            value="deep"
            checked={draft.depth === 'deep'}
            onChange={() => setDraft(d => ({ ...d, depth: 'deep' }))}
          />
          Deep — iterative research
        </label>
      </div>
    </div>
  </>
)}
```

**Step 4: Ollama blocking**

If the pipeline's model starts with `ollama/`, filter out RESEARCH_SOURCES from the selection step. Pass the pipeline model as a prop to AddSourceModal and conditionally render:

```tsx
{!pipelineModel?.startsWith('ollama/') && RESEARCH_SOURCES.length > 0 && (
  // Research section
)}
```

**Step 5: Update PipelineSource type**

Ensure the frontend `PipelineSource` type includes `topic?: string` and `depth?: string`.

**Step 6: Commit**

```bash
git add web/src/features/configure-pipeline-sources/AddSourceModal.tsx web/src/shared/types.ts
git commit -m "feat: add AI Research source type to AddSourceModal"
```

---

### Task 7: Progress Feedback — Add progress field to SourceFetch

**Files:**
- Modify: `internal/upal/content.go:57-68` (SourceFetch struct)
- Modify: `internal/services/fetcher_research.go` (update progress during deep mode)

**Step 1: Add Progress field to SourceFetch**

```go
type SourceFetch struct {
	// ... existing fields ...
	Progress *ResearchProgress `json:"progress,omitempty"` // Research: live progress for deep mode
}

type ResearchProgress struct {
	CurrentStep  int    `json:"current_step"`
	MaxSteps     int    `json:"max_steps"`
	CurrentQuery string `json:"current_query,omitempty"`
	FindingsCount int   `json:"findings_count"`
}
```

**Step 2: Update researchFetcher deep mode**

In `fetchDeep`, after each web_search tool call, update the SourceFetch record in the DB with current progress. This requires passing `ContentSessionService` and `sessionID` to the fetcher (or using a callback).

Preferred approach: pass a progress callback to the fetcher via the CollectSource (or via a context value):

```go
type ResearchProgressFn func(progress ResearchProgress)
```

The `fetchAndRecord` method in content_collector.go sets up the callback before calling the fetcher, which updates the SourceFetch record in the DB.

**Step 3: Frontend display**

In the session detail view, when a source has `progress` set and no `error`, show:

```
● AI Research: {topic}  searching {current_step}/{max_steps}
  └ "{current_query}"
```

The existing 3s polling via `refetchInterval` will pick this up automatically.

**Step 4: Commit**

```bash
git add internal/upal/content.go internal/services/fetcher_research.go internal/services/content_collector.go
git commit -m "feat: add research progress tracking to SourceFetch"
```

---

### Task 8: Testing

**Files:**
- Create: `internal/services/fetcher_research_test.go`

**Step 1: Test research source mapping**

Test that `mapPipelineSources` correctly maps research type:

```go
func TestMapPipelineSources_Research(t *testing.T) {
	sources := []upal.PipelineSource{{
		Type:  "research",
		Topic: "EV market trends",
		Depth: "deep",
	}}
	result := mapPipelineSources(sources, false, 0)
	require.Len(t, result, 1)
	cs := result[0].collectSource
	assert.Equal(t, "research", cs.Type)
	assert.Equal(t, "EV market trends", cs.Topic)
	assert.Equal(t, "deep", cs.Depth)
}
```

**Step 2: Test convertToSourceItems for research type**

```go
func TestConvertToSourceItems_Research(t *testing.T) {
	data := []map[string]any{
		{"title": "EV Report", "url": "https://example.com", "summary": "Market growing"},
	}
	items := convertToSourceItems("research", data)
	require.Len(t, items, 1)
	assert.Equal(t, "EV Report", items[0].Title)
	assert.Equal(t, "https://example.com", items[0].URL)
}
```

**Step 3: Test Ollama rejection**

```go
func TestResearchFetcher_RejectsOllama(t *testing.T) {
	// Mock resolver that returns a model without NativeToolProvider
	fetcher := NewResearchFetcher(mockResolver, mockSkills, "")
	src := upal.CollectSource{Type: "research", Topic: "test", Model: "ollama/llama3"}
	_, _, err := fetcher.Fetch(context.Background(), src)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "native tools required")
}
```

**Step 4: Test depth default**

```go
func TestMapPipelineSources_ResearchDefaultDepth(t *testing.T) {
	sources := []upal.PipelineSource{{
		Type:  "research",
		Topic: "test topic",
		// Depth omitted
	}}
	result := mapPipelineSources(sources, false, 0)
	assert.Equal(t, "light", result[0].collectSource.Depth)
}
```

**Step 5: Run tests**

```bash
go test ./internal/services/... -v -race -run TestResearch
go test ./internal/services/... -v -race -run TestMapPipelineSources
go test ./internal/services/... -v -race -run TestConvertToSourceItems
```

**Step 6: Commit**

```bash
git add internal/services/fetcher_research_test.go
git commit -m "test: add research fetcher unit tests"
```

---

### Task 9: Frontend Type Check

**Step 1: Run type check**

```bash
cd web && npx tsc -b
```

Fix any type errors from the new `topic`, `depth` fields.

**Step 2: Run lint**

```bash
cd web && npm run lint
```

**Step 3: Commit fixes if needed**

```bash
git add web/
git commit -m "fix: resolve frontend type errors for research source"
```

---

### Task 10: Backend Full Test

**Step 1: Run full backend test suite**

```bash
make test
```

**Step 2: Fix any failures**

**Step 3: Commit fixes if needed**

```bash
git commit -m "fix: resolve test failures from research source integration"
```
