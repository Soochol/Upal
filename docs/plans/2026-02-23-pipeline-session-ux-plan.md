# Pipeline Session UX Redesign — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the modal-based SessionDetailModal with a full-page linear timeline session detail, add a Workflows section to Pipeline Settings, and wire the 2-gate approval flow.

**Architecture:** Full-page `SessionDetailPage` at `/pipelines/:id/sessions/:sessionId` using `MainLayout`. Four stages (Collect → Analyze → Produce → Publish) rendered as vertical sections with collapsed/active/locked states. Pipeline Settings gains a Workflows accordion to register workflow templates. Backend gets a `Workflows` field on Pipeline and a PATCH endpoint for analysis edits.

**Tech Stack:** React 19, TypeScript, TanStack Query, Zustand, Tailwind CSS v4, Lucide icons, Go 1.23 (backend)

**Design doc:** `docs/plans/2026-02-23-pipeline-session-ux-design.md`

---

## Task 1: Backend — Add Workflows field to Pipeline

**Files:**
- Modify: `internal/upal/pipeline.go:6-20`
- Modify: `web/src/shared/types/index.ts:220-234`

**Step 1: Add PipelineWorkflow struct and Workflows field to Go Pipeline**

In `internal/upal/pipeline.go`, add after the `PipelineSource` struct:

```go
// PipelineWorkflow links an existing workflow to a pipeline for content production.
type PipelineWorkflow struct {
	WorkflowName string `json:"workflow_name"`
	Label        string `json:"label,omitempty"`
	AutoSelect   bool   `json:"auto_select,omitempty"`
}
```

Add to the `Pipeline` struct, after `Sources`:

```go
Workflows []PipelineWorkflow `json:"workflows,omitempty"`
```

**Step 2: Add matching TypeScript types**

In `web/src/shared/types/index.ts`, add after the `PipelineContext` type:

```typescript
export type PipelineWorkflow = {
  workflow_name: string
  label?: string
  auto_select?: boolean
}
```

Add to the `Pipeline` type, after `sources?`:

```typescript
workflows?: PipelineWorkflow[]
```

**Step 3: Verify type-check passes**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS (no errors)

**Step 4: Commit**

```bash
git add internal/upal/pipeline.go web/src/shared/types/index.ts
git commit -m "feat(pipeline): add Workflows field to Pipeline type"
```

---

## Task 2: Backend — Add PATCH endpoint for analysis editing

**Files:**
- Modify: `internal/api/content.go`
- Modify: `internal/api/server.go:112-122`
- Modify: `internal/services/content_session_service.go`
- Modify: `web/src/entities/content-session/api.ts`

**Step 1: Add UpdateAnalysis method to ContentSessionService**

Find the `ContentSessionService` in `internal/services/content_session_service.go`. Add a method:

```go
// UpdateAnalysis updates the summary and insights of an existing analysis.
func (s *ContentSessionService) UpdateAnalysis(sessionID string, summary string, insights []string) error {
	analysis, err := s.analysisRepo.GetBySession(sessionID)
	if err != nil {
		return err
	}
	if analysis == nil {
		return fmt.Errorf("no analysis found for session %s", sessionID)
	}
	analysis.Summary = summary
	analysis.Insights = insights
	return s.analysisRepo.Update(analysis)
}
```

Check if the `LLMAnalysisRepository` interface has an `Update` method. If not, add it to the interface and implement in memory/persistent adapters. The signature should be:

```go
Update(analysis *upal.LLMAnalysis) error
```

**Step 2: Add the PATCH handler in content.go**

Add to `internal/api/content.go`:

