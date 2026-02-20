# Code Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove ~600 lines of duplication and decompose oversized functions/components across both Go backend and React frontend.

**Architecture:** Two parallel tracks (Go backend / React frontend) with zero shared file overlap. Each track extracts utilities first, then decomposes large functions/components. API contract (event shapes, endpoints) is immutable.

**Tech Stack:** Go 1.24, ADK, Chi router | React 19, Zustand, TipTap, Tailwind v4, shadcn/ui

---

## Track 1: Go Backend

### Task 1.1: Extract `stripMarkdownJSON()` utility

Identical JSON-from-markdown parsing appears in `configure.go:169-179` and `generate.go:146-158`. Extract to shared utility.

**Files:**
- Create: `internal/llmutil/parse.go`
- Create: `internal/llmutil/parse_test.go`
- Modify: `internal/api/configure.go:169-179`
- Modify: `internal/generate/generate.go:146-164`

**Step 1: Write the failing test**

```go
// internal/llmutil/parse_test.go
package llmutil

import (
	"testing"
)

func TestStripMarkdownJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "clean JSON",
			input: `{"name":"test"}`,
			want:  `{"name":"test"}`,
		},
		{
			name:  "json fenced",
			input: "```json\n{\"name\":\"test\"}\n```",
			want:  `{"name":"test"}`,
		},
		{
			name:  "leading text before JSON",
			input: "Here is your config:\n{\"name\":\"test\"}",
			want:  `{"name":"test"}`,
		},
		{
			name:  "generic fenced",
			input: "```\n{\"name\":\"test\"}\n```",
			want:  `{"name":"test"}`,
		},
		{
			name:    "no JSON at all",
			input:   "no json here",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StripMarkdownJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/llmutil/... -v -race -run TestStripMarkdownJSON`
Expected: FAIL — package does not exist

**Step 3: Write minimal implementation**

```go
// internal/llmutil/parse.go
package llmutil

import (
	"fmt"
	"strings"
)

// StripMarkdownJSON removes markdown code fences and leading text from an LLM
// response, returning the raw JSON string. It finds the first '{' and returns
// from there, trimming any trailing fences.
func StripMarkdownJSON(text string) (string, error) {
	s := strings.TrimSpace(text)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	// Skip to the first '{' in case there is leading text.
	if start := strings.Index(s, "{"); start > 0 {
		s = s[start:]
	}

	if !strings.HasPrefix(s, "{") {
		return "", fmt.Errorf("no JSON object found in text")
	}

	return s, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/llmutil/... -v -race -run TestStripMarkdownJSON`
Expected: PASS

**Step 5: Replace usages in configure.go**

In `internal/api/configure.go`, replace lines 169-183 with:

```go
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, text), http.StatusInternalServerError)
		return
	}

	var configResp ConfigureNodeResponse
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&configResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}
```

Add `"github.com/soochol/upal/internal/llmutil"` to imports.

**Step 6: Replace usages in generate.go**

In `internal/generate/generate.go`, replace lines 146-163 with:

```go
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parse generated workflow (model output may be malformed): %w\nraw output: %s", err, text)
	}

	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&wf); err != nil {
		return nil, fmt.Errorf("parse generated workflow (model output may be malformed): %w\nraw output: %s", err, content)
	}
