# Pipeline Detail UX Redesign — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign `/pipelines/:id` to show sessions as a searchable table with settings in a right panel, and open session detail as a full-screen popup at `/pipelines/:id/sessions/:sessionId`.

**Architecture:** `PipelineDetailPage` reads an optional `sessionId` URL param (via nested route) and shows `SessionDetailModal` as an overlay when present. The right panel (`MainLayout`'s existing `rightPanel` prop) holds pipeline settings (previously in accordion). `/inbox` and `/inbox/:sessionId` routes are deleted.

**Tech Stack:** React Router v6 nested routes, React Query, Tailwind v4, existing `MainLayout` `rightPanel` prop

---

### Task 1: Remove `/inbox` from nav and router

**Files:**
- Modify: `web/src/app/layout.tsx`
- Modify: `web/src/app/router.tsx`

**Step 1: Remove Inbox from NAV_ITEMS in layout.tsx**

In `web/src/app/layout.tsx`, remove the `Inbox` nav item and its badge logic.

Current `NAV_ITEMS`:
```tsx
const NAV_ITEMS = [
    { icon: Box, label: 'Workflows', to: '/workflows' },
    { icon: Workflow, label: 'Pipelines', to: '/pipelines' },
    { icon: Inbox, label: 'Inbox', to: '/inbox' },      // ← remove this
    { icon: Globe, label: 'Published', to: '/published' },
    { icon: Activity, label: 'Runs', to: '/runs' },
    { icon: Zap, label: 'Connections', to: '/connections' },
];
```

After removing:
```tsx
const NAV_ITEMS = [
    { icon: Box, label: 'Workflows', to: '/workflows' },
    { icon: Workflow, label: 'Pipelines', to: '/pipelines' },
    { icon: Globe, label: 'Published', to: '/published' },
    { icon: Activity, label: 'Runs', to: '/runs' },
    { icon: Zap, label: 'Connections', to: '/connections' },
];
```

Also remove the `Inbox` import from lucide-react and remove `pendingCount` usage (the `showBadge` logic references `/inbox`):
- Remove `Inbox` from import
- Remove `const pendingCount = useContentSessionStore((s) => s.pendingCount);`
- Remove `import { useContentSessionStore }` if no longer used
- Remove `const showBadge = item.to === '/inbox' && pendingCount && pendingCount > 0;` — simplify to `const showBadge = false;` or remove the badge block entirely

**Step 2: Update router.tsx — remove inbox, add nested session route**

```tsx
// web/src/app/router.tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import ProductLandingPage from '@/pages/landing/ProductLanding'
import LandingPage from '@/pages/landing'
import EditorPage from '@/pages/Editor'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import PipelineDetailPage from '@/pages/pipelines/PipelineDetail'
import PipelineNewPage from '@/pages/pipelines/PipelineNew'
import ConnectionsPage from '@/pages/connections'
import { RunDetail } from '@/widgets/run-detail'
import PublishedPage from '@/pages/Published'
import { ErrorBoundary } from '@/shared/ui/ErrorBoundary'
import { ToastContainer } from '@/shared/ui/ToastContainer'

export function AppRouter() {
  return (
    <ErrorBoundary>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<ProductLandingPage />} />
          <Route path="/workflows" element={<LandingPage />} />
          <Route path="/editor" element={<EditorPage />} />
          <Route path="/runs" element={<RunsPage />} />
          <Route path="/runs/:id" element={<RunDetail />} />
          <Route path="/connections" element={<ConnectionsPage />} />

          {/* Pipelines */}
          <Route path="/pipelines" element={<PipelinesPage />} />
          <Route path="/pipelines/new" element={<PipelineNewPage />} />
          <Route path="/pipelines/:id" element={<PipelineDetailPage />} />
          <Route path="/pipelines/:id/sessions/:sessionId" element={<PipelineDetailPage />} />

          {/* Content */}
          <Route path="/published" element={<PublishedPage />} />
        </Routes>
      </BrowserRouter>
      <ToastContainer />
    </ErrorBoundary>
  )
}
```

Key change: `PipelineDetailPage` handles BOTH `/pipelines/:id` and `/pipelines/:id/sessions/:sessionId`. It will read `sessionId` from params and show the modal when present.

**Step 3: Verify app compiles**

```bash
cd web && npx tsc -b 2>&1 | head -30
```

Expected: Only errors from removed imports in router (InboxPage, SessionDetailPage) — those files will be deleted in Task 5.

**Step 4: Commit**

```bash
git add web/src/app/router.tsx web/src/app/layout.tsx
git commit -m "feat(ux): remove /inbox from nav and add /pipelines/:id/sessions/:sessionId route"
```

---

### Task 2: Create `SessionDetailModal` component

**Files:**
- Create: `web/src/pages/pipelines/SessionDetailModal.tsx`

This extracts the logic from `web/src/pages/inbox/SessionDetail.tsx` into a reusable modal.

**Step 1: Write the component**

```tsx
// web/src/pages/pipelines/SessionDetailModal.tsx
import { useState, useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  X, ExternalLink, CheckCircle, XCircle, CheckSquare, Square, Loader2,
} from 'lucide-react'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { fetchContentSession } from '@/entities/content-session/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { ContentSession, ContentAngle } from '@/entities/content-session'

// ─── Re-use panels from SessionDetail ─────────────────────────────────────────
// (Copy SourcePanel, AnalysisPanel, WorkflowExecutionPanel, PublishPanel,
//  and RejectDialog verbatim from web/src/pages/inbox/SessionDetail.tsx)

// ─── Source panel ─────────────────────────────────────────────────────────────

function SourcePanel({ session }: { session: ContentSession }) {
  const isCollecting = session.status === 'collecting'
  const sources = session.sources ?? []
  return (
    <div className="space-y-6">
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
    </div>
  )
}

// ─── Analysis panel ───────────────────────────────────────────────────────────

function AnalysisPanel({
  session, angles, onToggleAngle,
}: {
  session: ContentSession
  angles: ContentAngle[]
  onToggleAngle: (id: string) => void
}) {
  if (!session.analysis) return <p className="text-sm text-muted-foreground">No analysis results.</p>
  const { analysis } = session
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Total Collected: <span className="text-foreground">{analysis.total_collected}</span>
          {' → '}Selected: <span className="text-foreground">{analysis.total_selected}</span>
        </p>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 h-[500px]">
        <div className="flex flex-col rounded-xl border border-border bg-card overflow-hidden shadow-sm">
          <div className="bg-muted/50 border-b border-border px-4 py-2 flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">Original Sources (Raw)</span>
            <span className="text-[10px] bg-background px-2 py-0.5 rounded-md border border-border text-muted-foreground">Read-only</span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 text-sm text-muted-foreground leading-relaxed font-mono whitespace-pre-wrap">
            {(session.sources ?? []).flatMap(s => s.items).slice(0, 5).map((item, idx) => (
              `<article id="${idx}">\n  <title>${item.title}</title>\n  <source>${item.url || 'unknown'}</source>\n</article>\n\n`
            ))}
          </div>
        </div>
        <div className="flex flex-col rounded-xl border border-success/30 bg-success/5 overflow-hidden shadow-sm">
          <div className="bg-success/10 border-b border-success/20 px-4 py-2 flex items-center justify-between">
            <span className="text-xs font-medium text-success-foreground">AI Generated Summary &amp; Insights</span>
            <span className="flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-2 w-2 rounded-full bg-success opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-success" />
            </span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-5">
            <div>
              <h3 className="text-xs font-semibold text-success-foreground mb-1.5 uppercase tracking-wide">Summary</h3>
              <div className="rounded-lg border border-success/20 bg-background/50 px-3 py-2 text-sm leading-relaxed text-foreground focus-within:ring-2 focus-within:ring-success/50 outline-none" contentEditable suppressContentEditableWarning>
                {analysis.summary}
              </div>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-success-foreground mb-1.5 uppercase tracking-wide">Key Insights</h3>
              <ul className="space-y-1.5">
                {analysis.insights.map((insight, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <span className="text-success mt-0.5 shrink-0">•</span>
                    <div className="flex-1 outline-none text-foreground/90 focus-within:text-foreground" contentEditable suppressContentEditableWarning>{insight}</div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </div>
      <div>
        <h3 className="text-sm font-semibold mb-2 mt-4 ml-1">Recommended Workflows (Content Formats)</h3>
        <div className="rounded-xl border border-border overflow-hidden bg-card">
          {angles.map((angle) => (
            <button key={angle.id} onClick={() => onToggleAngle(angle.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors text-left cursor-pointer ${angle.selected ? 'bg-primary/5' : ''}`}
            >
              {angle.selected ? <CheckSquare className="h-4 w-4 text-primary shrink-0" /> : <Square className="h-4 w-4 text-muted-foreground shrink-0" />}
              <span className="text-xs font-mono text-muted-foreground w-12 shrink-0">{angle.format}</span>
              <span className={`text-sm ${angle.selected ? 'text-foreground font-medium' : 'text-foreground/80'}`}>{angle.title}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

