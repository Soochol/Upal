# Pipeline Detail Sessions-First Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure `PipelineDetail` so sessions are the primary full-width content with inline approve/reject and workflow status, with pipeline settings demoted to a collapsible accordion.

**Architecture:** Single file change (`PipelineDetail.tsx`) — add 4 new sub-components (`SessionCard`, `WorkflowResultBadge`, `SessionFilterTabs`, `PipelineSettingsAccordion`), replace `SessionHistoryTab` with the new card list, and remove `WorkflowTemplatesTab` (workflow info moves into `SessionCard`).

**Tech Stack:** React 19, TypeScript, TanStack Query, Zustand, Tailwind v4, Lucide icons

---

## Context

All changes are in one file: `web/src/pages/pipelines/PipelineDetail.tsx`

**Key types (from `@/entities/content-session/types`):**
```typescript
type ContentSessionStatus = 'collecting' | 'pending_review' | 'approved' | 'rejected' | 'producing' | 'published'
type WorkflowResult = { workflow_name: string; run_id: string; status: 'pending'|'running'|'success'|'failed'; output_url?: string; completed_at?: string }
type ContentSession = { id: string; session_number?: number; trigger_type: 'schedule'|'manual'|'surge'; status: ContentSessionStatus; sources?: SourceFetch[]; analysis?: LLMAnalysis; workflow_results?: WorkflowResult[]; created_at: string; ... }
```

**Approve/Reject — use Zustand store (same pattern as `/inbox`):**
```typescript
await useContentSessionStore.getState().approveSession(id, [])
await useContentSessionStore.getState().rejectSession(id)
```

**Existing components to keep:** `SourceConfigTab`, `EditorialBriefForm`, `AddSourceModal` — unchanged.
**Remove:** `SessionHistoryTab` (line 22–113), `WorkflowTemplatesTab` (line 403–442).

---

## Task 1: Add `WorkflowResultBadge` and `SessionCard` components

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

**Step 1: Add imports**

Add to the existing import block at the top of the file:

```typescript
import { Check, X, CheckCircle2, XCircle, ChevronDown } from 'lucide-react'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { ContentSession } from '@/entities/content-session/types'
import type { WorkflowResult } from '@/entities/content-session/types'
```

**Step 2: Add `WorkflowResultBadge` after the existing imports (before `SessionHistoryTab`)**

```typescript
function WorkflowResultBadge({ result }: { result: WorkflowResult }) {
  const icon = result.status === 'running'
    ? <Loader2 className="h-3 w-3 animate-spin text-info" />
    : result.status === 'success'
    ? <CheckCircle2 className="h-3 w-3 text-success" />
    : result.status === 'failed'
    ? <XCircle className="h-3 w-3 text-destructive" />
    : <Clock className="h-3 w-3 text-muted-foreground" />

  return (
    <span className="flex items-center gap-1 text-xs text-muted-foreground">
      {icon}
      <span>{result.workflow_name}</span>
    </span>
  )
}
```

**Step 3: Add `SessionCard` after `WorkflowResultBadge`**