```

Add `"github.com/soochol/upal/internal/llmutil"` to imports. Remove unused `strings` import if applicable.

**Step 7: Run all tests**

Run: `make test`
Expected: All PASS

**Step 8: Commit**

```bash
git add internal/llmutil/ internal/api/configure.go internal/generate/generate.go
git commit -m "refactor: extract stripMarkdownJSON utility to eliminate duplication"
```

---

### Task 1.2: Extract `resolveLLM()` helper in builders.go

The LLM resolution pattern (parse "provider/model", lookup in map, wrap with namedLLM) appears 3 times: `builders.go:210-218`, `builders.go:281-289`, `builders.go:404-418`.

**Files:**
- Modify: `internal/agents/builders.go`

**Step 1: Add the helper function at package level**

Add before `buildLLMAgent`:

```go
// resolveLLM resolves a "provider/model" format ID into an LLM and model name.
// If modelID is empty or not found, it falls back to the first available LLM.
func resolveLLM(modelID string, llms map[string]adkmodel.LLM) (adkmodel.LLM, string) {
	if modelID != "" {
		parts := strings.SplitN(modelID, "/", 2)
		if len(parts) == 2 {
			if l, ok := llms[parts[0]]; ok {
				return &namedLLM{LLM: l, name: parts[1]}, parts[1]
			}
		}
	}
	// Fallback: first available LLM
	for _, l := range llms {
		return l, ""
	}
	return nil, ""
}
```

**Step 2: Replace in `generateAutoLayout` (lines 206-226)**

Replace lines 206-226 with:

```go
	llm, modelName := resolveLLM(layoutModel, llms)
	if llm == nil {
		return "", fmt.Errorf("no LLM available for auto-layout")
	}
```

**Step 3: Replace in `generateManualLayout` (lines 277-301)**

Replace lines 277-301 with:

```go
	llm, modelName := resolveLLM(layoutModel, llms)
	if llm == nil {
		return "", fmt.Errorf("no LLM available for manual layout")
	}
```

**Step 4: Replace in `buildLLMAgent` (lines 405-418)**

Replace lines 405-418 with:

```go
	var llm adkmodel.LLM
	var modelName string
	if llms != nil {
		llm, modelName = resolveLLM(modelID, llms)
	}
```

(Keep the existing nil check at line 420-422 as-is — buildLLMAgent requires a non-nil LLM unlike layout functions.)

**Step 5: Run all tests**

Run: `go test ./internal/agents/... -v -race`
Expected: All PASS

**Step 6: Commit**

```bash
git add internal/agents/builders.go
git commit -m "refactor: extract resolveLLM helper, eliminate 3x duplicate resolution"
```

---

### Task 1.3: Consolidate `generateAutoLayout` + `generateManualLayout`

These two functions (lines 204-272 and 274-340) are 95% identical — they differ only in the system prompt and user message content.

**Files:**
- Modify: `internal/agents/builders.go`

**Step 1: Create unified `generateLayout` function**

Replace both `generateAutoLayout` and `generateManualLayout` with:

```go
// generateLayout calls an LLM with a system prompt and user content to produce HTML.
func generateLayout(ctx agent.InvocationContext, userContent string, systemPrompt string, layoutModel string, llms map[string]adkmodel.LLM) (string, error) {
	llm, modelName := resolveLLM(layoutModel, llms)
	if llm == nil {
		return "", fmt.Errorf("no LLM available for layout generation")
	}

	req := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userContent, genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("layout LLM call: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	var text string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			text += p.Text
		}
	}

	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```html")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	return text, nil
}
```

**Step 2: Update callers in `buildOutputAgent`**

Replace line 171:
```go
if html, err := generateManualLayout(ctx, resolvedPrompt, layoutModel, llms); err == nil && html != "" {
```
with:
```go
if html, err := generateLayout(ctx, resolvedPrompt, manualLayoutSystemPrompt, layoutModel, llms); err == nil && html != "" {
```

Replace line 179:
```go
if html, err := generateAutoLayout(ctx, result, layoutModel, llms); err == nil && html != "" {
```
with:
```go
autoContent := fmt.Sprintf("Create a styled HTML page presenting the following content:\n\n%s", result)
if html, err := generateLayout(ctx, autoContent, autoLayoutSystemPrompt, layoutModel, llms); err == nil && html != "" {
```

**Step 3: Delete the old `generateAutoLayout` and `generateManualLayout` functions**

Remove lines 204-340 entirely (both functions).

**Step 4: Run all tests**

