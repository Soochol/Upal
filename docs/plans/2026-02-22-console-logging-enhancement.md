# Console Logging Enhancement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add token counts, finish reason, and precise timing to SSE events; redesign the console with log-level filtering, search, node filter, expandable rows, relative timestamps, and group-by-node view.

**Architecture:** Backend enriches `node_completed` events with `tokens`, `finish_reason`, and `completed_at` by fixing a field-drop bug in `llm_builder.go` and adding usage parsing to model adapters. Frontend extends types, adds a `runStartTime` field to the store, and redesigns `Console.tsx` with rich filtering/grouping UI.

**Tech Stack:** Go 1.23 (backend), React 19 + TypeScript + Zustand + Tailwind v4 (frontend)

---

## Task 1: Anthropic — Parse and Propagate Token Usage

**Files:**
- Modify: `internal/model/anthropic.go`
- Test: `internal/model/anthropic_test.go`

**Step 1: Add failing test for token counts**

Add to `anthropic_test.go` (after existing tests):

```go
func TestAnthropicLLM_TokenUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "Hello!"}},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  42,
				"output_tokens": 17,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))
	req := &adkmodel.LLMRequest{
		Model:    "claude-sonnet-4-20250514",
		Contents: []*genai.Content{{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("hi")}}},
	}

	var got []*adkmodel.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, resp)
	}

	if len(got) != 1 {
		t.Fatalf("got %d responses, want 1", len(got))
	}
	u := got[0].UsageMetadata
	if u == nil {
		t.Fatal("UsageMetadata is nil, expected token counts")
	}
	if u.PromptTokenCount != 42 {
		t.Errorf("PromptTokenCount = %d, want 42", u.PromptTokenCount)
	}
	if u.CandidatesTokenCount != 17 {
		t.Errorf("CandidatesTokenCount = %d, want 17", u.CandidatesTokenCount)
	}
	if u.TotalTokenCount != 59 {
		t.Errorf("TotalTokenCount = %d, want 59", u.TotalTokenCount)
	}
}
```

**Step 2: Run test to verify it fails**

```
go test ./internal/model/... -run TestAnthropicLLM_TokenUsage -v
```
Expected: FAIL — `UsageMetadata is nil, expected token counts`

**Step 3: Add `anthropicUsage` type and `Usage` field to `anthropicAPIResponse`**

In `internal/model/anthropic.go`, locate the `anthropicAPIResponse` struct (line 323) and extend it:

```go
// Before:
type anthropicAPIResponse struct {
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
}

// After:
type anthropicUsage struct {
	InputTokens  int32 `json:"input_tokens"`
	OutputTokens int32 `json:"output_tokens"`
}

type anthropicAPIResponse struct {
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
	Usage      anthropicUsage         `json:"usage"`
}
```

**Step 4: Populate `UsageMetadata` in `convertResponse`**

In `convertResponse` (line 278), after the `FinishReason` switch, before the `return`:

```go
// After the switch apiResp.StopReason { ... } block, before return llmResp:
if apiResp.Usage.InputTokens > 0 || apiResp.Usage.OutputTokens > 0 {
	llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     apiResp.Usage.InputTokens,
		CandidatesTokenCount: apiResp.Usage.OutputTokens,
		TotalTokenCount:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
	}
}
```

Import `google.golang.org/genai` is already present.

**Step 5: Run test to verify it passes**

```
go test ./internal/model/... -run TestAnthropicLLM_TokenUsage -v -race
```
Expected: PASS

**Step 6: Run all model tests**

```
go test ./internal/model/... -v -race
```
Expected: all PASS

**Step 7: Commit**

```bash
git add internal/model/anthropic.go internal/model/anthropic_test.go
git commit -m "feat(model): parse Anthropic usage metadata into LLMResponse.UsageMetadata"
```

---

## Task 2: OpenAI — Parse and Propagate Token Usage

**Files:**
- Modify: `internal/model/openai.go`

**Step 1: Read the current openai.go response types**

Look for the response struct in `internal/model/openai.go` — find `openaiChatResponse` and `openaiChoice`.

**Step 2: Add `openaiUsage` type and `Usage` field**

Add after existing response types in `openai.go`:

```go
type openaiUsage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}
```

Add `Usage openaiUsage \`json:"usage"\`` to `openaiChatResponse`.

**Step 3: Populate `UsageMetadata` in the OpenAI `convertResponse` (or equivalent)**

In the function that converts `openaiChatResponse` to `*adkmodel.LLMResponse`, before returning:

