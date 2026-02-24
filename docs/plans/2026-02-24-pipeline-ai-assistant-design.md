# Pipeline Settings AI Assistant — Design

## Overview

Add an LLM-assisted configuration chat to the pipeline settings panel, mirroring the existing workflow node `AIChatEditor`. Users describe their pipeline intent in natural language and the LLM configures sources, schedule, workflows, model, and editorial brief automatically.

## Approach

Extract a shared `ConfigureChat` UI component from the existing `AIChatEditor`, then build a `PipelineChatEditor` that reuses it with pipeline-specific request/response logic.

## Frontend

### 1. Shared Chat Component

**File**: `web/src/shared/ui/ConfigureChat.tsx` (new)

Extracted from `AIChatEditor` — contains only UI concerns:

- Model selector dropdown (fetches from `/api/models`)
- Thinking toggle (persisted to localStorage)
- Status display (last assistant message or loading spinner)
- Textarea input + send button
- Collapsible wrapper with "AI Assistant" header

```typescript
type ChatMessage = { role: 'user' | 'assistant'; content: string; isError?: boolean }

type ConfigureChatProps = {
  onSubmit: (params: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => Promise<{ explanation: string }>
  placeholder?: string
  loadingText?: string
}
```

The component manages its own state (messages, input, model, thinking, loading) and calls `onSubmit` when the user sends a message. On success, it displays the returned `explanation` as an assistant message. On error, it displays the error message.

### 2. AIChatEditor Refactoring

**File**: `web/src/features/edit-node/ui/AIChatEditor.tsx` (modified)

Refactored to use `ConfigureChat`. Domain logic stays here:

- Reads node data + upstream nodes
- Calls `configureNode()` API
- Applies response via `updateNodeConfig/Label/Description`
- Returns `{ explanation }` to ConfigureChat

### 3. PipelineChatEditor

**File**: `web/src/features/configure-pipeline/ui/PipelineChatEditor.tsx` (new)

```typescript
type PipelineChatEditorProps = {
  pipelineId: string
  currentSources: PipelineSource[]
  currentSchedule: string
  currentWorkflows: PipelineWorkflow[]
  currentModel: string
  currentContext?: PipelineContext
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (s: string) => void
  onWorkflowsChange: (w: PipelineWorkflow[]) => void
  onModelChange: (m: string) => void
  onContextSave: (ctx: PipelineContext) => Promise<void>
}
```

On submit: calls `POST /api/pipelines/{id}/configure`, then applies each non-null field from the response via the corresponding `onChange` callback. This triggers the existing auto-save debounce in `PipelineDetailPage`.

### 4. PipelineSettingsPanel Layout

**File**: `web/src/pages/pipelines/PipelineDetail.tsx` (modified)

```
<div className="flex flex-col h-full">
  <div className="...">Settings header + auto-save</div>
  <div className="flex-1 overflow-y-auto p-4 space-y-4">
    ... existing settings sections ...
  </div>
  <div className="border-t border-border bg-background/80 backdrop-blur-sm">
    <PipelineChatEditor ... />
  </div>
</div>
```

### 5. Types

**File**: `web/src/shared/types/index.ts` (additions)

```typescript
export type ConfigurePipelineRequest = {
  message: string
  model?: string
  thinking?: boolean
  history?: { role: string; content: string }[]
  current_sources: PipelineSource[]
  current_schedule: string
  current_workflows: PipelineWorkflow[]
  current_model: string
  current_context?: PipelineContext
}

export type ConfigurePipelineResponse = {
  sources?: PipelineSource[]
  schedule?: string
  workflows?: PipelineWorkflow[]
  model?: string
  context?: PipelineContext
  name?: string
  description?: string
  explanation: string
}
```

## Backend

### 1. Handler

**File**: `internal/api/configure_pipeline.go` (new)

`POST /api/pipelines/{id}/configure`

Flow mirrors `configureNode` (internal/api/configure.go):

1. Decode request, validate `message` required
2. Resolve LLM model (requested or default)
3. Build context: serialize current pipeline settings as JSON
4. Assemble system prompt: `s.skills.GetPrompt("pipeline-configure")` + available models list (same pattern as configure.go:118-145)
5. Build LLM request with chat history + context message
6. Call `llm.GenerateContent()`, parse JSON response
7. Return `ConfigurePipelineResponse`

**Key difference from node configure**: No sub-skill lookup (`Get()`). Pipeline uses a single `GetPrompt("pipeline-configure")` skill that contains all source types, cron syntax, and brief field guidance inline.

### 2. Route Registration

**File**: `internal/api/server.go` (modified)

```go
r.Post("/pipelines/{id}/configure", s.configurePipeline)
```

### 3. Skill

**File**: `internal/skills/prompts/pipeline-configure.md` (new)

Loaded via `GetPrompt("pipeline-configure")`. No YAML frontmatter required (filename used as key). Contents:

- Pipeline configuration specialist persona
- Source types and their config fields:
  - RSS: url, limit
  - Reddit: subreddit, min_score, limit
  - HN: min_score, limit
  - Google Trends: keywords, limit
  - Twitter/X: keywords, limit
  - HTTP: url, limit
- Source type classification: `static` (RSS, HTTP) vs `signal` (HN, Reddit, Google Trends, Twitter)
- Schedule cron syntax with common presets
- Workflow selection: use only from available workflows list
- Editorial brief fields: purpose, target_audience, tone_style, focus_keywords, exclude_keywords, language
- Model selection: pick from injected available models list
- Output format: JSON with only changed fields + explanation
- All user-facing text in Korean

## Node Configure vs Pipeline Configure

| Aspect | Node Configure | Pipeline Configure |
|--------|---------------|-------------------|
| Base prompt | `GetPrompt("node-configure")` | `GetPrompt("pipeline-configure")` |
| Sub-skill | `Get(nodeType + "-node")` per type | None — single skill covers all |
| Model list injection | Yes (configure.go:118-145) | Yes — same pattern |
| Context message | Node config JSON + upstream nodes | Pipeline full settings JSON |
| Response format | `{ config, label, description, explanation }` | `{ sources?, schedule?, workflows?, model?, context?, name?, description?, explanation }` |
| Response apply | config merge + label/desc overwrite | Non-null fields → corresponding setter |

## Files Changed

| File | Action |
|------|--------|
| `web/src/shared/ui/ConfigureChat.tsx` | New — shared chat UI component |
| `web/src/features/edit-node/ui/AIChatEditor.tsx` | Refactor — use ConfigureChat |
| `web/src/features/configure-pipeline/ui/PipelineChatEditor.tsx` | New — pipeline-specific logic |
| `web/src/features/configure-pipeline/api/index.ts` | New — API function |
| `web/src/shared/types/index.ts` | Add — request/response types |
| `web/src/pages/pipelines/PipelineDetail.tsx` | Modify — add PipelineChatEditor to settings panel bottom |
| `internal/api/configure_pipeline.go` | New — backend handler |
| `internal/api/server.go` | Modify — route registration |
| `internal/skills/prompts/pipeline-configure.md` | New — LLM skill prompt |