Run: `go test ./internal/agents/... -v -race`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/agents/builders.go
git commit -m "refactor: consolidate generateAutoLayout + generateManualLayout into single generateLayout"
```

---

### Task 1.4: Extract `extractResponseText()` helper

The pattern of extracting text from `*adkmodel.LLMResponse` appears in `builders.go` (lines 258-262, 326-330, 501-506), `configure.go` (lines 162-167), and `generate.go` (lines 139-144). Extract once.

**Files:**
- Create: `internal/llmutil/response.go`
- Modify: `internal/agents/builders.go`
- Modify: `internal/api/configure.go`
- Modify: `internal/generate/generate.go`

**Step 1: Write utility**

```go
// internal/llmutil/response.go
package llmutil

import (
	adkmodel "google.golang.org/adk/model"
)

// ExtractText joins all text parts from an LLM response.
func ExtractText(resp *adkmodel.LLMResponse) string {
	if resp == nil || resp.Content == nil {
		return ""
	}
	var text string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			text += p.Text
		}
	}
	return text
}
```

**Step 2: Replace all usages**

In each file, replace the loop pattern with `text := llmutil.ExtractText(resp)`.

**Step 3: Run all tests**

Run: `make test`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/llmutil/response.go internal/agents/builders.go internal/api/configure.go internal/generate/generate.go
git commit -m "refactor: extract extractResponseText to llmutil package"
```

---

### Task 1.5: Extract SSE error helpers in run.go

The pattern `json.Marshal(map[string]string{"status":"failed","error":...})` + `Fprintf` + `Flush` repeats 4 times in `run.go` (lines 63-66, 76-79, 98-101, 108-111).

**Files:**
- Modify: `internal/api/run.go`

**Step 1: Add helper functions at package level**

```go
// sendSSEError writes an SSE "done" event with a failure status.
func sendSSEError(w http.ResponseWriter, flusher http.Flusher, err error) {
	data, _ := json.Marshal(map[string]string{"status": "failed", "error": err.Error()})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
}

// sendSSEDone writes an SSE "done" event with the given payload.
func sendSSEDone(w http.ResponseWriter, flusher http.Flusher, payload any) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
}
```

**Step 2: Replace all 4 error occurrences**

Replace each 3-line block with `sendSSEError(w, flusher, err); return`.

Replace the final done event (lines 137-139) with:
```go
sendSSEDone(w, flusher, map[string]any{"status": "completed", "session_id": sessionID, "state": finalState})
```

**Step 3: Run all tests**

Run: `go test ./internal/api/... -v -race`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/api/run.go
git commit -m "refactor: extract SSE error/done helpers in run.go"
```

---

### Task 1.6: Extract tool execution from `buildLLMAgent()`

The tool execution loop (lines 525-556 in buildLLMAgent) is deeply nested. Extract to a standalone function.

**Files:**
- Modify: `internal/agents/builders.go`

**Step 1: Extract function**

```go
// executeToolCalls runs each FunctionCall against the tool registry and returns
// a Content with FunctionResponse parts suitable for appending to the conversation.
func executeToolCalls(ctx agent.InvocationContext, calls []*genai.FunctionCall, upalTools map[string]tools.Tool) *genai.Content {
	var parts []*genai.Part
	for _, fc := range calls {
		var output map[string]any
		if t, ok := upalTools[fc.Name]; ok {
			result, err := t.Execute(ctx, fc.Args)
			if err != nil {
				output = map[string]any{"error": err.Error()}
			} else if m, ok := result.(map[string]any); ok {
				output = m
			} else {
				output = map[string]any{"result": fmt.Sprintf("%v", result)}
			}
		} else {
			output = map[string]any{"error": fmt.Sprintf("unknown tool %q", fc.Name)}
		}
		parts = append(parts, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     fc.Name,
				Response: output,
			},
		})
	}
	return &genai.Content{Role: genai.RoleUser, Parts: parts}
}
```

**Step 2: Replace the inline block in `buildLLMAgent`**

Replace lines 525-555 with:
```go
				contents = append(contents, resp.Content)
				contents = append(contents, executeToolCalls(ctx, toolCalls, upalTools))
```

**Step 3: Run all tests**

Run: `go test ./internal/agents/... -v -race`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/agents/builders.go
git commit -m "refactor: extract executeToolCalls from buildLLMAgent"
```