```go
if resp.Usage.TotalTokens > 0 {
	llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     resp.Usage.PromptTokens,
		CandidatesTokenCount: resp.Usage.CompletionTokens,
		TotalTokenCount:      resp.Usage.TotalTokens,
	}
}
```

**Step 4: Run all tests**

```
go test ./internal/model/... -v -race
```
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/model/openai.go
git commit -m "feat(model): parse OpenAI usage metadata into LLMResponse.UsageMetadata"
```

---

## Task 3: Fix `llm_builder.go` — Stop Dropping `FinishReason` and `UsageMetadata`

**Files:**
- Modify: `internal/agents/llm_builder.go:193-196`

This is the critical bug: when the node builder constructs the final `node_completed` session event, it creates a new `LLMResponse` struct that silently drops `FinishReason` and `UsageMetadata` from the model's response.

**Step 1: Open the file and locate the bug**

In `internal/agents/llm_builder.go`, find lines 190-199:

```go
if len(toolCalls) == 0 {
    result := strings.TrimSpace(llmutil.ExtractContentSavingAudio(resp, outputDir))
    _ = state.Set(nodeID, result)

    event := session.NewEvent(ctx.InvocationID())
    event.Author = nodeID
    event.Branch = ctx.Branch()
    event.LLMResponse = adkmodel.LLMResponse{  // ← line 193: drops FinishReason + UsageMetadata
        Content:      resp.Content,
        TurnComplete: true,
    }
    event.Actions.StateDelta[nodeID] = result
    yield(event, nil)
    return
}
```

**Step 2: Fix the bug — propagate all fields from `resp`**

```go
// Replace lines 193-196 with:
event.LLMResponse = adkmodel.LLMResponse{
    Content:       resp.Content,
    TurnComplete:  true,
    FinishReason:  resp.FinishReason,
    UsageMetadata: resp.UsageMetadata,
}
```

**Step 3: Build to verify no compile errors**

```
go build ./internal/agents/...
```

**Step 4: Run all tests**

```
go test ./... -v -race
```
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/agents/llm_builder.go
git commit -m "fix(agents): propagate FinishReason and UsageMetadata from LLM response to session event"
```

---

## Task 4: Enrich `node_completed` Payload in `classifyEvent`

**Files:**
- Modify: `internal/services/workflow.go`

**Step 1: Find `classifyEvent` in `workflow.go`**

Look for `case` that returns `EventNodeCompleted`. The current payload is:

```go
return upal.WorkflowEvent{
    Type:   upal.EventNodeCompleted,
    NodeID: nodeID,
    Payload: map[string]any{
        "node_id":     nodeID,
        "output":      llmutil.ExtractContent(&event.LLMResponse),
        "state_delta": event.Actions.StateDelta,
    },
}
```

**Step 2: Extend the payload to include tokens and finish_reason**

```go
payload := map[string]any{
    "node_id":     nodeID,
    "output":      llmutil.ExtractContent(&event.LLMResponse),
    "state_delta": event.Actions.StateDelta,
}

// Add token counts when available (populated by Anthropic/OpenAI adapters)
if u := event.LLMResponse.UsageMetadata; u != nil {
    payload["tokens"] = map[string]any{
        "input":  u.PromptTokenCount,
        "output": u.CandidatesTokenCount,
        "total":  u.TotalTokenCount,
    }
}

// Add finish reason when non-default
if fr := event.LLMResponse.FinishReason; fr != 0 {
    payload["finish_reason"] = fr.String()
}

return upal.WorkflowEvent{
    Type:    upal.EventNodeCompleted,
    NodeID:  nodeID,
    Payload: payload,
}
```

`genai.FinishReason` has a `.String()` method. Zero value is `FinishReasonUnspecified`; non-zero values like `FinishReasonStop` will produce `"STOP"`.

**Step 3: Build and test**

```
go build ./internal/services/...
go test ./internal/services/... -v -race
```

**Step 4: Commit**

```bash
git add internal/services/workflow.go
git commit -m "feat(workflow): add tokens and finish_reason to node_completed SSE payload"
```

---

## Task 5: Add `completed_at` Timestamp to `node_completed` Events

**Files:**
- Modify: `internal/services/run/publisher.go`

**Step 1: Find the existing `started_at` injection in `publisher.go`**

Look for the block that adds `started_at` to `node_started` events. It looks like:

```go
if ev.Type == upal.EventNodeStarted {
    ev.Payload["started_at"] = time.Now().UnixMilli()
}
```

**Step 2: Add a symmetric `completed_at` injection immediately after**

```go
if ev.Type == upal.EventNodeCompleted {
    ev.Payload["completed_at"] = time.Now().UnixMilli()
}
```

