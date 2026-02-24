# Pipeline Settings AI Assistant — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an LLM chat to the pipeline settings panel that configures sources, schedule, workflows, model, and editorial brief from natural language.

**Architecture:** Extract shared `ConfigureChat` UI from existing `AIChatEditor`, create `PipelineChatEditor` with pipeline-specific API/apply logic, add `POST /pipelines/{id}/configure` backend handler with `pipeline-configure` skill.

**Tech Stack:** React 19, TypeScript, Go 1.24, Chi router, Google ADK, genai

---

### Task 1: Backend — Create pipeline-configure skill

**Files:**
- Create: `internal/skills/prompts/pipeline-configure.md`

**Step 1: Write the skill file**

```markdown
You are a pipeline configuration specialist for the Upal visual AI workflow platform. Your expertise includes content sourcing strategy, scheduling optimization, editorial workflow design, and translating vague user intent into precise pipeline settings. You understand how content pipelines collect, analyze, and produce content through configurable stages.

When the user describes what their pipeline should do — even briefly — you MUST produce comprehensive, production-ready settings. Infer and set every field that makes sense: data sources with proper URLs and parameters, cron schedules, editorial brief with rich context, and appropriate analysis models.

---

## Pipeline Settings Schema

### Sources
Array of data sources. Each source has:
```json
{
  "id": "unique-string",
  "type": "rss|hn|reddit|google_trends|twitter|http",
  "source_type": "static|signal",
  "label": "한국어 표시 이름",
  "url": "...",
  "subreddit": "...",
  "min_score": 10,
  "keywords": ["keyword1"],
  "limit": 20
}
```

Source type classification:
- `static`: RSS, HTTP — fetch content directly from URL
- `signal`: HN, Reddit, Google Trends, Twitter — discover trending/popular content

Type-specific required fields:
- **rss**: `url` (feed URL), `limit` (max items, default 20)
- **hn**: `min_score` (minimum points, default 10), `limit`
- **reddit**: `subreddit` (name without r/), `min_score`, `limit`
- **google_trends**: `keywords` (array of search terms), `limit`
- **twitter**: `keywords` (array of search terms), `limit`
- **http**: `url` (endpoint URL), `limit`

Source `id` format: `"src-{type}-{index}"` (e.g. `"src-rss-1"`, `"src-hn-1"`).

### Schedule
Standard 5-field cron expression. Common presets:
- `"0 * * * *"` — Every hour
- `"0 */6 * * *"` — Every 6 hours
- `"0 */12 * * *"` — Every 12 hours
- `"0 9 * * *"` — Daily at 09:00
- `"0 9 * * 1-5"` — Weekdays at 09:00
- `"0 9 * * 1"` — Weekly Monday 09:00
- `"0 9 1 * *"` — Monthly 1st 09:00

### Workflows
Array of workflow references:
```json
{ "workflow_name": "exact-slug", "label": "표시 이름" }
```
ONLY use workflows from the "Available workflows" list. If no workflows are listed, omit this field.

### Model
Analysis model in `"provider/model"` format. Pick from the "Available models" list. Leave empty string `""` for system default.

### Editorial Brief (context)
```json
{
  "purpose": "파이프라인의 목적",
  "target_audience": "대상 독자",
  "tone_style": "어조와 스타일",
  "focus_keywords": ["키워드1", "키워드2"],
  "exclude_keywords": ["제외어1"],
  "language": "ko|en|ja|zh"
}
```

---

## Rules

1. **Partial updates**: Only include fields that the user's request affects. If the user only asks about sources, only return `sources`. If they describe a full pipeline, return all relevant fields.
2. **Source IDs**: Always generate unique `id` values for new sources using `"src-{type}-{index}"` format.
3. **Source type**: Always set `source_type` correctly — `"static"` for RSS/HTTP, `"signal"` for HN/Reddit/Google Trends/Twitter.
4. **Workflow names**: ONLY reference workflows from the "Available workflows" list. NEVER invent workflow names.
5. **Model selection**: ONLY use models from the "Available models" list. Match model capability to the pipeline's analysis needs.
6. **Language**: ALL user-facing text (labels, purpose, descriptions, explanation) MUST be in Korean (한국어).
7. **Explanation**: Always include a clear Korean explanation summarizing what was changed and why.

---

## Output Format

```json
{
  "sources": [...],
  "schedule": "cron expression",
  "workflows": [...],
  "model": "provider/model",
  "context": { ... },
  "name": "new-pipeline-name",
  "description": "새 설명",
  "explanation": "변경사항 요약"
}
```

Only include fields that were changed. `explanation` is always required.

Return ONLY valid JSON, no markdown fences, no extra text.
```

