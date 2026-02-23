import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft, Play, Loader2, Trash2, Plus, ExternalLink,
  Clock, Database, RefreshCw, Check, X, CheckCircle2, XCircle, ChevronDown,
} from 'lucide-react'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { MainLayout } from '@/app/layout'
import {
  fetchPipeline, updatePipeline, collectPipeline,
} from '@/entities/pipeline'
import { fetchContentSessions } from '@/entities/content-session/api'
import type { PipelineSource, PipelineContext } from '@/shared/types'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { ContentSession } from '@/entities/content-session/types'
import type { WorkflowResult } from '@/entities/content-session/types'

// ─── Sub-panels ──────────────────────────────────────────────────────────────

const STATUS_BAR_COLOR: Record<string, string> = {
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/50',
  rejected: 'bg-destructive/50',
  collecting: 'bg-primary',
}

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

  const createdAt = new Date(session.created_at).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  })

  const sourceCount = session.sources?.reduce((sum, s) => sum + s.count, 0) ?? 0

  return (
    <div className={`relative rounded-xl border border-border bg-card pl-5 pr-4 py-4 overflow-hidden
      ${session.status === 'rejected' ? 'opacity-60' : ''}`}
    >
      {/* Left status bar */}
      <div className={`absolute left-0 top-0 bottom-0 w-1 rounded-l-xl ${STATUS_BAR_COLOR[session.status] ?? 'bg-muted'}`} />

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
                bg-success text-primary-foreground hover:opacity-90 disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
            >
              {isApproving
                ? <Loader2 className="h-3 w-3 animate-spin" />
                : <Check className="h-3 w-3" />}
              Approve &amp; Run All
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
                  ? 'bg-warning text-primary-foreground'
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

function SourceConfigTab({
  sources,
  schedule,
  onSourcesChange,
  onScheduleChange,
  onSave,
}: {
  sources: PipelineSource[]
  schedule: string
  onSourcesChange: (sources: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onSave: () => Promise<void>
}) {
  const [showAddModal, setShowAddModal] = useState(false)
  const [saving, setSaving] = useState(false)

  const handleSave = async () => {
    setSaving(true)
    try { await onSave() } finally { setSaving(false) }
  }

  return (
    <div className="space-y-5">
      {/* Source list */}
      <div className="flex items-center justify-end">
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-1 text-xs text-muted-foreground
            hover:text-foreground transition-colors cursor-pointer"
        >
          <Plus className="h-3.5 w-3.5" />
          Add source
        </button>
      </div>

      {sources.length === 0 ? (
        <div className="py-6 text-center rounded-xl border border-dashed border-border">
          <p className="text-sm text-muted-foreground mb-3">No sources configured.</p>
          <button
            onClick={() => setShowAddModal(true)}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
              bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
          >
            <Plus className="h-3.5 w-3.5" />
            Add source
          </button>
        </div>
      ) : (
        <div className="rounded-xl border border-border overflow-hidden">
          {sources.map((src, i) => (
            <div
              key={src.id}
              className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0"
            >
              <SourceTypeBadge type={src.source_type} />
              <span className="text-sm font-medium flex-1">{src.label}</span>
              <span className="text-xs text-muted-foreground truncate max-w-[180px]">
                {src.url ?? src.subreddit ?? (src.keywords?.join(', ') ?? '')}
              </span>
              <button
                onClick={() => onSourcesChange(sources.filter((_, j) => j !== i))}
                className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Schedule */}
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          <Clock className="inline h-3.5 w-3.5 mr-1" />
          Collection schedule (cron)
        </label>
        <input
          type="text"
          value={schedule}
          onChange={(e) => onScheduleChange(e.target.value)}
          placeholder="0 */6 * * *  (every 6 hours)"
          className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm
            font-mono outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground placeholder:font-sans"
        />
        <p className="text-xs text-muted-foreground mt-1">
          Cron expression. e.g. <code className="font-mono">0 */6 * * *</code> = every 6 hours
        </p>
      </div>

      <button
        onClick={handleSave}
        disabled={saving}
        className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
          bg-foreground text-background hover:opacity-90 transition-opacity
          disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
      >
        {saving ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Saving…</> : 'Save'}
      </button>

      {showAddModal && (
        <AddSourceModal
          onAdd={(src) => onSourcesChange([...sources, src])}
          onClose={() => setShowAddModal(false)}
        />
      )}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function PipelineDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: pipeline, isLoading } = useQuery({
    queryKey: ['pipeline', id],
    queryFn: () => fetchPipeline(id!),
    enabled: !!id,
  })

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')

  // Sync local state when a different pipeline loads
  useEffect(() => {
    if (pipeline) {
      setLocalSources(pipeline.sources ?? [])
      setLocalSchedule(pipeline.schedule ?? '')
    }
  }, [pipeline?.id])

  const collectMutation = useMutation({
    mutationFn: () => collectPipeline(id!),
    onSuccess: ({ session_id }) => {
      navigate(`/inbox/${session_id}`)
    },
  })

  const updateContextMutation = useMutation({
    mutationFn: (ctx: PipelineContext) => updatePipeline(id!, { ...pipeline!, context: ctx }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
    },
  })

  const handleSaveSourcesAndSchedule = async () => {
    if (!pipeline) return
    await updatePipeline(id!, { ...pipeline, sources: localSources, schedule: localSchedule })
    queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
  }

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

  if (isLoading || !pipeline) {
    return (
      <MainLayout headerContent={<span className="font-semibold text-muted-foreground">Loading Pipeline...</span>}>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        </div>
      </MainLayout>
    )
  }

  const lastCollectedLabel = pipeline.last_collected_at
    ? new Date(pipeline.last_collected_at).toLocaleString('en-US', {
      month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
    })
    : 'Never'

  return (
    <MainLayout
      headerContent={
        <div className="flex items-center gap-3 w-full justify-between">
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
          <div className="flex items-center gap-2 shrink-0">
            <button
              onClick={() => collectMutation.mutate()}
              disabled={collectMutation.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium
                bg-primary text-primary-foreground hover:opacity-90 transition-opacity
                disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
            >
              {collectMutation.isPending
                ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Collecting…</>
                : <><Play className="h-3.5 w-3.5" />Collect Now</>
              }
            </button>
          </div>
        </div>
      }
    >
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
                Data Sources &amp; Schedule
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
                Editorial Brief &amp; Context
              </h3>
              <EditorialBriefForm
                initialContext={pipeline.context}
                onSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
              />
            </div>
          </PipelineSettingsAccordion>

        </div>
      </div>
    </MainLayout>
  )
}