```go
func (s *Server) patchSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Summary  string   `json:"summary"`
		Insights []string `json:"insights"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.contentSvc.UpdateAnalysis(id, body.Summary, body.Insights); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	analysis, _ := s.contentSvc.GetAnalysis(id)
	json.NewEncoder(w).Encode(analysis)
}
```

**Step 3: Register the route in server.go**

In `internal/api/server.go`, inside the `/content-sessions/{id}` group, add:

```go
r.Patch("/{id}/analysis", s.patchSessionAnalysis)
```

**Step 4: Add frontend API function**

In `web/src/entities/content-session/api.ts`, add:

```typescript
export async function updateSessionAnalysis(
  id: string,
  data: { summary: string; insights: string[] },
): Promise<LLMAnalysis> {
  const res = await fetch(`${BASE}/content-sessions/${id}/analysis`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}
```

Import `LLMAnalysis` from the types file.

**Step 5: Verify**

Run: `go build ./cmd/upal && cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/api/content.go internal/api/server.go internal/services/content_session_service.go web/src/entities/content-session/api.ts
git commit -m "feat(api): add PATCH endpoint for session analysis editing"
```

---

## Task 3: Frontend — StageSection wrapper component

**Files:**
- Create: `web/src/pages/pipelines/session/StageSection.tsx`

This is the reusable wrapper that renders each stage in collapsed/active/locked state.

**Step 1: Create the component**

```tsx
import { useState, useRef, useEffect } from 'react'
import { ChevronDown, Check, Lock } from 'lucide-react'

export type StageState = 'completed' | 'active' | 'locked'

interface StageSectionProps {
  id: string               // for scroll targeting
  number: number           // 1-4
  title: string            // "Collect", "Analyze", etc.
  state: StageState
  summary?: string         // collapsed summary text for completed state
  statusText?: string      // right-side status like "✓ 35 items" or "⏳ pending"
  children: React.ReactNode
}

export function StageSection({
  id, number, title, state, summary, statusText, children,
}: StageSectionProps) {
  const [expanded, setExpanded] = useState(state === 'active')
  const ref = useRef<HTMLElement>(null)

  // Auto-expand when state becomes active
  useEffect(() => {
    if (state === 'active') setExpanded(true)
  }, [state])

  const isVisible = state !== 'locked' ? expanded || state === 'active' : false

  return (
    <section id={id} ref={ref} className="mb-6">
      <button
        onClick={() => state !== 'locked' && setExpanded(v => !v)}
        disabled={state === 'locked'}
        className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl border transition-all cursor-pointer
          ${state === 'active'
            ? 'border-foreground/20 bg-foreground/5'
            : state === 'completed'
            ? 'border-success/20 bg-success/5 hover:bg-success/10'
            : 'border-border/50 bg-muted/20 opacity-50 cursor-not-allowed'}`}
      >
        {/* Step indicator */}
        <div className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold shrink-0
          ${state === 'completed' ? 'bg-success text-white'
            : state === 'active' ? 'bg-foreground text-background'
            : 'bg-muted text-muted-foreground'}`}>
          {state === 'completed' ? <Check className="h-3.5 w-3.5" /> : number}
        </div>

        <span className={`text-sm font-semibold ${state === 'locked' ? 'text-muted-foreground' : 'text-foreground'}`}>
          {title}
        </span>

        {/* Summary for completed state when collapsed */}
        {state === 'completed' && !expanded && summary && (
          <span className="text-xs text-muted-foreground truncate flex-1 text-left ml-1">
            {summary}
          </span>
        )}

        <div className="flex-1" />

        {statusText && (
          <span className="text-xs text-muted-foreground shrink-0">{statusText}</span>
        )}

        {state === 'locked'
          ? <Lock className="h-3.5 w-3.5 text-muted-foreground" />
          : <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${isVisible ? 'rotate-180' : ''}`} />}
      </button>

      {isVisible && (
        <div className="mt-3 px-1 animate-in fade-in slide-in-from-top-2 duration-200">
          {children}
        </div>
      )}
    </section>
  )
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/StageSection.tsx
git commit -m "feat(session): add StageSection wrapper component"
```

---

## Task 4: Frontend — StickyProgressBar component

**Files:**
- Create: `web/src/pages/pipelines/session/StickyProgressBar.tsx`

**Step 1: Create the component**

```tsx
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import type { ContentSession } from '@/entities/content-session'

type Stage = 'collect' | 'analyze' | 'produce' | 'publish'

const STAGES: { value: Stage; label: string; sectionId: string }[] = [
  { value: 'collect', label: 'Collect', sectionId: 'stage-collect' },
  { value: 'analyze', label: 'Analyze', sectionId: 'stage-analyze' },
  { value: 'produce', label: 'Produce', sectionId: 'stage-produce' },
  { value: 'publish', label: 'Publish', sectionId: 'stage-publish' },
]

function getActiveStage(session: ContentSession): Stage {
  switch (session.status) {
    case 'collecting': return 'collect'
    case 'pending_review': return 'analyze'
    case 'approved':
    case 'producing': return 'produce'
    case 'published': return 'publish'
    default: return 'collect'
  }
}

function isStageCompleted(stage: Stage, activeStage: Stage): boolean {
  const order: Stage[] = ['collect', 'analyze', 'produce', 'publish']
  return order.indexOf(stage) < order.indexOf(activeStage)
}

interface StickyProgressBarProps {
  session: ContentSession
}

export function StickyProgressBar({ session }: StickyProgressBarProps) {
  const activeStage = getActiveStage(session)
  const createdAt = new Date(session.created_at).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  })

  const scrollTo = (sectionId: string) => {
    document.getElementById(sectionId)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return (
    <div className="sticky top-0 z-30 bg-background/95 backdrop-blur-sm border-b border-border px-6 py-3">
      <div className="max-w-4xl mx-auto">
        {/* Stepper */}
        <div className="flex items-center gap-1 mb-2">
          {STAGES.map((stage, idx) => {
            const isActive = activeStage === stage.value
            const isCompleted = isStageCompleted(stage.value, activeStage)
            // Also mark publish as completed if status is published
            const isDone = isCompleted || (stage.value === 'publish' && session.status === 'published')
            return (
              <div key={stage.value} className="flex items-center flex-1">
                <button
                  onClick={() => scrollTo(stage.sectionId)}
                  className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer
                    ${isActive
                      ? 'bg-foreground text-background'
                      : isDone
                      ? 'text-success hover:bg-success/10'
                      : 'text-muted-foreground hover:bg-muted/50'}`}
                >
                  <span className={`w-1.5 h-1.5 rounded-full ${isDone ? 'bg-success' : isActive ? 'bg-background' : 'bg-muted-foreground/30'}`} />
                  {stage.label}
                </button>
                {idx < STAGES.length - 1 && (
                  <div className={`flex-1 h-px mx-1 ${isDone ? 'bg-success/40' : 'bg-border/40'}`} />
                )}
              </div>
            )
          })}
        </div>

        {/* Meta info */}
        <div className="flex items-center gap-2 text-xs">
          <span className="font-medium">
            Session {session.session_number != null ? `#${session.session_number}` : session.id.slice(0, 8)}
          </span>
          <StatusBadge status={session.status} />
          {session.analysis && <ScoreIndicator score={session.analysis.score} />}
          <span className="text-muted-foreground">{createdAt}</span>
          <span className="text-muted-foreground">· {session.trigger_type}</span>
        </div>
      </div>
    </div>
  )
}