**Step 2: Verify skill loads**

Run: `go test ./internal/skills/... -v -race -run TestRegistry`
Expected: PASS (new file auto-discovered from embedded prompts/ directory)

**Step 3: Commit**

```bash
git add internal/skills/prompts/pipeline-configure.md
git commit -m "feat(skills): add pipeline-configure prompt for AI-assisted pipeline settings"
```

---

### Task 2: Backend — Create configure_pipeline handler

**Files:**
- Create: `internal/api/configure_pipeline.go`
- Modify: `internal/api/server.go:109` (add route inside pipelines block)

**Step 1: Write the handler**

Create `internal/api/configure_pipeline.go`:

```go
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

type ConfigurePipelineRequest struct {
	Message          string           `json:"message"`
	Model            string           `json:"model,omitempty"`
	Thinking         bool             `json:"thinking,omitempty"`
	History          []ConfigChatMsg  `json:"history,omitempty"`
	CurrentSources   json.RawMessage  `json:"current_sources"`
	CurrentSchedule  string           `json:"current_schedule"`
	CurrentWorkflows json.RawMessage  `json:"current_workflows"`
	CurrentModel     string           `json:"current_model"`
	CurrentContext   json.RawMessage  `json:"current_context,omitempty"`
}

type ConfigurePipelineResponse struct {
	Sources     json.RawMessage `json:"sources,omitempty"`
	Schedule    *string         `json:"schedule,omitempty"`
	Workflows   json.RawMessage `json:"workflows,omitempty"`
	Model       *string         `json:"model,omitempty"`
	Context     json.RawMessage `json:"context,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Explanation string          `json:"explanation"`
}