**Step 3: Build and test**

```
go build ./internal/services/run/...
go test ./... -v -race
```

**Step 4: Commit**

```bash
git add internal/services/run/publisher.go
git commit -m "feat(publisher): inject completed_at timestamp on node_completed events"
```

---

## Task 6: Frontend — Extend Types for New SSE Fields

**Files:**
- Modify: `web/src/shared/types/index.ts`

**Step 1: Add `TokenUsage` type and extend `NodeCompletedEvent`**

In `web/src/shared/types/index.ts`, find line 146:

```ts
// Before:
export type NodeCompletedEvent = { type: 'node_completed'; nodeId: string; output: string; stateDelta: Record<string, unknown> }

// After:
export type TokenUsage = {
  input: number
  output: number
  total: number
}

export type NodeCompletedEvent = {
  type: 'node_completed'
  nodeId: string
  output: string
  stateDelta: Record<string, unknown>
  tokens?: TokenUsage
  finishReason?: string
  completedAt?: number
}
```

**Step 2: Type-check**

```
cd web && npx tsc -b --noEmit
```
Expected: no errors

**Step 3: Commit**

```bash
git add web/src/shared/types/index.ts
git commit -m "feat(types): add TokenUsage type and extend NodeCompletedEvent with tokens/finishReason/completedAt"
```

---

## Task 7: Frontend — Parse New Fields in SSE Payload

**Files:**
- Modify: `web/src/entities/run/api/index.ts`

**Step 1: Add `TokenUsage` to imports and extend `node_completed` parsing**

In `web/src/entities/run/api/index.ts`, update the import at line 2:

```ts
import type { RunRecord, RunListResponse, RunEvent, ToolCall, TokenUsage } from '@/shared/types'
```

Update the `case 'node_completed':` block (lines 29-35):

```ts
case 'node_completed':
  return {
    type: 'node_completed',
    nodeId,
    output: data.output as string,
    stateDelta: (data.state_delta ?? {}) as Record<string, unknown>,
    tokens: data.tokens as TokenUsage | undefined,
    finishReason: data.finish_reason as string | undefined,
    completedAt: data.completed_at as number | undefined,
  }
```

**Step 2: Type-check**

```
cd web && npx tsc -b --noEmit
```
Expected: no errors

**Step 3: Commit**

```bash
git add web/src/entities/run/api/index.ts
git commit -m "feat(api): parse tokens, finishReason, completedAt from node_completed SSE event"
```

---

## Task 8: Frontend — Add `runStartTime` to Execution Store

**Files:**
- Modify: `web/src/entities/run/model/store.ts`

**Step 1: Add `runStartTime` field and actions to `ExecutionState`**

In `web/src/entities/run/model/store.ts`:

```ts
// Add to ExecutionState type:
runStartTime: number | null
setRunStartTime: (t: number | null) => void
```

In the `create<ExecutionState>` initializer:

```ts
// Add to initial state:
runStartTime: null,
```

Add the action:

```ts
setRunStartTime: (t) => set({ runStartTime: t }),
```

Update `clearRunEvents` to also reset `runStartTime`:

```ts
// Before:
clearRunEvents: () => {
  set({ runEvents: [], sessionState: {} })
},

// After:
clearRunEvents: () => {
  set({ runEvents: [], sessionState: {}, runStartTime: null })
},
```

**Step 2: Type-check**

```
cd web && npx tsc -b --noEmit
```

**Step 3: Commit**

```bash
git add web/src/entities/run/model/store.ts
git commit -m "feat(store): add runStartTime field for console relative timestamp display"
```

---

## Task 9: Wire `runStartTime` in `useExecuteRun`

**Files:**
- Modify: `web/src/features/execute-workflow/model/useExecuteRun.ts`

**Step 1: Subscribe to `setRunStartTime` and call it when run begins**

In `web/src/features/execute-workflow/model/useExecuteRun.ts`:

Add to store subscriptions (after line 42):
```ts
const setRunStartTime = useExecutionStore((s) => s.setRunStartTime)
```

In `executeRun`, after `clearRunEvents()` (line 51):
```ts
clearRunEvents()
clearNodeStatuses()
setRunStartTime(Date.now())  // ← add this line
setIsRunning(true)
```

Add `setRunStartTime` to the `useCallback` dependency array at line 91.

**Step 2: Type-check**

```
cd web && npx tsc -b --noEmit
```

**Step 3: Commit**

```bash
git add web/src/features/execute-workflow/model/useExecuteRun.ts
git commit -m "feat(execute): set runStartTime when workflow execution begins"
```

