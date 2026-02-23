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
      <div className="grid grid-cols-[1fr_auto_auto_auto_auto] gap-4 px-4 py-2 bg-muted/30 border-b border-border text-xs font-medium text-muted-foreground pl-5">
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
              ) : (session.status === 'producing' || session.status === 'published') ? (
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