function WorkflowExecutionPanel({ session }: { session: ContentSession }) {
  const results = session.workflow_results ?? []
  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-border rounded-2xl">
        <Loader2 className="h-8 w-8 text-muted-foreground/50 mb-4 animate-spin" />
        <h3 className="text-sm font-semibold text-foreground">Waiting for Workflows</h3>
        <p className="text-xs text-muted-foreground mt-1">Accept the analysis to trigger parallel workflows.</p>
      </div>
    )
  }
  const statusLabel: Record<string, string> = { pending: 'Pending', running: 'Running...', success: 'Completed', failed: 'Failed' }
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold">Workflow Execution Monitor <span className="text-muted-foreground font-normal ml-2">({results.length} active)</span></h3>
        <button className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1">
          <ExternalLink className="w-3.5 h-3.5" /> Open Global Logs
        </button>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {results.map((result, idx) => {
          const isDone = result.status === 'success' || result.status === 'failed'
          const statusColor = result.status === 'success' ? 'text-success bg-success/10 border-success/20'
            : result.status === 'failed' ? 'text-destructive bg-destructive/10 border-destructive/20'
            : result.status === 'running' ? 'text-info bg-info/10 border-info/20'
            : 'text-muted-foreground bg-muted/20 border-border'
          const borderHighlight = result.status === 'running' ? 'border-info/50 shadow-[0_0_15px_rgba(59,130,246,0.15)]' : 'border-border/50'
          return (
            <div key={result.run_id} className={`flex flex-col rounded-xl border bg-card overflow-hidden shadow-sm transition-all ${borderHighlight}`}>
              <div className="bg-muted/30 border-b border-border px-4 py-2.5 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="size-5 rounded-md bg-background flex items-center justify-center shadow-sm border border-black/5">
                    <span className="text-[10px] font-bold text-foreground">{idx + 1}</span>
                  </div>
                  <span className="text-sm font-bold tracking-tight text-foreground truncate max-w-[150px]">{result.workflow_name}</span>
                </div>
                <div className={`px-2 py-0.5 rounded-md border text-[10px] font-bold uppercase tracking-wider flex items-center gap-1.5 ${statusColor}`}>
                  {result.status === 'running' && <div className="w-1.5 h-1.5 rounded-full bg-info animate-pulse" />}
                  {statusLabel[result.status] ?? result.status}
                </div>
              </div>
              <div className="flex-1 flex flex-col p-3 bg-background/50 space-y-3">
                <div className="flex-1 rounded-lg border border-black/5 bg-background shadow-inner p-3 font-mono text-xs overflow-y-auto h-[120px]">
                  {result.status === 'pending' && <span className="text-muted-foreground">Waiting for agent to initialize...</span>}
                  {result.status === 'running' && (
                    <div className="space-y-1.5">
                      <div className="text-muted-foreground">{"[System]"} Injecting context parameters... <span className="text-success">OK</span></div>
                      <div className="text-muted-foreground">{"[Agent]"} Planning execution steps... <span className="text-success">OK</span></div>
                      <div className="text-foreground flex items-center gap-2"><Loader2 className="w-3 h-3 animate-spin text-info" /> Generating content formatting...</div>
                    </div>
                  )}
                  {isDone && (
                    <div className="space-y-1">
                      <div className="text-success">Execution finished successfully in 4.2s.</div>
                      <div className="text-muted-foreground mt-2">Tokens used: 1,240 input / 450 output</div>
                    </div>
                  )}
                </div>
                <div className="flex items-center justify-between pt-1">
                  <span className="text-[10px] text-muted-foreground font-mono">Run ID: {result.run_id.slice(0, 8)}</span>
                  {result.output_url && result.status === 'success' ? (
                    <button className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-foreground text-background hover:opacity-90 transition-opacity">Preview Output</button>
                  ) : (
                    <button disabled className="text-xs font-semibold px-3 py-1.5 rounded-lg border border-border text-muted-foreground opacity-50 cursor-not-allowed">Preview Output</button>
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

function PublishPanel({ session }: { session: ContentSession }) {
  const results = session.workflow_results ?? []
  const hasSuccess = results.some(r => r.status === 'success')
  if (!hasSuccess) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center border border-border bg-card rounded-2xl">
        <CheckSquare className="w-12 h-12 text-muted-foreground/30 mb-4" />
        <h3 className="text-lg font-semibold mb-2 text-muted-foreground">No content ready for publishing</h3>
        <p className="text-sm text-muted-foreground/80">Wait for the workflows to complete successfully.</p>
      </div>
    )
  }
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-sm font-semibold">Dry Run / Preview</h3>
          <p className="text-xs text-muted-foreground mt-0.5">Review the generated content before final dispatch.</p>
        </div>
        <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-success/15 text-success hover:bg-success/20 transition-colors cursor-pointer text-xs font-semibold">
          <CheckCircle className="h-3.5 w-3.5" /> Publish All Approvals
        </button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {results.filter(r => r.status === 'success').map((result) => (
          <div key={result.run_id} className="rounded-xl border border-border bg-card overflow-hidden shadow-sm flex flex-col">
            <div className="bg-muted/30 border-b border-border px-4 py-3 flex items-center justify-between">
              <span className="text-sm font-bold text-foreground">{result.workflow_name}</span>
              <span className="text-[10px] bg-background px-2 py-0.5 rounded-md border text-muted-foreground">Draft Mode</span>
            </div>
            <div className="p-4 flex-1">
              <div className="flex items-start gap-3 mb-3">
                <div className="w-10 h-10 rounded-full bg-muted animate-pulse shrink-0" />
                <div className="space-y-1.5 flex-1">
                  <div className="h-3.5 bg-muted rounded animate-pulse w-1/3" />
                  <div className="h-2.5 bg-muted rounded animate-pulse w-1/4" />
                </div>
              </div>
              <div className="space-y-2 mb-4">
                <div className="h-3 bg-muted rounded animate-pulse w-full" />
                <div className="h-3 bg-muted rounded animate-pulse w-full" />
                <div className="h-3 bg-muted rounded animate-pulse w-5/6" />
                <div className="h-3 bg-muted rounded animate-pulse w-4/6" />
              </div>
              <div className="rounded-lg aspect-video bg-muted/50 border border-border flex items-center justify-center">
                <span className="text-xs font-mono text-muted-foreground/50">Generated Media / Link Card Preview</span>
              </div>
            </div>
            <div className="border-t border-border bg-muted/10 p-3 flex gap-2">
              <button className="flex-1 text-xs font-medium py-1.5 rounded-lg border border-border hover:bg-muted transition-colors">Edit Content</button>
              <button className="flex-[2] text-xs font-medium py-1.5 rounded-lg bg-foreground text-background hover:opacity-90 transition-opacity">Schedule Post</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function RejectDialog({ onConfirm, onCancel, isLoading }: { onConfirm: (reason: string) => void; onCancel: () => void; isLoading: boolean }) {
  const [reason, setReason] = useState('')
  return (
    <div className="fixed inset-0 z-[60] flex items-end sm:items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative bg-card border border-border rounded-2xl p-5 w-full max-w-sm mx-4 shadow-xl">
        <h3 className="text-sm font-semibold mb-1">Reject Session</h3>
        <p className="text-xs text-muted-foreground mb-3">Enter a reason (optional)</p>
        <textarea value={reason} onChange={(e) => setReason(e.target.value)}
          placeholder="e.g. Low relevance, reschedule for next week..." rows={3}
          className="w-full rounded-xl border border-border bg-muted/20 px-3 py-2 text-sm placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-border resize-none mb-3"
        />
        <div className="flex items-center justify-end gap-2">
          <button onClick={onCancel} className="px-3 py-1.5 rounded-lg text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer">Cancel</button>
          <button onClick={() => onConfirm(reason)} disabled={isLoading}
            className="flex items-center gap-1.5 px-4 py-1.5 rounded-xl text-sm font-medium bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed">
            {isLoading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            Confirm Reject
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Modal ────────────────────────────────────────────────────────────────────

type Stage = 'collect' | 'analyze' | 'workflow' | 'publish'

interface SessionDetailModalProps {
  sessionId: string
  pipelineId: string
  onClose: () => void
}

export function SessionDetailModal({ sessionId, pipelineId, onClose }: SessionDetailModalProps) {
  const queryClient = useQueryClient()
  const [activeStage, setActiveStage] = useState<Stage>('collect')
  const [showRejectDialog, setShowRejectDialog] = useState(false)
  const [angles, setAngles] = useState<ContentAngle[]>([])
  const [isApproving, setIsApproving] = useState(false)
  const [isRejecting, setIsRejecting] = useState(false)

  const { data: session, isLoading } = useQuery({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId),
    enabled: !!sessionId,
    refetchInterval: (query) =>
      query.state.data?.status === 'collecting' ? 3000 : false,
  })

  useEffect(() => {
    if (session?.analysis?.angles) {
      setAngles(session.analysis.angles)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.id])

  // Close on Escape key
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const { approveSession, rejectSession } = useContentSessionStore()

  const handleApprove = async () => {
    setIsApproving(true)
    try {
      await approveSession(sessionId, angles.filter((a) => a.selected).map((a) => a.id))
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      onClose()
    } finally {
      setIsApproving(false)
    }
  }

  const handleReject = async (reason: string) => {
    setIsRejecting(true)
    try {
      await rejectSession(sessionId, reason || undefined)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      onClose()
    } finally {
      setIsRejecting(false)
    }
  }

  const selectedCount = angles.filter((a) => a.selected).length
  const isPending = session?.status === 'pending_review'

  const stages: { value: Stage; label: string }[] = [
    { value: 'collect', label: '1. Collect' },
    { value: 'analyze', label: '2. AI Analyze' },
    { value: 'workflow', label: '3. Workflow' },
    { value: 'publish', label: '4. Publish' },
  ]

  const createdAt = session
    ? new Date(session.created_at).toLocaleString('en-US', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
      })
    : ''

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      {/* Modal panel */}
      <div className="relative z-10 bg-background border border-border rounded-2xl shadow-2xl
        w-full max-w-5xl mx-4 max-h-[90vh] overflow-hidden flex flex-col
        animate-in fade-in slide-in-from-bottom-4 duration-300">

        {/* Header */}
        <div className="flex items-start justify-between gap-4 px-6 py-4 border-b border-border shrink-0">
          {isLoading || !session ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="text-sm">Loading session...</span>
            </div>
          ) : (
            <div>
              <div className="flex items-center gap-2 flex-wrap mb-0.5">
                <h2 className="text-base font-semibold">
                  {session.session_number != null ? `Session ${session.session_number}` : session.id}
                </h2>
                <StatusBadge status={session.status} />
                {session.analysis && <ScoreIndicator score={session.analysis.score} />}
              </div>
              <p className="text-xs text-muted-foreground">
                {createdAt} · trigger: {session.trigger_type}
              </p>
            </div>
          )}
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer shrink-0"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {session && (
          <>
            {/* Stage stepper */}
            <div className="px-6 pt-4 pb-2 border-b border-border shrink-0">
              <div className="flex items-center gap-2 relative">
                <div className="absolute left-[10%] right-[10%] top-1/2 -translate-y-1/2 h-0.5 bg-border/40 z-0" />
                {stages.map((stage, idx) => {
                  const isActive = activeStage === stage.value
                  const isPast = stages.findIndex(s => s.value === activeStage) > idx
                  return (
                    <button
                      key={stage.value}
                      onClick={() => setActiveStage(stage.value)}
                      className={`relative z-10 flex-1 flex flex-col items-center justify-center p-2.5 rounded-xl border transition-all duration-300 cursor-pointer text-sm font-bold tracking-tight
                        ${isActive
                          ? 'bg-foreground border-foreground text-background shadow-lg shadow-black/20 scale-105'
                          : isPast
                          ? 'glass-panel border-success/30 text-success'
                          : 'glass-panel border-white/5 text-muted-foreground hover:bg-white/5'
                        }`}
                    >
                      {stage.label}
                      {isActive && <div className="absolute -bottom-1.5 w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />}
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Stage content */}
            <div className="flex-1 overflow-y-auto px-6 py-5 animate-in fade-in slide-in-from-bottom-2 duration-300">
              {activeStage === 'collect' && <SourcePanel session={session} />}
              {activeStage === 'analyze' && (
                <AnalysisPanel session={session} angles={angles} onToggleAngle={(id) => setAngles(prev => prev.map(a => a.id === id ? { ...a, selected: !a.selected } : a))} />
              )}
              {activeStage === 'workflow' && <WorkflowExecutionPanel session={session} />}
              {activeStage === 'publish' && <PublishPanel session={session} />}
            </div>

            {/* Bottom action bar */}
            {isPending && (
              <div className="border-t border-border bg-background/95 backdrop-blur-sm px-6 py-3 shrink-0 flex items-center justify-between gap-4">
                <p className="text-sm text-muted-foreground">{selectedCount} format(s) selected</p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setShowRejectDialog(true)}
                    disabled={isRejecting}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <XCircle className="h-4 w-4" /> Reject
                  </button>
                  <button
                    onClick={() => void handleApprove()}
                    disabled={selectedCount === 0 || isApproving}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-success/15 text-success hover:bg-success/25 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {isApproving ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}
                    Approve &amp; Run Selected Workflows
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {showRejectDialog && (
        <RejectDialog
          onConfirm={(reason) => void handleReject(reason)}
          onCancel={() => setShowRejectDialog(false)}
          isLoading={isRejecting}
        />
      )}
    </div>
  )
}
```

**Step 2: Type-check the new file**

```bash
cd web && npx tsc -b 2>&1 | grep SessionDetailModal
```

Expected: No errors related to `SessionDetailModal.tsx`

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/SessionDetailModal.tsx
git commit -m "feat(pipeline): add SessionDetailModal component"
```

---

### Task 3: Redesign `PipelineDetailPage`

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

Complete replacement of the file. Key changes:
- Session table instead of cards
- Search input
- Right panel via `MainLayout`'s `rightPanel` prop
- `SessionDetailModal` opened when `sessionId` param present
- `collectMutation` navigates to `/pipelines/${id}/sessions/${session_id}`

**Step 1: Write the new PipelineDetail.tsx**

```tsx
import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft, Play, Loader2, Trash2, Plus, Clock, Database, RefreshCw,
  Check, X, Search, CheckCircle2, XCircle, ChevronDown,
} from 'lucide-react'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { MainLayout } from '@/app/layout'
import { fetchPipeline, updatePipeline, collectPipeline } from '@/entities/pipeline'
import { fetchContentSessions } from '@/entities/content-session/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import { SessionDetailModal } from './SessionDetailModal'
import type { PipelineSource, PipelineContext } from '@/shared/types'
import type { ContentSession } from '@/entities/content-session/types'

// ─── Schedule presets ─────────────────────────────────────────────────────────

const SCHEDULE_PRESETS: { label: string; cron: string }[] = [
  { label: 'Every hour', cron: '0 * * * *' },
  { label: 'Every 6 hours', cron: '0 */6 * * *' },
  { label: 'Every 12 hours', cron: '0 */12 * * *' },
  { label: 'Daily at 09:00', cron: '0 9 * * *' },
  { label: 'Weekdays at 09:00', cron: '0 9 * * 1-5' },
  { label: 'Weekly (Mon 09:00)', cron: '0 9 * * 1' },
  { label: 'Monthly (1st 09:00)', cron: '0 9 1 * *' },
]

// ─── Right panel: Pipeline Settings ──────────────────────────────────────────

function PipelineSettingsPanel({
  sources,
  schedule,
  context,
  onSourcesChange,
  onScheduleChange,
  onSave,
  onContextSave,
}: {
  sources: PipelineSource[]
  schedule: string
  context: PipelineContext | undefined
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onSave: () => Promise<void>
  onContextSave: (ctx: PipelineContext) => Promise<void>
}) {
  const [showAddModal, setShowAddModal] = useState(false)
  const [saving, setSaving] = useState(false)
  const [sourcesOpen, setSourcesOpen] = useState(true)
  const [briefOpen, setBriefOpen] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try { await onSave() } finally { setSaving(false) }
  }

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold">Pipeline Settings</h2>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">

        {/* Sources & Schedule */}
        <section>
          <button
            onClick={() => setSourcesOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Sources &amp; Schedule
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${sourcesOpen ? 'rotate-180' : ''}`} />
          </button>

          {sourcesOpen && (
            <div className="space-y-4">
              {sources.length === 0 ? (
                <div className="py-4 text-center rounded-xl border border-dashed border-border">
                  <p className="text-xs text-muted-foreground mb-2">No sources configured.</p>
                  <button
                    onClick={() => setShowAddModal(true)}
                    className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                  >
                    <Plus className="h-3 w-3" /> Add source
                  </button>
                </div>
              ) : (
                <div className="rounded-xl border border-border overflow-hidden">
                  {sources.map((src, i) => (
                    <div key={src.id} className="flex items-center gap-2 px-3 py-2.5 border-b border-border last:border-b-0">
                      <SourceTypeBadge type={src.source_type} />
                      <span className="text-xs font-medium flex-1 truncate">{src.label}</span>
                      <button
                        onClick={() => onSourcesChange(sources.filter((_, j) => j !== i))}
                        className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              {sources.length > 0 && (
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  <Plus className="h-3 w-3" /> Add source
                </button>
              )}

              <div>
                <label className="block text-xs font-medium text-muted-foreground mb-1.5">
                  <Clock className="inline h-3 w-3 mr-1" />
                  Schedule
                </label>
                <select
                  value={SCHEDULE_PRESETS.some((p) => p.cron === schedule) ? schedule : '__custom__'}
                  onChange={(e) => onScheduleChange(e.target.value === '__custom__' ? '' : e.target.value)}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-xs outline-none focus:ring-1 focus:ring-ring cursor-pointer"
                >
                  <option value="" disabled>Select schedule…</option>
                  {SCHEDULE_PRESETS.map((p) => <option key={p.cron} value={p.cron}>{p.label}</option>)}
                  <option value="__custom__">Custom cron…</option>
                </select>
                <input
                  type="text"
                  value={schedule}
                  onChange={(e) => onScheduleChange(e.target.value)}
                  placeholder="0 */6 * * *"
                  readOnly={SCHEDULE_PRESETS.some((p) => p.cron === schedule)}
                  className={`w-full rounded-xl border border-input bg-background px-3 py-2 text-xs font-mono outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground mt-2 ${SCHEDULE_PRESETS.some((p) => p.cron === schedule) ? 'text-muted-foreground' : ''}`}
                />
              </div>

              <button
                onClick={handleSave}
                disabled={saving}
                className="w-full flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
              >
                {saving ? <><Loader2 className="h-3 w-3 animate-spin" />Saving…</> : 'Save'}
              </button>
            </div>
          )}
        </section>

        {/* Editorial Brief */}
        <section>
          <button
            onClick={() => setBriefOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Editorial Brief
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${briefOpen ? 'rotate-180' : ''}`} />
          </button>
          {briefOpen && (
            <EditorialBriefForm initialContext={context} onSave={onContextSave} />
          )}
        </section>
      </div>

      {showAddModal && (
        <AddSourceModal
          onAdd={(src) => onSourcesChange([...sources, src])}
          onClose={() => setShowAddModal(false)}
        />
      )}
    </div>
  )
}

// ─── Session table ────────────────────────────────────────────────────────────

type SessionFilter = 'all' | 'pending_review' | 'producing' | 'published' | 'rejected'

const STATUS_BAR_COLOR: Record<string, string> = {
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/50',
  rejected: 'bg-destructive/50',
  collecting: 'bg-primary',
}

function SessionTable({
  sessions,
  search,
  filter,
  approvingId,
  rejectingId,
  onRowClick,
  onApprove,
  onReject,
}: {
  sessions: ContentSession[]
  search: string
  filter: SessionFilter
  approvingId: string | null
  rejectingId: string | null
  onRowClick: (session: ContentSession) => void
  onApprove: (id: string) => void
  onReject: (id: string) => void
}) {
  const filtered = sessions
    .filter((s) => filter === 'all' || s.status === filter)
    .filter((s) => {
      if (!search) return true
      const q = search.toLowerCase()
      return (
        `session ${s.session_number}`.includes(q) ||
        s.analysis?.summary?.toLowerCase().includes(q) ||
        s.status.includes(q)
      )
    })

  if (filtered.length === 0) {
    return (
      <div className="text-center py-12 text-sm text-muted-foreground">
        {search ? `No sessions matching "${search}"` : 'No sessions with this status.'}
      </div>
    )
  }

  return (
    <div className="rounded-xl border border-border overflow-hidden">
      {/* Table header */}
      <div className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 px-4 py-2 bg-muted/30 border-b border-border text-xs font-medium text-muted-foreground">
        <span>Session</span>
        <span>Status</span>
        <span>Score</span>
        <span>Created</span>
        <span></span>
      </div>

      {filtered.map((session) => {
        const createdAt = new Date(session.created_at).toLocaleString('en-US', {
          month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
        })
        const isApproving = approvingId === session.id
        const isRejecting = rejectingId === session.id

        return (
          <div
            key={session.id}
            className={`relative grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 px-4 py-3.5 border-b border-border last:border-b-0 items-center
              hover:bg-muted/20 transition-colors cursor-pointer group pl-5
              ${session.status === 'rejected' ? 'opacity-60' : ''}`}
            onClick={() => onRowClick(session)}
          >
            {/* Left status bar */}
            <div className={`absolute left-0 top-0 bottom-0 w-1 ${STATUS_BAR_COLOR[session.status] ?? 'bg-muted'}`} />

            {/* Session name + summary */}
            <div className="min-w-0">
              <div className="text-sm font-semibold text-primary group-hover:underline">
                Session {session.session_number}
              </div>
              {session.analysis?.summary && (
                <div className="text-xs text-muted-foreground truncate mt-0.5 max-w-xs">
                  {session.analysis.summary}
                </div>
              )}
            </div>

            {/* Status */}
            <div onClick={(e) => e.stopPropagation()}>
              <StatusBadge status={session.status} />
            </div>

            {/* Score */}
            <div>
              {session.analysis
                ? <ScoreIndicator score={session.analysis.score} />
                : <span className="text-xs text-muted-foreground">—</span>}
            </div>

            {/* Created */}
            <div className="text-xs text-muted-foreground whitespace-nowrap">{createdAt}</div>

            {/* Actions */}
            <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
              {session.status === 'pending_review' ? (
                <>
                  <button
                    onClick={() => onApprove(session.id)}
                    disabled={isApproving || isRejecting}
                    title="Approve & Run All"
                    className="p-1.5 rounded-lg text-success hover:bg-success/15 transition-colors disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
                  >
                    {isApproving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
                  </button>
                  <button
                    onClick={() => onReject(session.id)}
                    disabled={isApproving || isRejecting}
                    title="Reject"
                    className="p-1.5 rounded-lg text-destructive hover:bg-destructive/15 transition-colors disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
                  >
                    {isRejecting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <X className="h-3.5 w-3.5" />}
                  </button>
                </>
              ) : session.status === 'producing' || session.status === 'published' ? (
                <div className="flex items-center gap-2">
                  {(session.workflow_results ?? []).map((wr) => (
                    <span key={wr.run_id} className="text-xs text-muted-foreground">
                      {wr.status === 'success'
                        ? <CheckCircle2 className="h-3.5 w-3.5 text-success inline" />
                        : wr.status === 'failed'
                        ? <XCircle className="h-3.5 w-3.5 text-destructive inline" />
                        : <Loader2 className="h-3.5 w-3.5 text-info animate-spin inline" />}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>
          </div>
        )
      })}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function PipelineDetailPage() {
  const { id, sessionId } = useParams<{ id: string; sessionId?: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: pipeline, isLoading } = useQuery({
    queryKey: ['pipeline', id],
    queryFn: () => fetchPipeline(id!),
    enabled: !!id,
  })

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')

  useEffect(() => {
    if (pipeline) {
      setLocalSources(pipeline.sources ?? [])
      setLocalSchedule(pipeline.schedule ?? '')
    }
  }, [pipeline?.id])

  const collectMutation = useMutation({
    mutationFn: () => collectPipeline(id!),
    onSuccess: ({ session_id }) => {
      navigate(`/pipelines/${id}/sessions/${session_id}`)
    },
  })

  const updateContextMutation = useMutation({
    mutationFn: (ctx: PipelineContext) => updatePipeline(id!, { ...pipeline!, context: ctx }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipeline', id] }),
  })

  const handleSaveSourcesAndSchedule = async () => {
    if (!pipeline) return
    await updatePipeline(id!, { ...pipeline, sources: localSources, schedule: localSchedule })
    queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
  }

  const [approvingId, setApprovingId] = useState<string | null>(null)
  const [rejectingId, setRejectingId] = useState<string | null>(null)
  const [activeFilter, setActiveFilter] = useState<SessionFilter>('all')
  const [search, setSearch] = useState('')

  const { data: sessions = [], isLoading: sessionsLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId: id }],
    queryFn: () => fetchContentSessions({ pipelineId: id }),
    enabled: !!id,
  })

  const handleApprove = async (sessionId: string) => {
    setApprovingId(sessionId)
    try {
      await useContentSessionStore.getState().approveSession(sessionId, [])
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    } finally {
      setApprovingId(null)
    }
  }

  const handleReject = async (sessionId: string) => {
    setRejectingId(sessionId)
    try {
      await useContentSessionStore.getState().rejectSession(sessionId)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    } finally {
      setRejectingId(null)
    }
  }

  const lastCollectedLabel = pipeline?.last_collected_at
    ? new Date(pipeline.last_collected_at).toLocaleString('en-US', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
      })
    : 'Never'

  const filterTabs: { value: SessionFilter; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'pending_review', label: 'Pending' },
    { value: 'producing', label: 'Producing' },
    { value: 'published', label: 'Published' },
    { value: 'rejected', label: 'Rejected' },
  ]

  const filterCounts: Record<SessionFilter, number> = {
    all: sessions.length,
    pending_review: sessions.filter(s => s.status === 'pending_review').length,
    producing: sessions.filter(s => s.status === 'producing').length,
    published: sessions.filter(s => s.status === 'published').length,
    rejected: sessions.filter(s => s.status === 'rejected').length,
  }

  if (isLoading || !pipeline) {
    return (
      <MainLayout headerContent={<span className="font-semibold text-muted-foreground">Loading Pipeline...</span>}>
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
            className="p-1.5 rounded-md hover:bg-muted transition-colors cursor-pointer shrink-0"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <span className="text-xs text-muted-foreground shrink-0">Pipelines /</span>
          <span className="font-semibold truncate">{pipeline.name}</span>
        </div>
      }
      rightPanel={
        <PipelineSettingsPanel
          sources={localSources}
          schedule={localSchedule}
          context={pipeline.context}
          onSourcesChange={setLocalSources}
          onScheduleChange={setLocalSchedule}
          onSave={handleSaveSourcesAndSchedule}
          onContextSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
        />
      }
    >
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-6 space-y-4">

          {/* Pipeline meta */}
          <div>
            {pipeline.description && (
              <p className="text-sm text-muted-foreground mb-2">{pipeline.description}</p>
            )}
            <div className="flex items-center gap-2 flex-wrap">
              <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-muted/40 border border-border/50 text-xs text-muted-foreground">
                <Database className="h-3 w-3" />
                {(pipeline.sources ?? []).length} sources
              </span>
              {pipeline.schedule && (
                <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-muted/40 border border-border/50 text-xs text-muted-foreground font-mono">
                  <Clock className="h-3 w-3 shrink-0 font-sans" />
                  {pipeline.schedule}
                </span>
              )}
              <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-muted/40 border border-border/50 text-xs text-muted-foreground">
                <RefreshCw className="h-3 w-3" />
                {lastCollectedLabel}
              </span>
            </div>
          </div>

          {/* Sessions section header */}
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <h2 className="text-sm font-semibold shrink-0">
                Sessions {sessions.length > 0 && `(${sessions.length})`}
              </h2>
              {/* Search */}
              <div className="relative flex-1 max-w-xs">
                <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
                <input
                  type="search"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search sessions..."
                  className="w-full pl-8 pr-3 py-1.5 rounded-lg border border-input bg-background text-xs outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
                />
              </div>
            </div>
            <button
              onClick={() => collectMutation.mutate()}
              disabled={collectMutation.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-primary text-primary-foreground hover:opacity-90 transition-opacity disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed shrink-0"
            >
              {collectMutation.isPending
                ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Collecting…</>
                : <><Play className="h-3.5 w-3.5" />Collect Now</>
              }
            </button>
          </div>

          {/* Filter tabs */}
          <div className="flex items-center gap-1 flex-wrap">
            {filterTabs.map((tab) => {
              const count = filterCounts[tab.value]
              const isActive = activeFilter === tab.value
              const isPending = tab.value === 'pending_review'
              return (
                <button
                  key={tab.value}
                  onClick={() => setActiveFilter(tab.value)}
                  className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer
                    ${isActive ? 'bg-foreground text-background' : 'text-muted-foreground hover:text-foreground hover:bg-muted'}`}
                >
                  {tab.label}
                  {count > 0 && (
                    <span className={`text-[10px] font-semibold rounded-full min-w-[16px] px-1 py-0 text-center
                      ${isActive ? 'bg-background/20 text-background' : isPending ? 'bg-warning text-primary-foreground' : 'bg-muted-foreground/20 text-muted-foreground'}`}>
                      {count}
                    </span>
                  )}
                </button>
              )
            })}
          </div>

          {/* Table */}
          {sessionsLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : sessions.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-sm text-muted-foreground mb-4">No sessions yet.</p>
              <button
                onClick={() => collectMutation.mutate()}
                disabled={collectMutation.isPending}
                className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:opacity-90 transition-opacity disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
              >
                {collectMutation.isPending
                  ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Collecting…</>
                  : <><Play className="h-3.5 w-3.5" />Collect Now</>}
              </button>
            </div>
          ) : (
            <SessionTable
              sessions={sessions}
              search={search}
              filter={activeFilter}
              approvingId={approvingId}
              rejectingId={rejectingId}
              onRowClick={(s) => navigate(`/pipelines/${id}/sessions/${s.id}`)}
              onApprove={handleApprove}
              onReject={handleReject}
            />
          )}

        </div>
      </div>

      {/* Session Detail Modal */}
      {sessionId && (
        <SessionDetailModal
          sessionId={sessionId}
          pipelineId={id!}
          onClose={() => navigate(`/pipelines/${id}`)}
        />
      )}
    </MainLayout>
  )
}
```

**Step 2: Check compilation**

```bash
cd web && npx tsc -b 2>&1 | head -40
```

Expected: No new errors from `PipelineDetail.tsx`

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline): redesign pipeline detail with session table and settings right panel"
```

---

### Task 4: Update collectPipeline navigation in PipelinesPage

**Files:**
- Modify: `web/src/pages/Pipelines.tsx`

The `collectMutation` currently navigates to `/inbox/${session_id}`. Update it to use the pipeline-relative URL. The pipeline ID is available via `collectMutation.variables` (the ID passed to `collectPipeline(p.id)`).

**Step 1: Update onSuccess handler**

Find in `web/src/pages/Pipelines.tsx`:
```tsx
const collectMutation = useMutation({
  mutationFn: (id: string) => collectPipeline(id),
  onSuccess: ({ session_id }) => navigate(`/inbox/${session_id}`),
})
```

Replace with:
```tsx
const collectMutation = useMutation({
  mutationFn: (id: string) => collectPipeline(id),
  onSuccess: ({ session_id }, pipelineId) => navigate(`/pipelines/${pipelineId}/sessions/${session_id}`),
})
```

Note: `useMutation`'s `onSuccess` receives `(data, variables)` — `variables` is the `id` string passed to `mutationFn`.

**Step 2: Verify compilation**

```bash
cd web && npx tsc -b 2>&1 | head -20
```

Expected: No errors.

**Step 3: Commit**

```bash
git add web/src/pages/Pipelines.tsx
git commit -m "fix(pipelines): update collect navigation to use pipeline-scoped session URL"
```

---

### Task 5: Delete `/inbox` pages and clean up router imports

**Files:**
- Delete: `web/src/pages/inbox/index.tsx`
- Delete: `web/src/pages/inbox/SessionDetail.tsx` (replaced by SessionDetailModal)
- Modify: `web/src/app/router.tsx` (remove stale imports)

**Step 1: Remove files**

```bash
rm web/src/pages/inbox/index.tsx
rm web/src/pages/inbox/SessionDetail.tsx
```

**Step 2: Clean router.tsx imports**

The router.tsx already updated in Task 1 doesn't import these files. Verify `router.tsx` has no references to `InboxPage` or `SessionDetailPage`:

```bash
cd web && grep -n "Inbox\|SessionDetail" src/app/router.tsx
```

Expected: No output (those imports were already removed in Task 1).

**Step 3: Check for any remaining references to /inbox**

```bash
cd web && grep -rn '"/inbox' src/ --include="*.tsx" --include="*.ts"
```

Review any remaining references and update them to the new `/pipelines/:id/sessions/:sessionId` pattern. Most common places:
- Surge `createSessionFromSurge` success handler in the now-deleted `inbox/index.tsx` — already gone
- Any direct link to `/inbox` — update to `/pipelines`

**Step 4: Full type check**

```bash
cd web && npx tsc -b 2>&1
```

Expected: 0 errors

**Step 5: Commit**

```bash
git add -A web/src/pages/inbox/ web/src/app/router.tsx
git commit -m "feat(ux): remove /inbox routes, sessions now accessed via pipeline detail"
```

---

### Task 6: Final verification

**Step 1: Lint**

```bash
cd web && npm run lint
```

**Step 2: Build**

```bash
cd web && npm run build
```

Expected: Clean build, no errors.

**Step 3: Manual smoke test checklist**

- [ ] `/pipelines` — pipeline list loads
- [ ] Click pipeline → `/pipelines/:id` — session table shows, right panel shows pipeline settings
- [ ] Search in session search box — filters rows live
- [ ] Click filter tab "Pending" — shows only pending sessions
- [ ] Click session row → URL changes to `/pipelines/:id/sessions/:sessionId`, modal opens
- [ ] Press Escape → modal closes, URL returns to `/pipelines/:id`
- [ ] Click backdrop → modal closes
- [ ] Approve from modal → modal closes, table row status updates
- [ ] Approve/Reject inline buttons on pending rows in table work
- [ ] "Collect Now" button → navigates to new session modal
- [ ] Right panel: sources save, schedule save, editorial brief save
- [ ] `/inbox` URL → 404 or redirect (expected: route not found, falls through)
- [ ] Left nav: "Inbox" item is gone

**Step 4: Commit if any minor fixes applied**

```bash
git add -A web/
git commit -m "fix(pipeline-ux): post-integration cleanup"
```