---

## Task 10: Add `formatEventVerbose` and `formatRelativeTime` Utilities

**Files:**
- Modify: `web/src/shared/lib/eventFormatting.ts`

**Step 1: Add `formatRelativeTime` helper**

At the bottom of `eventFormatting.ts`:

```ts
/** Returns "+1.2s" offset from run start, or '' if timestamps are unavailable. */
export function formatRelativeTime(eventMs: number | undefined, runStartMs: number | null): string {
  if (!runStartMs || !eventMs) return ''
  const delta = (eventMs - runStartMs) / 1000
  if (delta < 0) return ''
  return `+${delta.toFixed(1)}s`
}
```

**Step 2: Add `formatEventVerbose` — same as `formatEvent` but without truncation**

```ts
/** Same as formatEvent but without any character truncation.
 *  Use for "verbose" log level where the user wants to see the full content. */
export function formatEventVerbose(event: RunEvent): string {
  switch (event.type) {
    case 'node_started':
      return `[${event.nodeId}] started`
    case 'tool_call':
      return event.calls
        .map((c) => `[${event.nodeId}] ${c.name}(${JSON.stringify(c.args ?? {})})`)
        .join('\n')
    case 'tool_result':
      return event.results
        .map((r) => `[${event.nodeId}] ${r.name} \u2192 ${JSON.stringify(r.response ?? {})}`)
        .join('\n')
    case 'node_completed': {
      const deltaKeys = Object.keys(event.stateDelta ?? {})
      const suffix = deltaKeys.length > 0 ? ` state: {${deltaKeys.join(', ')}}` : ''
      return `[${event.nodeId}] ${event.output}${suffix}`
    }
    case 'done':
      return `status=${event.status}`
    case 'error':
      return event.message
    case 'info':
      return event.message
    case 'node_skipped':
      return `[${event.nodeId}] skipped`
    case 'node_waiting':
      return `[${event.nodeId}] waiting`
    case 'node_resumed':
      return `[${event.nodeId}] resumed`
    case 'log':
      return `[${event.nodeId}] ${event.message}`
  }
}
```

**Step 3: Type-check**

```
cd web && npx tsc -b --noEmit
```

**Step 4: Commit**

```bash
git add web/src/shared/lib/eventFormatting.ts
git commit -m "feat(formatting): add formatEventVerbose (no truncation) and formatRelativeTime"
```

---

## Task 11: Redesign `Console.tsx` with Rich Controls

**Files:**
- Modify: `web/src/widgets/bottom-console/ui/Console.tsx`

This is the largest change. Replace the entire file content.

**Step 1: Write the new Console.tsx**