```typescript
function SessionCard({
  session,
  onView,
  approvingId,
  rejectingId,
  onApprove,
  onReject,
}: {
  session: ContentSession
  onView: (id: string) => void
  approvingId: string | null
  rejectingId: string | null
  onApprove: (id: string) => void
  onReject: (id: string) => void
}) {
  const isApproving = approvingId === session.id
  const isRejecting = rejectingId === session.id

  const statusBarColor: Record<string, string> = {
    pending_review: 'bg-warning',
    approved: 'bg-success',
    producing: 'bg-info',
    published: 'bg-success/50',
    rejected: 'bg-destructive/50',
    collecting: 'bg-primary',
  }

  const createdAt = new Date(session.created_at).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  })

  const sourceCount = session.sources?.reduce((sum, s) => sum + s.count, 0) ?? 0

  return (
    <div className={`relative rounded-xl border border-border bg-card pl-5 pr-4 py-4 overflow-hidden
      ${session.status === 'rejected' ? 'opacity-60' : ''}`}
    >
      {/* Left status bar */}
      <div className={`absolute left-0 top-0 bottom-0 w-1 rounded-l-xl ${statusBarColor[session.status] ?? 'bg-muted'}`} />

      {/* Header row */}
      <div className="flex items-center justify-between gap-3 mb-2">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-semibold">Session {session.session_number}</span>
          <StatusBadge status={session.status} />
          {session.analysis && <ScoreIndicator score={session.analysis.score} />}
        </div>
        <button
          onClick={() => onView(session.id)}
          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors shrink-0 cursor-pointer"
        >
          <ExternalLink className="h-3.5 w-3.5" />
          View
        </button>
      </div>

      {/* Meta row */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-3 flex-wrap">
        <span>{createdAt}</span>
        {sourceCount > 0 && <><span>·</span><span>{sourceCount} articles</span></>}
        <span>·</span>
        <span>{session.trigger_type}</span>
      </div>

      {/* AI summary */}
      {session.analysis?.summary && (
        <>
          <p className="text-sm text-muted-foreground line-clamp-2 italic mb-3">
            "{session.analysis.summary}"
          </p>
          <div className="border-t border-border mb-3" />
        </>
      )}

      {/* collecting state */}
      {session.status === 'collecting' && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Collecting from sources…
        </div>
      )}

      {/* pending_review: workflow chips + approve/reject */}
      {session.status === 'pending_review' && (
        <div className="space-y-3">
          {(session.analysis?.angles ?? []).length > 0 && (
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-xs text-muted-foreground">Workflows:</span>
              {session.analysis!.angles.map((angle) => (
                <span
                  key={angle.id}
                  className="px-2 py-0.5 rounded-md text-xs bg-muted border border-border text-foreground"
                >
                  {angle.format}
                </span>
              ))}
            </div>
          )}
          <div className="flex items-center gap-2">
            <button
              onClick={() => onApprove(session.id)}
              disabled={isApproving || isRejecting}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium
                bg-success text-white hover:opacity-90 disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
            >
              {isApproving
                ? <Loader2 className="h-3 w-3 animate-spin" />
                : <Check className="h-3 w-3" />}
              Approve & Run All
            </button>
            <button
              onClick={() => onReject(session.id)}
              disabled={isApproving || isRejecting}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium
                text-destructive border border-destructive/30 hover:bg-destructive/10
                disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
            >
              {isRejecting
                ? <Loader2 className="h-3 w-3 animate-spin" />
                : <X className="h-3 w-3" />}
              Reject
            </button>
          </div>
        </div>
      )}

      {/* producing / published: workflow results */}
      {(session.status === 'producing' || session.status === 'published') &&
        (session.workflow_results ?? []).length > 0 && (
          <div className="flex items-center gap-4 flex-wrap">
            {session.workflow_results!.map((wr) => (
              <WorkflowResultBadge key={wr.run_id} result={wr} />
            ))}
          </div>
        )}
    </div>
  )
}
```

**Step 4: Run type-check**

```bash
cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | head -40
```