export { getActiveStage, isStageCompleted }
export type { Stage }
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/StickyProgressBar.tsx
git commit -m "feat(session): add StickyProgressBar component"
```

---

## Task 5: Frontend — CollectStage component

**Files:**
- Create: `web/src/pages/pipelines/session/stages/CollectStage.tsx`

Extracts the existing `SourcePanel` logic from `SessionDetailModal.tsx` into a standalone component.

**Step 1: Create the component**

```tsx
import { Loader2, ExternalLink } from 'lucide-react'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import type { ContentSession } from '@/entities/content-session'

interface CollectStageProps {
  session: ContentSession
}

export function CollectStage({ session }: CollectStageProps) {
  const isCollecting = session.status === 'collecting'
  const sources = session.sources ?? []

  return (
    <div className="space-y-4">
      {isCollecting && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Collecting sources...
        </div>
      )}

      {sources.map((src) => (
        <div key={src.id}>
          <div className="flex items-center gap-2 mb-2">
            <SourceTypeBadge type={src.source_type} />
            <span className="text-sm font-medium">{src.label}</span>
            <span className="text-xs text-muted-foreground">· {src.count} items</span>
          </div>
          <div className="rounded-xl border border-border overflow-hidden">
            {src.items.map((item, i) => (
              <div
                key={i}
                className="flex items-center gap-3 px-4 py-2.5 border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors"
              >
                {item.score != null && (
                  <span className="text-xs text-muted-foreground tabular-nums w-16 shrink-0">
                    {item.extra ?? item.score}
                  </span>
                )}
                <span className="text-sm flex-1 truncate">{item.title}</span>
                {item.url && (
                  <a href={item.url} target="_blank" rel="noopener noreferrer"
                    className="text-muted-foreground hover:text-foreground transition-colors shrink-0">
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}

      {sources.length === 0 && !isCollecting && (
        <p className="text-sm text-muted-foreground">No sources collected.</p>
      )}

      {/* Footer: collection metadata */}
      {sources.length > 0 && (
        <p className="text-xs text-muted-foreground pt-2">
          Collected {new Date(session.created_at).toLocaleString('en-US', {
            month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
          })} · Trigger: {session.trigger_type}
        </p>
      )}
    </div>
  )
}

/** Summary text for collapsed state */
export function collectSummary(session: ContentSession): string {
  const sources = session.sources ?? []
  const total = sources.reduce((sum, s) => sum + s.count, 0)
  const parts = sources.map(s => `${s.label} ${s.count}`).join(', ')
  return `${total} items from ${sources.length} sources (${parts})`
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/stages/CollectStage.tsx
git commit -m "feat(session): add CollectStage component"
```

---

## Task 6: Frontend — AnalyzeStage component

**Files:**
- Create: `web/src/pages/pipelines/session/stages/AnalyzeStage.tsx`

Refactored from the existing `AnalysisPanel` in `SessionDetailModal.tsx`. Adds editable summary/insights persistence and workflow selection from pipeline pool.

**Step 1: Create the component**

```tsx
import { useState, useCallback, useRef, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import { CheckSquare, Square, Loader2, CheckCircle, XCircle } from 'lucide-react'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { updateSessionAnalysis } from '@/entities/content-session/api'
import type { ContentSession, ContentAngle } from '@/entities/content-session'
import type { PipelineWorkflow } from '@/shared/types'

interface AnalyzeStageProps {
  session: ContentSession
  pipelineWorkflows: PipelineWorkflow[]
  onApprove: (selectedWorkflows: string[]) => void
  onReject: () => void
  isApproving: boolean
  isRejecting: boolean
}

export function AnalyzeStage({
  session, pipelineWorkflows, onApprove, onReject, isApproving, isRejecting,
}: AnalyzeStageProps) {
  const [angles, setAngles] = useState<ContentAngle[]>(session.analysis?.angles ?? [])
  const isPending = session.status === 'pending_review'

  // Sync angles when session changes
  useEffect(() => {
    if (session.analysis?.angles) setAngles(session.analysis.angles)
  }, [session.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const toggleAngle = (id: string) => {
    setAngles(prev => prev.map(a => a.id === id ? { ...a, selected: !a.selected } : a))
  }

  // Editable summary/insights with auto-save
  const summaryRef = useRef<HTMLDivElement>(null)
  const insightsRef = useRef<(HTMLDivElement | null)[]>([])
  const saveTimer = useRef<ReturnType<typeof setTimeout>>()

  const saveMutation = useMutation({
    mutationFn: (data: { summary: string; insights: string[] }) =>
      updateSessionAnalysis(session.id, data),
  })

  const scheduleSave = useCallback(() => {
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      const summary = summaryRef.current?.textContent ?? ''
      const insights = insightsRef.current
        .filter((el): el is HTMLDivElement => el !== null)
        .map(el => el.textContent ?? '')
        .filter(Boolean)
      saveMutation.mutate({ summary, insights })
    }, 1500)
  }, [session.id]) // eslint-disable-line react-hooks/exhaustive-deps

  if (!session.analysis) {
    return <p className="text-sm text-muted-foreground">Analyzing sources...</p>
  }

  const { analysis } = session
  const selectedWorkflows = angles.filter(a => a.selected).map(a => {
    // Map angle format to pipeline workflow name
    const match = pipelineWorkflows.find(pw =>
      pw.label?.toLowerCase() === a.format.toLowerCase() ||
      pw.workflow_name.toLowerCase().includes(a.format.toLowerCase())
    )
    return match?.workflow_name ?? a.format
  })

  return (
    <div className="space-y-6">
      {/* Score bar */}
      <div className="flex items-center gap-3 px-4 py-3 rounded-xl border border-border bg-card">
        <ScoreIndicator score={analysis.score} />
        <div className="flex-1 h-2 rounded-full bg-muted overflow-hidden">
          <div
            className="h-full rounded-full bg-success transition-all duration-500"
            style={{ width: `${analysis.score}%` }}
          />
        </div>
        <span className="text-xs text-muted-foreground">
          {analysis.total_collected} collected → {analysis.total_selected} selected
        </span>
      </div>

      {/* Two-column: raw sources ↔ AI summary */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 min-h-[400px]">
        {/* Left: Raw sources (read-only) */}
        <div className="flex flex-col rounded-xl border border-border bg-card overflow-hidden">
          <div className="bg-muted/50 border-b border-border px-4 py-2 flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">Original Sources</span>
            <span className="text-[10px] bg-background px-2 py-0.5 rounded-md border border-border text-muted-foreground">Read-only</span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 text-sm text-muted-foreground leading-relaxed font-mono whitespace-pre-wrap">
            {(session.sources ?? []).flatMap(s => s.items).slice(0, 10).map((item, idx) => (
              <div key={idx} className="mb-2">
                <span className="text-foreground/70">[{idx + 1}]</span> {item.title}
                {item.url && <span className="text-muted-foreground/50 text-xs block ml-4">{item.url}</span>}
              </div>
            ))}
          </div>
        </div>

        {/* Right: AI summary (editable) */}
        <div className="flex flex-col rounded-xl border border-success/30 bg-success/5 overflow-hidden">
          <div className="bg-success/10 border-b border-success/20 px-4 py-2 flex items-center justify-between">
            <span className="text-xs font-medium text-success-foreground">AI Summary & Insights</span>
            {saveMutation.isPending && <Loader2 className="h-3 w-3 animate-spin text-success" />}
            {saveMutation.isSuccess && <CheckCircle className="h-3 w-3 text-success" />}
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            <div>
              <h3 className="text-xs font-semibold text-success-foreground mb-1.5 uppercase tracking-wide">Summary</h3>
              <div
                ref={summaryRef}
                contentEditable={isPending}
                suppressContentEditableWarning
                onInput={scheduleSave}
                className="rounded-lg border border-success/20 bg-background/50 px-3 py-2 text-sm leading-relaxed text-foreground outline-none focus:ring-2 focus:ring-success/50"
              >
                {analysis.summary}
              </div>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-success-foreground mb-1.5 uppercase tracking-wide">Key Insights</h3>
              <ul className="space-y-1.5">
                {analysis.insights.map((insight, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <span className="text-success mt-0.5 shrink-0">•</span>
                    <div
                      ref={el => { insightsRef.current[i] = el }}
                      contentEditable={isPending}
                      suppressContentEditableWarning
                      onInput={scheduleSave}
                      className="flex-1 outline-none text-foreground/90 focus:text-foreground"
                    >
                      {insight}
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </div>

      {/* Workflow selection */}
      <div>
        <h3 className="text-sm font-semibold mb-2 ml-1">Recommended Workflows</h3>
        <div className="rounded-xl border border-border overflow-hidden bg-card">
          {angles.length === 0 && (
            <p className="px-4 py-3 text-sm text-muted-foreground">No workflows recommended.</p>
          )}
          {angles.map((angle) => (
            <button
              key={angle.id}
              onClick={() => isPending && toggleAngle(angle.id)}
              disabled={!isPending}
              className={`w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0
                hover:bg-muted/30 transition-colors text-left cursor-pointer disabled:cursor-default
                ${angle.selected ? 'bg-primary/5' : ''}`}
            >
              {angle.selected
                ? <CheckSquare className="h-4 w-4 text-primary shrink-0" />
                : <Square className="h-4 w-4 text-muted-foreground shrink-0" />}
              <span className="text-xs font-mono text-muted-foreground w-16 shrink-0">{angle.format}</span>
              <span className={`text-sm ${angle.selected ? 'text-foreground font-medium' : 'text-foreground/80'}`}>
                {angle.title}
              </span>
            </button>
          ))}
        </div>
      </div>

      {/* Approval actions */}
      {isPending && (
        <div className="flex items-center justify-between pt-2">
          <p className="text-sm text-muted-foreground">
            {selectedWorkflows.length} workflow(s) selected
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={onReject}
              disabled={isRejecting}
              className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors
                cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <XCircle className="h-4 w-4" /> Reject
            </button>
            <button
              onClick={() => onApprove(selectedWorkflows)}
              disabled={selectedWorkflows.length === 0 || isApproving}
              className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
                bg-success/15 text-success hover:bg-success/25 transition-colors
                cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isApproving
                ? <Loader2 className="h-4 w-4 animate-spin" />
                : <CheckCircle className="h-4 w-4" />}
              Approve & Run {selectedWorkflows.length} Workflows
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/stages/AnalyzeStage.tsx
git commit -m "feat(session): add AnalyzeStage component with editable analysis"
```

---

## Task 7: Frontend — ProduceStage component

**Files:**
- Create: `web/src/pages/pipelines/session/stages/ProduceStage.tsx`

Refactored from the existing `WorkflowExecutionPanel` in `SessionDetailModal.tsx`.

**Step 1: Create the component**

```tsx
import { Loader2, ExternalLink, CheckCircle2, XCircle } from 'lucide-react'
import type { ContentSession } from '@/entities/content-session'

interface ProduceStageProps {
  session: ContentSession
}

const STATUS_LABEL: Record<string, string> = {
  pending: 'Pending', running: 'Running...', success: 'Completed', failed: 'Failed',
}

const STATUS_COLOR: Record<string, string> = {
  success: 'text-success bg-success/10 border-success/20',
  failed: 'text-destructive bg-destructive/10 border-destructive/20',
  running: 'text-info bg-info/10 border-info/20',
  pending: 'text-muted-foreground bg-muted/20 border-border',
}

export function ProduceStage({ session }: ProduceStageProps) {
  const results = session.workflow_results ?? []

  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center border border-dashed border-border rounded-xl">
        <Loader2 className="h-6 w-6 text-muted-foreground/40 mb-3 animate-spin" />
        <h3 className="text-sm font-semibold">Waiting for Approval</h3>
        <p className="text-xs text-muted-foreground mt-1">
          Approve the analysis to trigger workflow execution.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-1">
        <h3 className="text-sm font-semibold">
          Workflow Monitor
          <span className="text-muted-foreground font-normal ml-2">({results.length} active)</span>
        </h3>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {results.map((result, idx) => {
          const isDone = result.status === 'success' || result.status === 'failed'
          const borderHighlight = result.status === 'running'
            ? 'border-info/50 shadow-[0_0_12px_rgba(59,130,246,0.1)]'
            : 'border-border/50'

          return (
            <div key={result.run_id} className={`flex flex-col rounded-xl border bg-card overflow-hidden shadow-sm transition-all ${borderHighlight}`}>
              {/* Header */}
              <div className="bg-muted/30 border-b border-border px-4 py-2.5 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="size-5 rounded-md bg-background flex items-center justify-center shadow-sm border border-black/5">
                    <span className="text-[10px] font-bold">{idx + 1}</span>
                  </div>
                  <span className="text-sm font-bold truncate max-w-[180px]">{result.workflow_name}</span>
                </div>
                <div className={`px-2 py-0.5 rounded-md border text-[10px] font-bold uppercase tracking-wider flex items-center gap-1.5 ${STATUS_COLOR[result.status] ?? STATUS_COLOR.pending}`}>
                  {result.status === 'running' && <div className="w-1.5 h-1.5 rounded-full bg-info animate-pulse" />}
                  {STATUS_LABEL[result.status] ?? result.status}
                </div>
              </div>

              {/* Body */}
              <div className="flex-1 flex flex-col p-3 space-y-3">
                <div className="flex-1 rounded-lg border border-black/5 bg-background/50 shadow-inner p-3 font-mono text-xs overflow-y-auto h-[100px]">
                  {result.status === 'pending' && (
                    <span className="text-muted-foreground">Waiting for agent to initialize...</span>
                  )}
                  {result.status === 'running' && (
                    <div className="space-y-1.5">
                      <div className="text-muted-foreground">[System] Injecting context... <span className="text-success">OK</span></div>
                      <div className="flex items-center gap-2 text-foreground">
                        <Loader2 className="w-3 h-3 animate-spin text-info" /> Generating content...
                      </div>
                    </div>
                  )}
                  {isDone && (
                    <div className="space-y-1">
                      {result.status === 'success'
                        ? <div className="text-success flex items-center gap-1.5"><CheckCircle2 className="h-3.5 w-3.5" /> Completed successfully</div>
                        : <div className="text-destructive flex items-center gap-1.5"><XCircle className="h-3.5 w-3.5" /> Execution failed</div>}
                    </div>
                  )}
                </div>

                <div className="flex items-center justify-between pt-1">
                  <span className="text-[10px] text-muted-foreground font-mono">
                    {result.run_id.slice(0, 8)}
                  </span>
                  {result.output_url && result.status === 'success' ? (
                    <a href={result.output_url} target="_blank" rel="noopener noreferrer"
                      className="flex items-center gap-1 text-xs font-semibold px-3 py-1.5 rounded-lg bg-foreground text-background hover:opacity-90 transition-opacity">
                      <ExternalLink className="h-3 w-3" /> Preview
                    </a>
                  ) : (
                    <button disabled className="text-xs px-3 py-1.5 rounded-lg border border-border text-muted-foreground opacity-50 cursor-not-allowed">
                      Preview
                    </button>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

/** Summary text for collapsed state */
export function produceSummary(session: ContentSession): string {
  const results = session.workflow_results ?? []
  const done = results.filter(r => r.status === 'success').length
  const failed = results.filter(r => r.status === 'failed').length
  return `${done}/${results.length} completed${failed ? `, ${failed} failed` : ''}`
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/stages/ProduceStage.tsx
git commit -m "feat(session): add ProduceStage component"
```

---

## Task 8: Frontend — PublishStage component

**Files:**
- Create: `web/src/pages/pipelines/session/stages/PublishStage.tsx`

Refactored from the existing `PublishPanel` in `SessionDetailModal.tsx`. Adds real content preview and 2nd approval gate.

**Step 1: Create the component**

```tsx
import { useState } from 'react'
import { CheckCircle, CheckSquare, Square, Loader2 } from 'lucide-react'
import type { ContentSession, WorkflowResult } from '@/entities/content-session'

interface PublishStageProps {
  session: ContentSession
  onPublish: (approvedRunIds: string[]) => void
  onRejectAll: () => void
  isPublishing: boolean
}

export function PublishStage({ session, onPublish, onRejectAll, isPublishing }: PublishStageProps) {
  const results = (session.workflow_results ?? []).filter(r => r.status === 'success')
  const [approved, setApproved] = useState<Set<string>>(new Set(results.map(r => r.run_id)))

  const toggleApproval = (runId: string) => {
    setApproved(prev => {
      const next = new Set(prev)
      if (next.has(runId)) next.delete(runId)
      else next.add(runId)
      return next
    })
  }

  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center border border-dashed border-border rounded-xl">
        <CheckSquare className="w-8 h-8 text-muted-foreground/30 mb-3" />
        <h3 className="text-sm font-semibold text-muted-foreground">No content ready</h3>
        <p className="text-xs text-muted-foreground mt-1">Wait for workflows to complete.</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold">Review & Publish</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Review generated content before publishing.
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {results.map((result) => {
          const isApproved = approved.has(result.run_id)
          return (
            <div key={result.run_id} className={`rounded-xl border overflow-hidden shadow-sm transition-all
              ${isApproved ? 'border-success/30 bg-success/5' : 'border-border bg-card'}`}>
              {/* Header */}
              <div className="bg-muted/30 border-b border-border px-4 py-3 flex items-center justify-between">
                <span className="text-sm font-bold">{result.workflow_name}</span>
                <span className="text-[10px] bg-background px-2 py-0.5 rounded-md border text-muted-foreground">
                  Draft
                </span>
              </div>

              {/* Content preview */}
              <div className="p-4 min-h-[120px]">
                <div className="rounded-lg border border-border bg-background p-3 text-sm leading-relaxed min-h-[80px]">
                  <span className="text-muted-foreground italic text-xs">
                    Content preview will appear here when workflow output is available.
                  </span>
                </div>
              </div>

              {/* Actions */}
              <div className="border-t border-border p-3 flex items-center gap-2">
                <button
                  onClick={() => toggleApproval(result.run_id)}
                  className={`flex items-center gap-1.5 flex-1 justify-center py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer
                    ${isApproved
                      ? 'bg-success/15 text-success hover:bg-success/25'
                      : 'border border-border text-muted-foreground hover:bg-muted'}`}
                >
                  {isApproved
                    ? <><CheckSquare className="h-3.5 w-3.5" /> Approved</>
                    : <><Square className="h-3.5 w-3.5" /> Approve</>}
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {/* Publish all */}
      <div className="flex items-center justify-between pt-2">
        <p className="text-sm text-muted-foreground">{approved.size} of {results.length} approved</p>
        <div className="flex items-center gap-2">
          <button
            onClick={onRejectAll}
            className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
          >
            Reject All
          </button>
          <button
            onClick={() => onPublish(Array.from(approved))}
            disabled={approved.size === 0 || isPublishing}
            className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
              bg-success/15 text-success hover:bg-success/25 transition-colors
              cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isPublishing
              ? <Loader2 className="h-4 w-4 animate-spin" />
              : <CheckCircle className="h-4 w-4" />}
            Publish {approved.size} Items
          </button>
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/session/stages/PublishStage.tsx
git commit -m "feat(session): add PublishStage component with 2nd approval gate"
```

---

## Task 9: Frontend — SessionDetailPage (full page assembly)

**Files:**
- Create: `web/src/pages/pipelines/session/SessionDetailPage.tsx`
- Create: `web/src/pages/pipelines/session/index.ts` (barrel export)

This assembles StageSection + StickyProgressBar + all 4 stage components into the full page.

**Step 1: Create the page component**

```tsx
import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { MainLayout } from '@/app/layout'
import { fetchPipeline } from '@/entities/pipeline'
import { fetchContentSession } from '@/entities/content-session/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import { StickyProgressBar, getActiveStage, isStageCompleted } from './StickyProgressBar'
import { StageSection } from './StageSection'
import { CollectStage, collectSummary } from './stages/CollectStage'
import { AnalyzeStage } from './stages/AnalyzeStage'
import { ProduceStage, produceSummary } from './stages/ProduceStage'
import { PublishStage } from './stages/PublishStage'
import type { StageState } from './StageSection'
import type { Stage } from './StickyProgressBar'

function getStageState(stage: Stage, session: { status: string }): StageState {
  const activeStage = getActiveStage(session as any)
  if (isStageCompleted(stage, activeStage)) return 'completed'
  if (stage === activeStage) return 'active'
  // Special: if status is 'published', all stages completed
  if (session.status === 'published') return 'completed'
  return 'locked'
}

export default function SessionDetailPage() {
  const { id: pipelineId, sessionId } = useParams<{ id: string; sessionId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: pipeline } = useQuery({
    queryKey: ['pipeline', pipelineId],
    queryFn: () => fetchPipeline(pipelineId!),
    enabled: !!pipelineId,
  })

  const { data: session, isLoading } = useQuery({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId!),
    enabled: !!sessionId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'collecting' || status === 'producing' ? 3000 : false
    },
  })

  const [isApproving, setIsApproving] = useState(false)
  const [isRejecting, setIsRejecting] = useState(false)
  const [isPublishing, setIsPublishing] = useState(false)

  const { approveSession, rejectSession } = useContentSessionStore()

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
  }

  const handleApprove = async (selectedWorkflows: string[]) => {
    setIsApproving(true)
    try {
      await approveSession(sessionId!, selectedWorkflows)
      invalidate()
    } finally { setIsApproving(false) }
  }

  const handleReject = async () => {
    setIsRejecting(true)
    try {
      await rejectSession(sessionId!)
      invalidate()
      navigate(`/pipelines/${pipelineId}`)
    } finally { setIsRejecting(false) }
  }

  const handlePublish = async (_approvedRunIds: string[]) => {
    setIsPublishing(true)
    try {
      // TODO: call publish endpoint when available
      invalidate()
    } finally { setIsPublishing(false) }
  }

  if (isLoading || !session) {
    return (
      <MainLayout headerContent={<span className="text-muted-foreground text-sm">Loading…</span>}>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout
      headerContent={
        <div className="flex items-center gap-2 overflow-hidden">
          <button
            onClick={() => navigate('/pipelines')}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer shrink-0"
          >
            Pipelines
          </button>
          <span className="text-muted-foreground/40 text-xs">/</span>
          <button
            onClick={() => navigate(`/pipelines/${pipelineId}`)}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer shrink-0"
          >
            {pipeline?.name ?? 'Pipeline'}
          </button>
          <span className="text-muted-foreground/40 text-xs">/</span>
          <span className="font-semibold text-sm truncate">
            Session {session.session_number != null ? `#${session.session_number}` : session.id.slice(0, 8)}
          </span>
        </div>
      }
    >
      <StickyProgressBar session={session} />

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-6 py-6">
          <StageSection
            id="stage-collect"
            number={1}
            title="Collect"
            state={getStageState('collect', session)}
            summary={collectSummary(session)}
            statusText={session.sources ? `${session.sources.reduce((s, src) => s + src.count, 0)} items` : undefined}
          >
            <CollectStage session={session} />
          </StageSection>

          <StageSection
            id="stage-analyze"
            number={2}
            title="Analyze"
            state={getStageState('analyze', session)}
            summary={session.analysis?.summary?.slice(0, 80)}
            statusText={session.analysis ? `Score ${session.analysis.score}` : undefined}
          >
            <AnalyzeStage
              session={session}
              pipelineWorkflows={pipeline?.workflows ?? []}
              onApprove={handleApprove}
              onReject={handleReject}
              isApproving={isApproving}
              isRejecting={isRejecting}
            />
          </StageSection>

          <StageSection
            id="stage-produce"
            number={3}
            title="Produce"
            state={getStageState('produce', session)}
            summary={produceSummary(session)}
            statusText={session.workflow_results ? `${session.workflow_results.length} workflows` : undefined}
          >
            <ProduceStage session={session} />
          </StageSection>

          <StageSection
            id="stage-publish"
            number={4}
            title="Publish"
            state={getStageState('publish', session)}
          >
            <PublishStage
              session={session}
              onPublish={handlePublish}
              onRejectAll={handleReject}
              isPublishing={isPublishing}
            />
          </StageSection>
        </div>
      </div>
    </MainLayout>
  )
}
```

**Step 2: Create barrel export**

Create `web/src/pages/pipelines/session/index.ts`:

```typescript
export { default } from './SessionDetailPage'
```

**Step 3: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/pages/pipelines/session/
git commit -m "feat(session): add SessionDetailPage full-page assembly"
```

---

## Task 10: Frontend — Update router + remove modal

**Files:**
- Modify: `web/src/app/router.tsx`
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

**Step 1: Update the router**

In `web/src/app/router.tsx`, add a lazy import for SessionDetailPage and change the session route:

Find the existing imports section, add:
```typescript
const SessionDetailPage = lazy(() => import('@/pages/pipelines/session'))
```

Find the route `/pipelines/:id/sessions/:sessionId` (currently pointing to `PipelineDetailPage`) and change it to:
```typescript
{ path: '/pipelines/:id/sessions/:sessionId', element: <SessionDetailPage /> },
```

**Step 2: Remove SessionDetailModal from PipelineDetail**

In `web/src/pages/pipelines/PipelineDetail.tsx`:

1. Remove the `sessionId` destructuring from `useParams` (line 354). Change to:
   ```typescript
   const { id } = useParams<{ id: string }>()
   ```

2. Remove the import of `SessionDetailModal` (line 18).

3. Remove the `{sessionId && <SessionDetailModal ... />}` block at the bottom (lines 655-661).

**Step 3: Verify type-check and that the app builds**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/app/router.tsx web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(routing): wire SessionDetailPage, remove SessionDetailModal from PipelineDetail"
```

---

## Task 11: Frontend — WorkflowPicker modal

**Files:**
- Create: `web/src/pages/pipelines/WorkflowPicker.tsx`

A modal to select existing workflows and add them to the pipeline's workflows list.

**Step 1: Create the component**

```tsx
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { X, Search, Plus, Loader2 } from 'lucide-react'
import type { PipelineWorkflow } from '@/shared/types'

// Minimal workflow list item from GET /api/workflows
type WorkflowListItem = {
  name: string
  description?: string
}

interface WorkflowPickerProps {
  existingWorkflows: PipelineWorkflow[]
  onAdd: (workflow: PipelineWorkflow) => void
  onClose: () => void
}

export function WorkflowPicker({ existingWorkflows, onAdd, onClose }: WorkflowPickerProps) {
  const [search, setSearch] = useState('')

  const { data: workflows = [], isLoading } = useQuery<WorkflowListItem[]>({
    queryKey: ['workflows'],
    queryFn: async () => {
      const res = await fetch('/api/workflows')
      if (!res.ok) throw new Error('Failed to fetch workflows')
      return res.json()
    },
  })

  const existingNames = new Set(existingWorkflows.map(w => w.workflow_name))

  const filtered = workflows.filter(w => {
    if (existingNames.has(w.name)) return false
    if (!search) return true
    return w.name.toLowerCase().includes(search.toLowerCase()) ||
      w.description?.toLowerCase().includes(search.toLowerCase())
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-md mx-4 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">Add Workflow</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Search */}
        <div className="px-5 py-3 border-b border-border">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
            <input
              type="search"
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search workflows..."
              autoFocus
              className="w-full pl-8 pr-3 py-2 rounded-lg border border-input bg-background
                text-sm outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
            />
          </div>
        </div>

        {/* List */}
        <div className="max-h-80 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            </div>
          ) : filtered.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-8">
              {search ? 'No matching workflows.' : 'No workflows available.'}
            </p>
          ) : (
            filtered.map(w => (
              <button
                key={w.name}
                onClick={() => {
                  onAdd({ workflow_name: w.name, auto_select: true })
                  onClose()
                }}
                className="w-full flex items-center gap-3 px-5 py-3 border-b border-border last:border-b-0
                  hover:bg-muted/30 transition-colors text-left cursor-pointer"
              >
                <Plus className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                <div className="flex-1 min-w-0">
                  <span className="text-sm font-medium">{w.name}</span>
                  {w.description && (
                    <p className="text-xs text-muted-foreground truncate mt-0.5">{w.description}</p>
                  )}
                </div>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/WorkflowPicker.tsx
git commit -m "feat(pipeline): add WorkflowPicker modal component"
```

---

## Task 12: Frontend — Workflows section in PipelineSettingsPanel

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

Add the Workflows accordion section to the existing `PipelineSettingsPanel` component.

**Step 1: Add state and props**

In `PipelineSettingsPanel`, add to the props interface:
```typescript
workflows: PipelineWorkflow[]
onWorkflowsChange: (w: PipelineWorkflow[]) => void
```

Import the `WorkflowPicker`:
```typescript
import { WorkflowPicker } from './WorkflowPicker'
import type { PipelineWorkflow } from '@/shared/types'
```

Add state inside the component:
```typescript
const [workflowsOpen, setWorkflowsOpen] = useState(false)
const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)
```

**Step 2: Add the Workflows section JSX**

Insert between the Sources & Schedule section and the Editorial Brief section:

```tsx
<section>
  <button
    onClick={() => setWorkflowsOpen(v => !v)}
    className="w-full flex items-center justify-between mb-2 cursor-pointer"
  >
    <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
      Workflows
    </span>
    <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${workflowsOpen ? 'rotate-180' : ''}`} />
  </button>

  {workflowsOpen && (
    <div className="space-y-3">
      {workflows.length === 0 ? (
        <div className="py-4 text-center rounded-xl border border-dashed border-border">
          <p className="text-xs text-muted-foreground mb-2">No workflows configured.</p>
          <button
            onClick={() => setShowWorkflowPicker(true)}
            className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
          >
            <Plus className="h-3 w-3" /> Add workflow
          </button>
        </div>
      ) : (
        <div className="rounded-xl border border-border overflow-hidden">
          {workflows.map((wf, i) => (
            <div key={wf.workflow_name} className="flex items-center gap-2 px-3 py-2.5 border-b border-border last:border-b-0">
              <span className="text-xs font-medium flex-1 truncate">{wf.label || wf.workflow_name}</span>
              <button
                onClick={() => onWorkflowsChange(workflows.filter((_, j) => j !== i))}
                className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
              >
                <Trash2 className="h-3 w-3" />
              </button>
            </div>
          ))}
        </div>
      )}

      {workflows.length > 0 && (
        <button
          onClick={() => setShowWorkflowPicker(true)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        >
          <Plus className="h-3 w-3" /> Add workflow
        </button>
      )}
    </div>
  )}

  {showWorkflowPicker && (
    <WorkflowPicker
      existingWorkflows={workflows}
      onAdd={(wf) => onWorkflowsChange([...workflows, wf])}
      onClose={() => setShowWorkflowPicker(false)}
    />
  )}
</section>
```

**Step 3: Wire in the parent PipelineDetailPage**

In the `PipelineDetailPage` component:

1. Add local state:
   ```typescript
   const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
   ```

2. In the useEffect that syncs pipeline data, add:
   ```typescript
   setLocalWorkflows(pipeline.workflows ?? [])
   ```

3. Add `localWorkflows` to the isDirty check:
   ```typescript
   JSON.stringify(localWorkflows) !== JSON.stringify(pipeline.workflows ?? [])
   ```

4. Add ref: `const localWorkflowsRef = useRef(localWorkflows)` and keep it updated.

5. Include in `doSave`:
   ```typescript
   await updatePipeline(id!, { ...p, sources: localSourcesRef.current, schedule: localScheduleRef.current, workflows: localWorkflowsRef.current })
   ```

6. Pass to the settings panel:
   ```tsx
   <PipelineSettingsPanel
     ...existing props...
     workflows={localWorkflows}
     onWorkflowsChange={setLocalWorkflows}
   />
   ```

**Step 4: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline): add Workflows section to PipelineSettingsPanel"
```

---

## Task 13: Cleanup — Delete SessionDetailModal

**Files:**
- Delete: `web/src/pages/pipelines/SessionDetailModal.tsx`

**Step 1: Verify no remaining imports**

Run: `cd web && grep -r 'SessionDetailModal' src/`
Expected: No results (all references were removed in Task 10)

**Step 2: Delete the file**

```bash
rm web/src/pages/pipelines/SessionDetailModal.tsx
```

**Step 3: Verify type-check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 4: Commit**

```bash
git add -A web/src/pages/pipelines/SessionDetailModal.tsx
git commit -m "refactor: remove SessionDetailModal (replaced by SessionDetailPage)"
```

---

## Task 14: Verify full build

**Step 1: Backend build**

Run: `go build ./cmd/upal`
Expected: PASS

**Step 2: Frontend build**

Run: `cd web && npm run build`
Expected: PASS

**Step 3: Final commit if any fixes needed**

```bash
git add -A && git commit -m "fix: resolve build issues from session UX redesign"
```