```tsx
import { useEffect, useRef, useState, useMemo } from 'react'
import { useExecutionStore } from '@/entities/run'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import {
  ChevronUp, ChevronDown, Trash2, Terminal, Copy, Check, Layers,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import {
  eventColorMap,
  formatEvent,
  formatEventVerbose,
  formatRelativeTime,
} from '@/shared/lib/eventFormatting'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import { useCopyToClipboard } from '@/shared/lib/useCopyToClipboard'
import type { RunEvent, NodeCompletedEvent, NodeStartedEvent } from '@/shared/types'

const DEFAULT_HEIGHT = 160
const MIN_HEIGHT = 80
const MAX_HEIGHT = 500

type LogLevel = 'info' | 'debug' | 'verbose'

const INFO_TYPES = new Set([
  'node_started', 'node_completed', 'node_skipped', 'node_waiting', 'node_resumed', 'done', 'error',
])

function matchesLevel(event: RunEvent, level: LogLevel): boolean {
  if (level === 'info') return INFO_TYPES.has(event.type)
  return true // debug + verbose show all types
}

function getEventTimestamp(event: RunEvent): number | undefined {
  if (event.type === 'node_started') return (event as NodeStartedEvent).startedAt
  if (event.type === 'node_completed') return (event as NodeCompletedEvent).completedAt
  return undefined
}

// ── ConsoleRow ──────────────────────────────────────────────────────────────
type ConsoleRowProps = {
  event: RunEvent
  originalIndex: number
  isExpanded: boolean
  onToggle: (i: number) => void
  runStartTime: number | null
  verbose: boolean
}

function ConsoleRow({ event, originalIndex, isExpanded, onToggle, runStartTime, verbose }: ConsoleRowProps) {
  const ts = getEventTimestamp(event)
  const relTime = formatRelativeTime(ts, runStartTime)
  const color = eventColorMap[event.type] ?? 'text-muted-foreground'
  const text = verbose ? formatEventVerbose(event) : formatEvent(event)
  const completed = event.type === 'node_completed' ? (event as NodeCompletedEvent) : null

  return (
    <div>
      <div
        className={cn('flex items-center gap-1.5 py-0.5 cursor-pointer hover:bg-muted/20 rounded px-1 group', color)}
        onClick={() => onToggle(originalIndex)}
      >
        <span className="text-muted-foreground/50 w-10 text-right shrink-0 text-[10px] tabular-nums">
          {relTime}
        </span>
        <span className="text-muted-foreground/80 shrink-0 w-[110px] truncate text-[10px]">
          {event.type}
        </span>
        <span className="flex-1 truncate">{text}</span>
        {completed?.tokens && (
          <span className="text-muted-foreground/50 shrink-0 text-[10px] tabular-nums">
            {completed.tokens.input}↑{completed.tokens.output}↓
          </span>
        )}
        <ChevronDown
          className={cn(
            'h-3 w-3 shrink-0 text-muted-foreground/30 transition-transform opacity-0 group-hover:opacity-100',
            isExpanded && 'rotate-180 opacity-100',
          )}
        />
      </div>
      {isExpanded && (
        <pre className="ml-[88px] my-0.5 text-[11px] text-muted-foreground/80 bg-muted/20 rounded p-2 overflow-x-auto whitespace-pre-wrap break-all">
          {JSON.stringify(event, null, 2)}
        </pre>
      )}
    </div>
  )
}

// ── NodeGroup ────────────────────────────────────────────────────────────────
type GroupEntry = { event: RunEvent; originalIndex: number }

function NodeGroup({
  nodeId,
  entries,
  expandedRows,
  onToggle,
  runStartTime,
  verbose,
}: {
  nodeId: string
  entries: GroupEntry[]
  expandedRows: Set<number>
  onToggle: (i: number) => void
  runStartTime: number | null
  verbose: boolean
}) {
  const [collapsed, setCollapsed] = useState(false)

  // Aggregate: total tokens and duration for this group
  const { totalIn, totalOut, durationStr } = useMemo(() => {
    let totalIn = 0, totalOut = 0
    let firstStart: number | undefined
    let lastComplete: number | undefined
    for (const { event: e } of entries) {
      if (e.type === 'node_completed') {
        const c = e as NodeCompletedEvent
        if (c.tokens) { totalIn += c.tokens.input; totalOut += c.tokens.output }
        if (c.completedAt) lastComplete = Math.max(lastComplete ?? 0, c.completedAt)
      }
      if (e.type === 'node_started') {
        const s = e as NodeStartedEvent
        if (s.startedAt) firstStart = Math.min(firstStart ?? Infinity, s.startedAt)
      }
    }
    let durationStr = ''
    if (firstStart && lastComplete && lastComplete > firstStart) {
      durationStr = `${((lastComplete - firstStart) / 1000).toFixed(1)}s`
    }
    return { totalIn, totalOut, durationStr }
  }, [entries])

  return (
    <div className="mb-1">
      <div
        className="flex items-center gap-1.5 px-1 py-0.5 cursor-pointer hover:bg-muted/20 rounded text-muted-foreground"
        onClick={() => setCollapsed(!collapsed)}
      >
        <ChevronDown className={cn('h-3 w-3 transition-transform', collapsed && '-rotate-90')} />
        <span className="font-medium text-foreground text-[11px]">{nodeId}</span>
        <span className="text-muted-foreground/50 text-[10px]">({entries.length})</span>
        {(totalIn > 0 || totalOut > 0) && (
          <span className="text-muted-foreground/50 text-[10px] tabular-nums ml-1">
            {totalIn}↑{totalOut}↓
          </span>
        )}
        {durationStr && (
          <span className="text-muted-foreground/50 text-[10px] ml-1">{durationStr}</span>
        )}
      </div>
      {!collapsed && (
        <div className="ml-4">
          {entries.map(({ event, originalIndex }) => (
            <ConsoleRow
              key={originalIndex}
              event={event}
              originalIndex={originalIndex}
              isExpanded={expandedRows.has(originalIndex)}
              onToggle={onToggle}
              runStartTime={runStartTime}
              verbose={verbose}
            />
          ))}
        </div>
      )}
    </div>
  )
}

// ── Console (main) ───────────────────────────────────────────────────────────
export function Console() {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const runStartTime = useExecutionStore((s) => s.runStartTime)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isExpanded, setIsExpanded] = useState(false)
  const { size: height, handleMouseDown } = useResizeDrag({
    direction: 'vertical',
    min: MIN_HEIGHT,
    max: MAX_HEIGHT,
    initial: DEFAULT_HEIGHT,
  })
  const { copied, copyToClipboard } = useCopyToClipboard()

  // Filter/view state
  const [logLevel, setLogLevel] = useState<LogLevel>('info')
  const [searchQuery, setSearchQuery] = useState('')
  const [nodeFilter, setNodeFilter] = useState('')
  const [groupByNode, setGroupByNode] = useState(false)
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())

  // Unique nodeIds for filter dropdown
  const nodeIds = useMemo(
    () => [...new Set(runEvents.flatMap((e) => ('nodeId' in e && e.nodeId ? [e.nodeId] : [])))],
    [runEvents],
  )

  // Filtered events (preserving original indices for stable expandedRows)
  const filtered = useMemo<GroupEntry[]>(() => {
    const verbose = logLevel === 'verbose'
    return runEvents
      .map((event, originalIndex) => ({ event, originalIndex }))
      .filter(({ event }) => {
        if (!matchesLevel(event, logLevel)) return false
        if (nodeFilter && 'nodeId' in event && event.nodeId !== nodeFilter) return false
        if (searchQuery) {
          const text = (verbose ? formatEventVerbose(event) : formatEvent(event)).toLowerCase()
          const typeStr = event.type.toLowerCase()
          if (!text.includes(searchQuery.toLowerCase()) && !typeStr.includes(searchQuery.toLowerCase())) return false
        }
        return true
      })
  }, [runEvents, logLevel, nodeFilter, searchQuery])

  const handleToggleRow = (i: number) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const handleCopy = () => {
    const verbose = logLevel === 'verbose'
    const text = filtered
      .map(({ event }) => `[${event.type}] ${verbose ? formatEventVerbose(event) : formatEvent(event)}`)
      .join('\n')
    copyToClipboard(text)
  }

  // Auto-expand when a run starts
  useEffect(() => {
    if (isRunning) setIsExpanded(true)
  }, [isRunning])

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [filtered])

  // Build node groups for grouped view
  const nodeGroups = useMemo(() => {
    if (!groupByNode) return null
    const map = new Map<string, GroupEntry[]>()
    const ungrouped: GroupEntry[] = []
    for (const entry of filtered) {
      const nodeId = 'nodeId' in entry.event ? entry.event.nodeId : null
      if (nodeId) {
        if (!map.has(nodeId)) map.set(nodeId, [])
        map.get(nodeId)!.push(entry)
      } else {
        ungrouped.push(entry)
      }
    }
    return { groups: [...map.entries()], ungrouped }
  }, [filtered, groupByNode])

  return (
    <footer className="border-t border-border bg-background">
      {/* Resize handle */}
      {isExpanded && (
        <div
          onMouseDown={handleMouseDown}
          className="h-1 cursor-row-resize hover:bg-primary/20 active:bg-primary/40 transition-colors"
        />
      )}

      {/* Toggle bar */}
      <div className="flex items-center justify-between px-3 h-9 gap-2">
        {/* Left: label + badges */}
        <button
          className="flex items-center gap-1.5 text-xs hover:text-foreground transition-colors shrink-0"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <Terminal className="h-3.5 w-3.5" />
          <span className="font-medium">Console</span>
          {isRunning ? (
            <Badge variant="default" className="animate-pulse text-[10px] px-1.5 py-0">running</Badge>
          ) : (
            runEvents.length > 0 && (
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0">{filtered.length}/{runEvents.length}</Badge>
            )
          )}
          {isExpanded ? <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" /> : <ChevronUp className="h-3.5 w-3.5 text-muted-foreground" />}
        </button>

        {/* Right: controls (only when expanded) */}
        {isExpanded && (
          <div className="flex items-center gap-1 min-w-0">
            {/* Log level */}
            <div className="flex rounded border border-border overflow-hidden shrink-0">
              {(['info', 'debug', 'verbose'] as LogLevel[]).map((l) => (
                <button
                  key={l}
                  onClick={() => setLogLevel(l)}
                  className={cn(
                    'px-2 py-0.5 text-[10px] font-medium transition-colors',
                    logLevel === l
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground hover:bg-muted/30',
                  )}
                >
                  {l}
                </button>
              ))}
            </div>

            {/* Search */}
            <input
              type="text"
              placeholder="search…"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-6 w-28 rounded border border-border bg-background px-2 text-[11px] text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-primary/40 min-w-0"
            />

            {/* Node filter */}
            {nodeIds.length > 0 && (
              <select
                value={nodeFilter}
                onChange={(e) => setNodeFilter(e.target.value)}
                className="h-6 rounded border border-border bg-background px-1 text-[11px] text-foreground focus:outline-none focus:ring-1 focus:ring-primary/40 max-w-[100px]"
              >
                <option value="">all nodes</option>
                {nodeIds.map((id) => (
                  <option key={id} value={id}>{id}</option>
                ))}
              </select>
            )}

            {/* Group by node toggle */}
            <Button
              variant="ghost"
              size="icon"
              className={cn('h-6 w-6', groupByNode && 'text-primary bg-primary/10')}
              title="Group by node"
              onClick={() => setGroupByNode(!groupByNode)}
            >
              <Layers className="h-3 w-3" />
            </Button>

            {/* Copy */}
            {filtered.length > 0 && (
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleCopy}>
                {copied ? <Check className="h-3 w-3 text-success" /> : <Copy className="h-3 w-3" />}
              </Button>
            )}

            {/* Clear */}
            {runEvents.length > 0 && (
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
                <Trash2 className="h-3 w-3" />
              </Button>
            )}
          </div>
        )}
      </div>

      {/* Log area */}
      {isExpanded && (
        <div
          ref={scrollRef}
          style={{ height }}
          className="overflow-y-auto px-2 pb-3 font-mono text-xs border-t border-border"
        >
          {filtered.length === 0 ? (
            <p className="text-muted-foreground px-2 pt-2">
              {runEvents.length === 0 ? 'Ready. Run a workflow to see events…' : 'No events match current filters.'}
            </p>
          ) : groupByNode && nodeGroups ? (
            <div className="pt-1">
              {nodeGroups.ungrouped.map(({ event, originalIndex }) => (
                <ConsoleRow
                  key={originalIndex}
                  event={event}
                  originalIndex={originalIndex}
                  isExpanded={expandedRows.has(originalIndex)}
                  onToggle={handleToggleRow}
                  runStartTime={runStartTime}
                  verbose={logLevel === 'verbose'}
                />
              ))}
              {nodeGroups.groups.map(([nodeId, entries]) => (
                <NodeGroup
                  key={nodeId}
                  nodeId={nodeId}
                  entries={entries}
                  expandedRows={expandedRows}
                  onToggle={handleToggleRow}
                  runStartTime={runStartTime}
                  verbose={logLevel === 'verbose'}
                />
              ))}
            </div>
          ) : (
            <div className="pt-1">
              {filtered.map(({ event, originalIndex }) => (
                <ConsoleRow
                  key={originalIndex}
                  event={event}
                  originalIndex={originalIndex}
                  isExpanded={expandedRows.has(originalIndex)}
                  onToggle={handleToggleRow}
                  runStartTime={runStartTime}
                  verbose={logLevel === 'verbose'}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </footer>
  )
}
```