func (s *Server) configurePipeline(w http.ResponseWriter, r *http.Request) {
	var req ConfigurePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	// Resolve LLM model
	llm := s.generator.LLM()
	modelName := s.generator.Model()
	if req.Model != "" {
		if resolved, ok := s.resolveModel(req.Model); ok {
			llm = resolved.llm
			modelName = resolved.model
		}
	}

	// Build context message with current pipeline settings
	contextMsg := fmt.Sprintf(
		"Current pipeline settings:\nSources: %s\nSchedule: %q\nWorkflows: %s\nModel: %q\nEditorial brief: %s\n\nUser request: %s",
		string(req.CurrentSources),
		req.CurrentSchedule,
		string(req.CurrentWorkflows),
		req.CurrentModel,
		string(req.CurrentContext),
		req.Message,
	)

	// Build contents: history + current message
	var contents []*genai.Content
	for _, h := range req.History {
		switch h.Role {
		case "user":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleUser))
		case "assistant":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleModel))
		}
	}
	contents = append(contents, genai.NewContentFromText(contextMsg, genai.RoleUser))

	// Build system prompt from skill
	sysPrompt := ""
	if s.skills != nil {
		sysPrompt = s.skills.GetPrompt("pipeline-configure")
	}

	// Inject available models (same pattern as configureNode)
	if allModels := KnownModelsGrouped(s.providerConfigs); len(allModels) > 0 {
		sysPrompt += fmt.Sprintf("\n\nAvailable models (use in \"model\" field):\nDefault model: %q\n", modelName)
		var textModels, imageModels []ModelInfo
		for _, m := range allModels {
			switch m.Category {
			case ModelCategoryText:
				textModels = append(textModels, m)
			case ModelCategoryImage:
				imageModels = append(imageModels, m)
			}
		}
		if len(textModels) > 0 {
			sysPrompt += "\nText/reasoning models:\n"
			for _, m := range textModels {
				sysPrompt += fmt.Sprintf("- %q [%s] — %s\n", m.ID, m.Tier, m.Hint)
			}
		}
		if len(imageModels) > 0 {
			sysPrompt += "\nImage generation models:\n"
			for _, m := range imageModels {
				sysPrompt += fmt.Sprintf("- %q — %s\n", m.ID, m.Hint)
			}
		}
	}

	// Inject available workflows
	if s.repo != nil {
		if wfs, err := s.repo.List(); err == nil && len(wfs) > 0 {
			sysPrompt += "\n\nAvailable workflows:\n"
			for _, wf := range wfs {
				label := wf.Label
				if label == "" {
					label = wf.Name
				}
				sysPrompt += fmt.Sprintf("- %q (%s)\n", wf.Name, label)
			}
		}
	}

	llmReq := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: contents,
	}

	ctx := upalmodel.WithThinking(r.Context(), req.Thinking)

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			http.Error(w, fmt.Sprintf("LLM call failed: %v", err), http.StatusInternalServerError)
			return
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		http.Error(w, "empty response from LLM", http.StatusInternalServerError)
		return
	}

	text := llmutil.ExtractText(resp)
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, text), http.StatusInternalServerError)
		return
	}

	var configResp ConfigurePipelineResponse
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&configResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}
```

**Step 2: Register the route**

In `internal/api/server.go`, inside the pipelines route block (after line 109 `r.Post("/{id}/collect", s.collectPipeline)`), add:

```go
r.Post("/{id}/configure", s.configurePipeline)
```

**Step 3: Verify build**

Run: `go build ./cmd/upal`
Expected: Compiles with no errors

**Step 4: Commit**

```bash
git add internal/api/configure_pipeline.go internal/api/server.go
git commit -m "feat(api): add POST /pipelines/{id}/configure endpoint for AI-assisted pipeline settings"
```

---

### Task 3: Frontend — Create shared ConfigureChat component

**Files:**
- Create: `web/src/shared/ui/ConfigureChat.tsx`

**Step 1: Write the component**

Extract the pure UI from `web/src/features/edit-node/ui/AIChatEditor.tsx` (lines 1-249) into a generic component. The new component manages its own state (messages, input, model, thinking, loading) and delegates the actual API call + response application via an `onSubmit` prop.

```typescript
import { useState, useRef, useEffect } from 'react'
import { listModels } from '@/shared/api'
import type { ModelInfo } from '@/shared/types'
import { Button } from '@/shared/ui/button'
import { Textarea } from '@/shared/ui/textarea'
import {
  Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue,
} from '@/shared/ui/select'
import {
  Collapsible, CollapsibleContent, CollapsibleTrigger,
} from '@/shared/ui/collapsible'
import { Sparkles, SendHorizontal, Loader2, ChevronDown, Check, AlertCircle, BrainCircuit } from 'lucide-react'
import { cn, groupModelsByProvider } from '@/shared/lib/utils'

export type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
}

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