Expected: no errors for the new components (may have errors elsewhere from other modified files — that's OK for now).

**Step 5: Commit**

```bash
cd /home/dev/code/Upal && git add web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline-detail): add SessionCard and WorkflowResultBadge components"
```

---

## Task 2: Add `SessionFilterTabs` component

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

**Step 1: Add `SessionFilterTabs` after `SessionCard`**

```typescript
type SessionFilter = 'all' | 'pending_review' | 'producing' | 'published' | 'rejected'

function SessionFilterTabs({
  sessions,
  activeFilter,
  onFilterChange,
}: {
  sessions: ContentSession[]
  activeFilter: SessionFilter
  onFilterChange: (f: SessionFilter) => void
}) {
  const counts: Record<SessionFilter, number> = {
    all: sessions.length,
    pending_review: sessions.filter((s) => s.status === 'pending_review').length,
    producing: sessions.filter((s) => s.status === 'producing').length,
    published: sessions.filter((s) => s.status === 'published').length,
    rejected: sessions.filter((s) => s.status === 'rejected').length,
  }

  const tabs: { value: SessionFilter; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'pending_review', label: 'Pending' },
    { value: 'producing', label: 'Producing' },
    { value: 'published', label: 'Published' },
    { value: 'rejected', label: 'Rejected' },
  ]

  return (
    <div className="flex items-center gap-1 flex-wrap">
      {tabs.map((tab) => {
        const count = counts[tab.value]
        const isActive = activeFilter === tab.value
        const showBadge = count > 0
        const isPending = tab.value === 'pending_review'

        return (
          <button
            key={tab.value}
            onClick={() => onFilterChange(tab.value)}
            className={`flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer
              ${isActive
                ? 'bg-foreground text-background'
                : 'text-muted-foreground hover:text-foreground hover:bg-muted'
              }`}
          >
            {tab.label}
            {showBadge && (
              <span className={`text-[10px] font-semibold rounded-full min-w-[16px] px-1 py-0 text-center
                ${isActive
                  ? 'bg-background/20 text-background'
                  : isPending
                  ? 'bg-warning text-white'
                  : 'bg-muted-foreground/20 text-muted-foreground'
                }`}>
                {count}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
```

**Step 2: Run type-check**

```bash
cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | head -40
```

Expected: no new errors.

**Step 3: Commit**

```bash
cd /home/dev/code/Upal && git add web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline-detail): add SessionFilterTabs component"
```

---

## Task 3: Add `PipelineSettingsAccordion` component

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

**Step 1: Add `PipelineSettingsAccordion` after `SessionFilterTabs`**

```typescript
function PipelineSettingsAccordion({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false)

  return (
    <section className="glass-panel border border-white/5 rounded-2xl overflow-hidden shadow-sm">
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between px-5 py-4 text-sm font-semibold
          hover:bg-muted/30 transition-colors cursor-pointer"
      >
        <span>Pipeline Settings</span>
        <ChevronDown className={`h-4 w-4 text-muted-foreground transition-transform duration-200
          ${open ? 'rotate-180' : ''}`}
        />
      </button>
      {open && (
        <div className="px-5 pb-5 space-y-8 border-t border-border pt-5">
          {children}
        </div>
      )}
    </section>
  )
}
```

**Step 2: Run type-check**

```bash
cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | head -40
```

Expected: no new errors.

**Step 3: Commit**

```bash
cd /home/dev/code/Upal && git add web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline-detail): add PipelineSettingsAccordion component"
```

---

## Task 4: Restructure `PipelineDetailPage` — wire everything together

**Files:**
- Modify: `web/src/pages/pipelines/PipelineDetail.tsx`

**Step 1: Add `approvingId`, `rejectingId`, `activeFilter` state + handlers in `PipelineDetailPage`**

Inside `PipelineDetailPage` function body, after the existing `collectMutation`:

```typescript
const [approvingId, setApprovingId] = useState<string | null>(null)
const [rejectingId, setRejectingId] = useState<string | null>(null)
const [activeFilter, setActiveFilter] = useState<SessionFilter>('all')

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

const filteredSessions = activeFilter === 'all'
  ? sessions
  : sessions.filter((s) => s.status === activeFilter)
```

**Step 2: Replace the main layout JSX**

Replace everything inside `<MainLayout>` from `<div className="flex-1 overflow-y-auto">` to the closing `</div>` (lines ~315–396) with:

```tsx
<div className="flex-1 overflow-y-auto">
  <div className="max-w-3xl mx-auto px-4 sm:px-6 py-8 space-y-6">

    {/* Meta info */}
    <div>
      {pipeline.description && (
        <p className="text-sm text-muted-foreground mb-3 ml-1">{pipeline.description}</p>
      )}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg
          bg-muted/40 border border-border/50 text-xs text-muted-foreground">
          <Database className="h-3 w-3" />
          {(pipeline.sources ?? []).length} sources
        </span>
        {pipeline.schedule && (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg
            bg-muted/40 border border-border/50 text-xs text-muted-foreground font-mono">
            <Clock className="h-3 w-3 shrink-0 font-sans" />
            {pipeline.schedule}
          </span>
        )}
        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg
          bg-muted/40 border border-border/50 text-xs text-muted-foreground">
          <RefreshCw className="h-3 w-3" />
          {lastCollectedLabel}
        </span>
      </div>
    </div>

    {/* Sessions — primary content */}
    <section className="glass-panel border border-white/5 rounded-2xl p-5 shadow-sm">
      <div className="flex items-center justify-between gap-3 mb-4 flex-wrap">
        <h2 className="text-sm font-semibold text-foreground/90">
          Sessions {sessions.length > 0 && `(${sessions.length})`}
        </h2>
        <SessionFilterTabs
          sessions={sessions}
          activeFilter={activeFilter}
          onFilterChange={setActiveFilter}
        />
      </div>

      {sessionsLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      ) : sessions.length === 0 ? (
        <div className="text-center py-10">
          <p className="text-sm text-muted-foreground mb-4">No sessions yet.</p>
          <button
            onClick={() => collectMutation.mutate()}
            disabled={collectMutation.isPending}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
              bg-primary text-primary-foreground hover:opacity-90 transition-opacity
              disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
          >
            {collectMutation.isPending
              ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Collecting…</>
              : <><Play className="h-3.5 w-3.5" />Collect Now</>}
          </button>
        </div>
      ) : filteredSessions.length === 0 ? (
        <p className="text-sm text-muted-foreground text-center py-6">
          No sessions with this status.
        </p>
      ) : (
        <div className="space-y-3">
          {filteredSessions.map((session) => (
            <SessionCard
              key={session.id}
              session={session}
              onView={(sid) => navigate(`/inbox/${sid}`)}
              approvingId={approvingId}
              rejectingId={rejectingId}
              onApprove={handleApprove}
              onReject={handleReject}
            />
          ))}
        </div>
      )}
    </section>

    {/* Pipeline Settings — accordion */}
    <PipelineSettingsAccordion>
      <div>
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-3">
          Data Sources & Schedule
        </h3>
        <SourceConfigTab
          sources={localSources}
          schedule={localSchedule}
          onSourcesChange={setLocalSources}
          onScheduleChange={setLocalSchedule}
          onSave={handleSaveSourcesAndSchedule}
        />
      </div>
      <div>
        <h3 className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-3">
          Editorial Brief & Context
        </h3>
        <EditorialBriefForm
          initialContext={pipeline.context}
          onSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
        />
      </div>
    </PipelineSettingsAccordion>

  </div>
</div>
```

**Step 3: Delete `SessionHistoryTab` (lines 22–113) and `WorkflowTemplatesTab` (lines 403–442)**

These components are no longer used. Remove them entirely.

**Step 4: Run type-check**

```bash
cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | head -60
```

Expected: 0 errors. Fix any type errors before proceeding.

**Step 5: Visually verify in browser**

- Navigate to `http://localhost:5173/pipelines/pipe-202810b1fab57f5e`
- Sessions appear as full-width cards with status-appropriate content
- Filter tabs update the visible list
- Pending sessions show Approve & Run All / Reject buttons
- Pipeline Settings accordion expands/collapses correctly
- "Collect Now" button in header still works

**Step 6: Commit**

```bash
cd /home/dev/code/Upal && git add web/src/pages/pipelines/PipelineDetail.tsx
git commit -m "feat(pipeline-detail): sessions-first layout with inline approve/reject

- Sessions are now full-width primary content with rich state-aware cards
- pending_review cards show workflow chips + approve/reject inline
- producing/published cards show per-workflow execution status
- Filter tabs for All/Pending/Producing/Published/Rejected
- Pipeline settings (Sources, Schedule, Brief) moved to collapsible accordion
- WorkflowTemplatesTab removed (workflow info per session card)

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Verification Checklist

- [ ] `npx tsc -b --noEmit` passes with 0 errors
- [ ] Sessions load and display as cards with left status bar
- [ ] `pending_review` card: shows AI summary, workflow format chips, Approve/Reject buttons
- [ ] `collecting` card: shows spinner + "Collecting…"
- [ ] `producing` card: shows workflow names with spinning icons
- [ ] `published` card: shows workflow names with check/x icons
- [ ] `rejected` card: muted opacity
- [ ] Filter tabs: clicking "Pending" shows only pending sessions, badge count visible
- [ ] Approve button: calls store, card status updates (via query invalidation)
- [ ] Reject button: calls store, card moves to rejected
- [ ] Pipeline Settings accordion: collapsed by default, expands on click
- [ ] Sources + Schedule still editable inside accordion
- [ ] Editorial Brief still saves inside accordion
- [ ] "Collect Now" header button creates session + navigates to /inbox/{id}
- [ ] No regressions on other pages