**Step 2: Type-check**

```
cd web && npx tsc -b --noEmit
```

If there are any TypeScript errors, fix them before proceeding. Common issues:
- `e.nodeId` access on `RunEvent` union type — use `'nodeId' in e && e.nodeId` guards
- `formatEventVerbose` not exported from `eventFormatting.ts` — verify export in Task 10

**Step 3: Run frontend build**

```
cd web && npm run build
```
Expected: builds without errors

**Step 4: Commit**

```bash
git add web/src/widgets/bottom-console/ui/Console.tsx
git commit -m "feat(console): add log-level filter, search, node filter, group-by-node, expandable rows, timestamps, token badges"
```

---

## Task 12: Update `PanelConsole.tsx` with Expandable Rows and Token Badge

**Files:**
- Modify: `web/src/widgets/right-panel/ui/PanelConsole.tsx`

**Step 1: Add expandable rows and token badge**

Replace the file content:

```tsx
import { useEffect, useRef, useState } from 'react'
import { useExecutionStore } from '@/entities/run'
import { Button } from '@/shared/ui/button'
import { Trash2, Copy, Check, ChevronDown } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { eventColorMap, formatEvent, formatRelativeTime } from '@/shared/lib/eventFormatting'
import { useCopyToClipboard } from '@/shared/lib/useCopyToClipboard'
import type { RunEvent, NodeCompletedEvent, NodeStartedEvent } from '@/shared/types'

type PanelConsoleProps = {
  selectedNodeId: string | null
}

function eventBelongsToNode(event: RunEvent, nodeId: string): boolean {
  return 'nodeId' in event && event.nodeId === nodeId
}

function getEventTimestamp(event: RunEvent): number | undefined {
  if (event.type === 'node_started') return (event as NodeStartedEvent).startedAt
  if (event.type === 'node_completed') return (event as NodeCompletedEvent).completedAt
  return undefined
}

export function PanelConsole({ selectedNodeId }: PanelConsoleProps) {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const runStartTime = useExecutionStore((s) => s.runStartTime)
  const scrollRef = useRef<HTMLDivElement>(null)
  const { copied, copyToClipboard } = useCopyToClipboard()
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())

  const filtered = selectedNodeId
    ? runEvents.map((e, i) => ({ event: e, index: i })).filter(({ event: e }) => eventBelongsToNode(e, selectedNodeId))
    : runEvents.map((e, i) => ({ event: e, index: i }))

  const handleToggle = (i: number) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const handleCopy = () => {
    const text = filtered.map(({ event: e }) => `[${e.type}] ${formatEvent(e)}`).join('\n')
    copyToClipboard(text)
  }

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [filtered])

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-border shrink-0">
        <span className="text-xs text-muted-foreground">
          {selectedNodeId ? `Node: ${selectedNodeId}` : 'All nodes'}
        </span>
        <div className="flex items-center gap-1">
          {filtered.length > 0 && (
            <>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleCopy}>
                {copied ? <Check className="h-3 w-3 text-success" /> : <Copy className="h-3 w-3" />}
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
                <Trash2 className="h-3 w-3" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Log area */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-2 py-2 font-mono text-xs">
        {filtered.length === 0 ? (
          <p className="text-muted-foreground px-1">
            {runEvents.length === 0 ? 'Ready. Run a workflow to see events...' : 'No events for this node yet.'}
          </p>
        ) : (
          filtered.map(({ event, index }) => {
            const ts = getEventTimestamp(event)
            const relTime = formatRelativeTime(ts, runStartTime)
            const color = eventColorMap[event.type] ?? 'text-muted-foreground'
            const completed = event.type === 'node_completed' ? (event as NodeCompletedEvent) : null
            const isExpanded = expandedRows.has(index)

            return (
              <div key={index}>
                <div
                  className={cn('flex items-center gap-1.5 py-0.5 cursor-pointer hover:bg-muted/20 rounded px-1 group', color)}
                  onClick={() => handleToggle(index)}
                >
                  {relTime && (
                    <span className="text-muted-foreground/50 text-[10px] shrink-0 tabular-nums">{relTime}</span>
                  )}
                  <span className="text-muted-foreground/80 shrink-0 text-[10px]">{event.type}</span>
                  <span className="flex-1 truncate">{formatEvent(event)}</span>
                  {completed?.tokens && (
                    <span className="text-muted-foreground/50 shrink-0 text-[10px] tabular-nums">
                      {completed.tokens.input}↑{completed.tokens.output}↓
                    </span>
                  )}
                  <ChevronDown
                    className={cn(
                      'h-3 w-3 shrink-0 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-transform',
                      isExpanded && 'rotate-180 opacity-100',
                    )}
                  />
                </div>
                {isExpanded && (
                  <pre className="ml-2 my-0.5 text-[11px] text-muted-foreground/80 bg-muted/20 rounded p-2 overflow-x-auto whitespace-pre-wrap break-all">
                    {JSON.stringify(event, null, 2)}
                  </pre>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
```