export function ConfigureChat({ onSubmit, placeholder = 'Describe what you want...', loadingText = 'Configuring...' }: ConfigureChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(true)
  const [models, setModels] = useState<ModelInfo[]>([])
  const [selectedModel, setSelectedModel] = useState(
    () => localStorage.getItem('upal:ai-model') ?? '',
  )
  const [thinking, setThinking] = useState(
    () => localStorage.getItem('upal:ai-thinking') === 'true',
  )

  const textareaRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => { localStorage.setItem('upal:ai-model', selectedModel) }, [selectedModel])
  useEffect(() => { localStorage.setItem('upal:ai-thinking', String(thinking)) }, [thinking])
  useEffect(() => { listModels().then(setModels).catch(() => {}) }, [])

  const lastStatus = [...messages].reverse().find((m) => m.role === 'assistant') ?? null

  const handleSubmit = async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)

    try {
      const history = messages.map((m) => ({ role: m.role, content: m.content }))
      const result = await onSubmit({ message: trimmed, model: selectedModel, thinking, history })
      setMessages((prev) => [...prev, { role: 'assistant', content: result.explanation }])
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'An unknown error occurred'
      setMessages((prev) => [...prev, { role: 'assistant', content: errorMsg, isError: true }])
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger asChild>
        <button className="flex w-full items-center justify-between px-2 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">
          <span className="flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5" />
            AI Assistant
          </span>
          <ChevronDown className={cn('h-3.5 w-3.5 transition-transform', isOpen && 'rotate-180')} />
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="px-2 pb-2 space-y-2">
          {/* Model selector + thinking toggle */}
          <div className="flex items-center gap-0.5">
            {models.length > 0 && (
              <Select value={selectedModel} onValueChange={setSelectedModel}>
                <SelectTrigger className="h-5 text-[10px] w-fit border-none shadow-none px-1.5 py-0 gap-1 text-muted-foreground hover:text-foreground">
                  <SelectValue placeholder="Auto" />
                </SelectTrigger>
                <SelectContent position="popper" side="top" sideOffset={4}>
                  {Object.entries(groupModelsByProvider(models)).map(([provider, items]) => (
                    <SelectGroup key={provider}>
                      <SelectLabel className="text-[10px]">{provider}</SelectLabel>
                      {items.map((m) => (
                        <SelectItem key={m.id} value={m.id} className="text-[11px]">{m.name}</SelectItem>
                      ))}
                    </SelectGroup>
                  ))}
                </SelectContent>
              </Select>
            )}
            <button
              type="button"
              onClick={() => setThinking((v) => !v)}
              className={cn(
                'flex items-center gap-0.5 rounded px-1 py-0.5 text-[10px] transition-colors',
                thinking ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:text-foreground',
              )}
              title={thinking ? 'Thinking enabled' : 'Thinking disabled'}
            >
              <BrainCircuit className="h-3 w-3" />
            </button>
          </div>

          {/* Status */}
          {isLoading && (
            <div className="flex items-center gap-1.5 text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span className="text-[10px]">{loadingText}</span>
            </div>
          )}
          {!isLoading && lastStatus && (
            <div className={cn('flex items-start gap-1.5 text-[10px]', lastStatus.isError ? 'text-destructive' : 'text-muted-foreground')}>
              {lastStatus.isError ? <AlertCircle className="h-3 w-3 mt-px shrink-0" /> : <Check className="h-3 w-3 mt-px shrink-0" />}
              <span className="truncate">{lastStatus.content}</span>
            </div>
          )}

          {/* Input */}
          <div className="flex items-end gap-1">
            <Textarea
              ref={textareaRef}
              className="min-h-[32px] max-h-[60px] resize-none text-[11px]"
              rows={1}
              placeholder={placeholder}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={isLoading}
            />
            <Button size="icon" variant="ghost" className="h-7 w-7 shrink-0" onClick={handleSubmit} disabled={!input.trim() || isLoading}>
              {isLoading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <SendHorizontal className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 3: Commit**

```bash
git add web/src/shared/ui/ConfigureChat.tsx
git commit -m "feat(ui): extract ConfigureChat shared component from AIChatEditor"
```

---

### Task 4: Frontend — Refactor AIChatEditor to use ConfigureChat

**Files:**
- Modify: `web/src/features/edit-node/ui/AIChatEditor.tsx`

**Step 1: Rewrite AIChatEditor**

Replace the entire file content with:

```typescript
import { useWorkflowStore } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { configureNode } from '../api'
import { useUpstreamNodes } from '../model/useUpstreamNodes'
import { ConfigureChat } from '@/shared/ui/ConfigureChat'
import type { ChatMessage } from '@/shared/ui/ConfigureChat'

type AIChatEditorProps = {
  nodeId: string
  data: NodeData
}

export function AIChatEditor({ nodeId, data }: AIChatEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)
  const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)
  const upstreamNodes = useUpstreamNodes(nodeId)

  const handleSubmit = async ({ message, model, thinking, history }: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => {
    const response = await configureNode({
      node_type: data.nodeType,
      node_id: nodeId,
      current_config: data.config,
      label: data.label,
      description: data.description ?? '',
      message,
      model: model || undefined,
      thinking,
      history,
      upstream_nodes: upstreamNodes,
    })

    if (response.config && Object.keys(response.config).length > 0) {
      updateNodeConfig(nodeId, response.config)
    }
    if (response.label) {
      updateNodeLabel(nodeId, response.label)
    }
    const desc = response.description || (response.config?.description as string)
    if (desc) {
      updateNodeDescription(nodeId, desc)
    }

    return { explanation: response.explanation || 'Configuration updated.' }
  }

  return (
    <ConfigureChat
      onSubmit={handleSubmit}
      placeholder="Describe this node..."
      loadingText="Configuring..."
    />
  )
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 3: Verify workflow editor still works**

Open `http://localhost:5173`, create/open a workflow, select a node, and confirm the AI Assistant appears at the bottom of the right panel. Type a test message and confirm it responds.

**Step 4: Commit**

```bash
git add web/src/features/edit-node/ui/AIChatEditor.tsx
git commit -m "refactor(edit-node): use shared ConfigureChat in AIChatEditor"
```

---

### Task 5: Frontend — Add types and API function for pipeline configure

**Files:**
- Modify: `web/src/shared/types/index.ts:63` (add after ConfigureNodeResponse)
- Create: `web/src/features/configure-pipeline/api/index.ts`

**Step 1: Add TypeScript types**

In `web/src/shared/types/index.ts`, after line 63 (after `ConfigureNodeResponse`), add:

```typescript

// --- AI Pipeline Configuration ---

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

**Step 2: Create API function**

Create `web/src/features/configure-pipeline/api/index.ts`:

```typescript
import { API_BASE, apiFetch } from '@/shared/api/client'
import type { ConfigurePipelineRequest, ConfigurePipelineResponse } from '@/shared/types'

export async function configurePipeline(
  pipelineId: string,
  req: ConfigurePipelineRequest,
): Promise<ConfigurePipelineResponse> {
  return apiFetch<ConfigurePipelineResponse>(
    `${API_BASE}/pipelines/${encodeURIComponent(pipelineId)}/configure`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    },
  )
}
```

**Step 3: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/shared/types/index.ts web/src/features/configure-pipeline/api/index.ts
git commit -m "feat(pipeline): add ConfigurePipeline types and API function"
```

---

### Task 6: Frontend — Create PipelineChatEditor and wire into settings panel

**Files:**
- Create: `web/src/features/configure-pipeline/ui/PipelineChatEditor.tsx`
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx:35-269` (PipelineSettingsPanel)

**Step 1: Write PipelineChatEditor**

Create `web/src/features/configure-pipeline/ui/PipelineChatEditor.tsx`:

```typescript
import { ConfigureChat } from '@/shared/ui/ConfigureChat'
import type { ChatMessage } from '@/shared/ui/ConfigureChat'
import { configurePipeline } from '../api'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/shared/types'

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

export function PipelineChatEditor({
  pipelineId,
  currentSources, currentSchedule, currentWorkflows, currentModel, currentContext,
  onSourcesChange, onScheduleChange, onWorkflowsChange, onModelChange, onContextSave,
}: PipelineChatEditorProps) {
  const handleSubmit = async ({ message, model, thinking, history }: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => {
    const response = await configurePipeline(pipelineId, {
      message,
      model: model || undefined,
      thinking,
      history,
      current_sources: currentSources,
      current_schedule: currentSchedule,
      current_workflows: currentWorkflows,
      current_model: currentModel,
      current_context: currentContext,
    })

    // Apply non-null fields
    if (response.sources) onSourcesChange(response.sources)
    if (response.schedule !== undefined && response.schedule !== null) onScheduleChange(response.schedule)
    if (response.workflows) onWorkflowsChange(response.workflows)
    if (response.model !== undefined && response.model !== null) onModelChange(response.model)
    if (response.context) await onContextSave(response.context)

    return { explanation: response.explanation || 'Settings updated.' }
  }

  return (
    <ConfigureChat
      onSubmit={handleSubmit}
      placeholder="파이프라인 설정을 설명하세요..."
      loadingText="설정 중..."
    />
  )
}
```

**Step 2: Add PipelineChatEditor to PipelineSettingsPanel**

In `web/src/pages/pipelines/PipelineDetail.tsx`, modify `PipelineSettingsPanel`:

1. Add import at the top of file:
```typescript
import { PipelineChatEditor } from '@/features/configure-pipeline/ui/PipelineChatEditor'
```

2. Add `pipelineId` to the component's props type (line 37-49):
```typescript
function PipelineSettingsPanel({
  pipelineId,
  sources, schedule, context, workflows, model,
  onSourcesChange, onScheduleChange, onContextSave, onWorkflowsChange, onModelChange, autoSaveStatus,
}: {
  pipelineId: string
  sources: PipelineSource[]
  // ... rest unchanged
```

3. Change the outer `<div>` structure (line 59) to pin the chat at the bottom. Replace:
```tsx
<div className="flex flex-col h-full overflow-y-auto">
```
with:
```tsx
<div className="flex flex-col h-full">
```

4. After the closing `</div>` of the settings sections container (after line 258, after `</section>` for editorial brief), close the scrollable area and add the chat:
```tsx
      </div>  {/* end flex-1 overflow-y-auto */}

      {/* AI Assistant — pinned to bottom */}
      <div className="border-t border-border bg-background/80 backdrop-blur-sm">
        <PipelineChatEditor
          pipelineId={pipelineId}
          currentSources={sources}
          currentSchedule={schedule}
          currentWorkflows={workflows}
          currentModel={model}
          currentContext={context}
          onSourcesChange={onSourcesChange}
          onScheduleChange={onScheduleChange}
          onWorkflowsChange={onWorkflowsChange}
          onModelChange={onModelChange}
          onContextSave={onContextSave}
        />
      </div>

      {showAddModal && (
```

5. Pass `pipelineId` when rendering `PipelineSettingsPanel` (around line 577):
```tsx
<PipelineSettingsPanel
  pipelineId={id!}
  sources={localSources}
  // ... rest unchanged
```

**Step 3: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 4: Manual verification**

Open `http://localhost:5173/pipelines/{id}`. Verify:
1. Settings panel shows "AI Assistant" collapsible section pinned at bottom
2. Settings sections above are scrollable independently
3. Type "매일 오전 9시에 AI 뉴스를 수집하고 분석해줘" and submit
4. Confirm sources, schedule, and brief are auto-populated
5. Auto-save triggers and "Saved" indicator appears

**Step 5: Commit**

```bash
git add web/src/features/configure-pipeline/ui/PipelineChatEditor.tsx web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline): add AI assistant chat to pipeline settings panel"
```

---

### Task 7: Verify end-to-end and final commit

**Step 1: Run backend tests**

Run: `make test`
Expected: All tests pass

**Step 2: Run frontend type-check**

Run: `make test-frontend`
Expected: No errors

**Step 3: Run frontend lint**

Run: `cd web && npm run lint`
Expected: No errors (or only pre-existing warnings)

**Step 4: Manual E2E test**

1. Start dev: `make dev-backend` + `make dev-frontend`
2. Open pipeline detail page
3. Test AI assistant with various prompts:
   - "매일 오전 9시에 AI 뉴스 RSS 수집해줘" → sources + schedule set
   - "스케줄을 매주 월요일로 변경" → only schedule changes
   - "한국어 기술 블로그 독자를 대상으로 브리프 작성해줘" → only context set
4. Confirm existing workflow node AIChatEditor still works after refactoring