---

## Track 2: React Frontend

### Task 2.1: Extract `groupModelsByProvider()` utility

The `models.reduce<Record<string, ModelInfo[]>>(...)` pattern appears in `NodeEditor.tsx:40-43` and `AIChatEditor.tsx:183-187`.

**Files:**
- Modify: `web/src/lib/utils.ts`
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx:40-43`
- Modify: `web/src/components/panel/AIChatEditor.tsx:183-187`

**Step 1: Add utility to `lib/utils.ts`**

Append to existing `utils.ts`:

```typescript
import type { ModelInfo } from './api'

export function groupModelsByProvider(models: ModelInfo[]): Record<string, ModelInfo[]> {
  return models.reduce<Record<string, ModelInfo[]>>((acc, m) => {
    ;(acc[m.provider] ??= []).push(m)
    return acc
  }, {})
}
```

**Step 2: Replace in NodeEditor.tsx**

Replace lines 40-43 with:
```typescript
import { groupModelsByProvider } from '@/lib/utils'
// ...
const modelsByProvider = groupModelsByProvider(models)
```

**Step 3: Replace in AIChatEditor.tsx**

Replace the inline reduce (lines 183-187) with:
```typescript
import { groupModelsByProvider } from '@/lib/utils'
// ...
{Object.entries(groupModelsByProvider(models)).map(([provider, items]) => (
```

**Step 4: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/lib/utils.ts web/src/components/editor/nodes/NodeEditor.tsx web/src/components/panel/AIChatEditor.tsx
git commit -m "refactor: extract groupModelsByProvider utility"
```

---

### Task 2.2: Extract `<ModelSelector />` component

The model Select dropdown (with provider grouping, empty state) appears 3 times in `NodeEditor.tsx`: lines 87-111 (agent model), 188-212 (manual layout model), and 220-244 (auto layout model).

**Files:**
- Create: `web/src/components/editor/ModelSelector.tsx`
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`

**Step 1: Create the shared component**

```typescript
// web/src/components/editor/ModelSelector.tsx
import { useEffect, useState } from 'react'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { listModels, type ModelInfo } from '@/lib/api'
import { groupModelsByProvider } from '@/lib/utils'

type ModelSelectorProps = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
}

export function ModelSelector({ value, onChange, placeholder = 'Select a model...' }: ModelSelectorProps) {
  const [models, setModels] = useState<ModelInfo[]>([])

  useEffect(() => {
    listModels().then(setModels).catch(() => setModels([]))
  }, [])

  const modelsByProvider = groupModelsByProvider(models)

  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger className="h-7 text-xs w-full" size="sm">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        {Object.entries(modelsByProvider).map(([provider, providerModels]) => (
          <SelectGroup key={provider}>
            <SelectLabel>{provider}</SelectLabel>
            {providerModels.map((m) => (
              <SelectItem key={m.id} value={m.id} className="text-xs">
                {m.name}
              </SelectItem>
            ))}
          </SelectGroup>
        ))}
        {models.length === 0 && (
          <div className="px-2 py-4 text-xs text-muted-foreground text-center">
            No models available.<br />Configure providers in config.yaml
          </div>
        )}
      </SelectContent>
    </Select>
  )
}
```

**Step 2: Replace 3 usages in NodeEditor.tsx**

Replace each full `<Select>...</Select>` block for model selection with:
```tsx
<ModelSelector
  value={(config.model as string) ?? ''}
  onChange={(v) => setConfig('model', v)}
/>
```

(For layout models, use `config.layout_model` and key `'layout_model'` accordingly.)

Remove the `models` state, `useEffect(listModels)`, `modelsByProvider` reduce, and all Select-related imports from NodeEditor that are no longer needed.

**Step 3: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 4: Commit**

```bash
git add web/src/components/editor/ModelSelector.tsx web/src/components/editor/nodes/NodeEditor.tsx
git commit -m "refactor: extract ModelSelector component, eliminate 3x duplication in NodeEditor"
```

---

### Task 2.3: Split `NodeEditor` into node-type sub-components

After Task 2.2, NodeEditor still has a long if/else chain for each node type.

**Files:**
- Create: `web/src/components/editor/nodes/InputNodeEditor.tsx`
- Create: `web/src/components/editor/nodes/AgentNodeEditor.tsx`
- Create: `web/src/components/editor/nodes/ToolNodeEditor.tsx`
- Create: `web/src/components/editor/nodes/OutputNodeEditor.tsx`
- Create: `web/src/components/editor/nodes/ExternalNodeEditor.tsx`
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`

**Step 1: Define shared props type**

Each sub-component receives the same props:

```typescript
export type NodeEditorFieldProps = {
  nodeId: string
  config: Record<string, unknown>
  setConfig: (key: string, value: unknown) => void
}
```

**Step 2: Create each sub-component**

Extract the JSX from each `data.nodeType === 'xxx'` block into its own file. Example for `AgentNodeEditor.tsx`:

```typescript
import { Separator } from '@/components/ui/separator'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { ModelSelector } from '@/components/editor/ModelSelector'
import type { NodeEditorFieldProps } from './NodeEditor'

export function AgentNodeEditor({ nodeId, config, setConfig }: NodeEditorFieldProps) {
  return (
    <>
      <Separator />
      <div className="space-y-1">
        <Label className="text-xs">Model</Label>
        <ModelSelector
          value={(config.model as string) ?? ''}
          onChange={(v) => setConfig('model', v)}
        />
      </div>
      <div className="space-y-1">
        <Label htmlFor="node-max-turns" className="text-xs">Max Turns</Label>
        <Input
          id="node-max-turns"
          className="h-7 text-xs"
          type="number"
          min={1}
          value={(config.max_turns as number) ?? 1}
          onChange={(e) => setConfig('max_turns', parseInt(e.target.value) || 1)}
        />
      </div>
    </>
  )
}
```

Repeat for Input, Tool, Output, External — each extracting its respective JSX block.

**Step 3: Simplify NodeEditor to a router**

```typescript
const nodeEditors: Record<string, React.ComponentType<NodeEditorFieldProps>> = {
  input: InputNodeEditor,
  agent: AgentNodeEditor,
  tool: ToolNodeEditor,
  output: OutputNodeEditor,
  external: ExternalNodeEditor,
}

// Inside formContent:
const TypeEditor = nodeEditors[data.nodeType]
// ...
{TypeEditor && <TypeEditor nodeId={nodeId} config={config} setConfig={setConfig} />}
```

**Step 4: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/components/editor/nodes/
git commit -m "refactor: split NodeEditor into node-type sub-components"
```

---

### Task 2.4: Extract `useUpstreamNodes()` hook

Upstream node traversal appears in `PromptEditor.tsx:171-182` and `AIChatEditor.tsx:70-87`.

**Files:**
- Create: `web/src/hooks/useUpstreamNodes.ts`
- Modify: `web/src/components/editor/PromptEditor.tsx`
- Modify: `web/src/components/panel/AIChatEditor.tsx`

**Step 1: Create the hook**

```typescript
// web/src/hooks/useUpstreamNodes.ts
import { useMemo } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'

export type UpstreamNode = {
  id: string
  type: string
  label: string
}

export function useUpstreamNodes(nodeId: string): UpstreamNode[] {
  const edges = useWorkflowStore((s) => s.edges)
  const allNodes = useWorkflowStore((s) => s.nodes)

  return useMemo(() => {
    const sourceIds = new Set(
      edges.filter((e) => e.target === nodeId).map((e) => e.source),
    )
    return allNodes
      .filter((n) => sourceIds.has(n.id) && n.type !== 'groupNode')
      .map((n) => ({
        id: n.id,
        type: n.data.nodeType as string,
        label: n.data.label,
      }))
  }, [edges, allNodes, nodeId])
}
```

**Step 2: Replace in PromptEditor.tsx**

Replace lines 167-182 with:
```typescript
import { useUpstreamNodes } from '@/hooks/useUpstreamNodes'
// ...
const upstreamNodes = useUpstreamNodes(nodeId)
```

Remove the `edges`/`allNodes` subscriptions that are only used for upstream computation.

**Step 3: Replace in AIChatEditor.tsx**

Replace lines 66-87 (edges, nodes subscriptions + getUpstreamNodes callback) with:
```typescript
import { useUpstreamNodes } from '@/hooks/useUpstreamNodes'
// ...
const upstreamNodes = useUpstreamNodes(nodeId)
```

In `handleSubmit`, replace `getUpstreamNodes()` call with just `upstreamNodes`.

**Step 4: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/hooks/useUpstreamNodes.ts web/src/components/editor/PromptEditor.tsx web/src/components/panel/AIChatEditor.tsx
git commit -m "refactor: extract useUpstreamNodes hook, eliminate duplicate traversal"
```

---

### Task 2.5: Split `workflowStore` into focused slices

The mega-store (292 lines, 25 methods, 8 concerns) is the highest-priority refactoring target.

**Files:**
- Create: `web/src/stores/executionStore.ts`
- Create: `web/src/stores/uiStore.ts`
- Modify: `web/src/stores/workflowStore.ts`
- Modify: all consumers (update imports)

**Step 1: Create `executionStore.ts`**

Extract execution-related state: `isRunning`, `runEvents`, `sessionState`, `nodeStatuses`, `nodeStartTimes`, `nodeDurations`, and their methods.

```typescript
// web/src/stores/executionStore.ts
import { create } from 'zustand'

export type RunEvent = {
  type: string
  data: Record<string, unknown>
}

export type NodeRunStatus = 'idle' | 'running' | 'completed' | 'error'

type ExecutionState = {
  isRunning: boolean
  setIsRunning: (running: boolean) => void
  runEvents: RunEvent[]
  addRunEvent: (event: RunEvent) => void
  clearRunEvents: () => void
  sessionState: Record<string, unknown>
  setSessionState: (state: Record<string, unknown>) => void
  nodeStatuses: Record<string, NodeRunStatus>
  setNodeStatus: (nodeId: string, status: NodeRunStatus) => void
  clearNodeStatuses: () => void
  nodeStartTimes: Record<string, number>
  nodeDurations: Record<string, number>
}

export const useExecutionStore = create<ExecutionState>((set, get) => ({
  isRunning: false,
  runEvents: [],
  sessionState: {},
  nodeStatuses: {},
  nodeStartTimes: {},
  nodeDurations: {},
  setIsRunning: (running) => set({ isRunning: running }),
  addRunEvent: (event) => set({ runEvents: [...get().runEvents, event] }),
  clearRunEvents: () => set({ runEvents: [], sessionState: {} }),
  setSessionState: (state) => set({ sessionState: state }),
  setNodeStatus: (nodeId, status) => {
    const prev = get()
    const updates: Partial<ExecutionState> = {
      nodeStatuses: { ...prev.nodeStatuses, [nodeId]: status },
    }
    if (status === 'running') {
      updates.nodeStartTimes = { ...prev.nodeStartTimes, [nodeId]: Date.now() }
    } else if (status === 'completed' || status === 'error') {
      const start = prev.nodeStartTimes[nodeId]
      if (start) {
        updates.nodeDurations = {
          ...prev.nodeDurations,
          [nodeId]: (Date.now() - start) / 1000,
        }
      }
    }
    set(updates)
  },
  clearNodeStatuses: () => set({ nodeStatuses: {}, nodeStartTimes: {}, nodeDurations: {} }),
}))
```

**Step 2: Create `uiStore.ts`**

Extract UI state: `selectedNodeId`, `forcePreviewTab`.

```typescript
// web/src/stores/uiStore.ts
import { create } from 'zustand'

type UIState = {
  selectedNodeId: string | null
  selectNode: (id: string | null) => void
  forcePreviewTab: boolean
  setForcePreviewTab: (force: boolean) => void
}

export const useUIStore = create<UIState>((set) => ({
  selectedNodeId: null,
  selectNode: (id) => set({ selectedNodeId: id }),
  forcePreviewTab: false,
  setForcePreviewTab: (force) => set({ forcePreviewTab: force }),
}))
```

**Step 3: Trim `workflowStore.ts`**

Remove all execution and UI state/methods from `workflowStore.ts`. Keep only: nodes, edges, canvas operations, workflow identity, node CRUD, group management. Re-export types for backward compatibility:

```typescript
// At the top of workflowStore.ts
export type { RunEvent, NodeRunStatus } from './executionStore'
```

**Step 4: Update all consumer imports**

Search for all files importing from `workflowStore` and update:
- Files using `isRunning`, `runEvents`, `nodeStatuses`, etc. → import from `executionStore`
- Files using `selectedNodeId`, `forcePreviewTab` → import from `uiStore`

Key files to update:
- `web/src/pages/Editor.tsx`
- `web/src/components/panel/RightPanel.tsx`
- `web/src/components/panel/PanelLogs.tsx`
- `web/src/components/panel/PanelChat.tsx`
- `web/src/components/panel/PanelPreview.tsx`
- `web/src/components/panel/PanelData.tsx`
- `web/src/components/editor/Canvas.tsx`
- `web/src/components/editor/nodes/UpalNode.tsx`
- `web/src/components/console/BottomConsole.tsx`
- `web/src/hooks/useExecuteRun.ts`

**Step 5: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 6: Commit**

```bash
git add web/src/stores/ web/src/pages/ web/src/components/ web/src/hooks/
git commit -m "refactor: split workflowStore into execution + UI slices"
```

---

### Task 2.6: Extract PromptEditor TipTap extensions

Move TipTap extension configuration and serialization out of the 366-line PromptEditor component.

**Files:**
- Create: `web/src/components/editor/extensions/CustomMention.ts`
- Create: `web/src/lib/promptSerialization.ts`
- Modify: `web/src/components/editor/PromptEditor.tsx`

**Step 1: Move serialization functions**

Move `serializeContent` (lines 65-80) and `deserializeContent` (lines 84-119) to:

```typescript
// web/src/lib/promptSerialization.ts
import type { JSONContent } from '@tiptap/core'
import type { MentionItem } from '@/components/editor/MentionList'

export function serializeContent(json: JSONContent): string { /* exact same code */ }
export function deserializeContent(text: string, nodeMap: Map<string, MentionItem>): JSONContent { /* exact same code */ }
```

**Step 2: Move CustomMention extension**

Move `MentionPill` (lines 25-39), `customFindSuggestionMatch` (lines 43-61), and `CustomMention` (lines 123-147) to:

```typescript
// web/src/components/editor/extensions/CustomMention.ts
```

**Step 3: Update PromptEditor imports**

Replace inline definitions with imports. PromptEditor should shrink from 366 lines to ~180 lines.

**Step 4: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: No errors

**Step 5: Commit**

```bash
git add web/src/components/editor/extensions/ web/src/lib/promptSerialization.ts web/src/components/editor/PromptEditor.tsx
git commit -m "refactor: extract TipTap extensions and serialization from PromptEditor"
```

---

## Final Verification

After both tracks complete:

**Step 1:** Run full backend tests
```bash
make test
```

**Step 2:** Run frontend type check
```bash
make test-frontend
```

**Step 3:** Run full build
```bash
make build
```

All must pass with zero errors.

---

## Summary

| Track | Tasks | Lines Removed (est.) | Duplication Eliminated |
|-------|-------|---------------------|----------------------|
| Go Backend | 6 tasks | ~200 lines | JSON parsing (3x), LLM resolution (3x), layout gen (2x), text extract (5x), SSE errors (4x), tool loop (1x) |
| React Frontend | 6 tasks | ~400 lines | Model grouping (3x), model selector (3x), upstream nodes (2x), store concerns (8→3), PromptEditor concerns (3→1) |
| **Total** | **12 tasks** | **~600 lines** | **24 duplication points** |