**Step 2: Type-check and build**

```
cd web && npx tsc -b --noEmit && npm run build
```

**Step 3: Commit**

```bash
git add web/src/widgets/right-panel/ui/PanelConsole.tsx
git commit -m "feat(panel-console): add expandable rows, relative timestamps, and token badges"
```

---

## Final Verification

**1. Start the dev stack:**
```bash
# Terminal 1:
make dev-backend

# Terminal 2:
make dev-frontend
```

**2. Run a workflow with an Anthropic or OpenAI model node**

Open `http://localhost:5173`, create a workflow with an Agent node using `anthropic/claude-haiku-*` or `openai/gpt-4o-mini`, run it.

**3. Check in the console:**
- `node_completed` events show `42↑17↓` token badge (if provider returned usage)
- Log level `info` shows only lifecycle events; switch to `debug` to see tool calls
- Switch to `verbose` to see full untruncated output
- Click any event row → JSON detail expands below
- Enter text in search box → events filter in real time
- Select a node from the node dropdown → only that node's events shown
- Click the `Layers` icon → events grouped under collapsible node headers with aggregated tokens + duration
- Events show `+0.0s`, `+0.8s` etc. relative timestamps

**4. Run Go tests:**
```
make test
```
Expected: all PASS, no race conditions

**5. Run TypeScript check:**
```
make test-frontend
```
Expected: no type errors
