import { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Loader2, Trash2, Plus, Clock,
  Check, X, Search, CheckCircle2, XCircle, ChevronDown,
  MoreHorizontal, CloudUpload,
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
import type { PipelineSource, PipelineContext } from '@/shared/types'
import type { ContentSession } from '@/entities/content-session/types'

// ─── Schedule presets ──────────────────────────────────────────────────────────

const SCHEDULE_PRESETS: { label: string; cron: string }[] = [
  { label: 'Every hour', cron: '0 * * * *' },
  { label: 'Every 6 hours', cron: '0 */6 * * *' },
  { label: 'Every 12 hours', cron: '0 */12 * * *' },
  { label: 'Daily at 09:00', cron: '0 9 * * *' },
  { label: 'Weekdays at 09:00', cron: '0 9 * * 1-5' },
  { label: 'Weekly (Mon 09:00)', cron: '0 9 * * 1' },
  { label: 'Monthly (1st 09:00)', cron: '0 9 1 * *' },
]

// ─── Right panel: Pipeline Settings ───────────────────────────────────────────

function PipelineSettingsPanel({
  sources, schedule, context,
  onSourcesChange, onScheduleChange, onContextSave, autoSaveStatus,
}: {
  sources: PipelineSource[]
  schedule: string
  context: PipelineContext | undefined
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onContextSave: (ctx: PipelineContext) => Promise<void>
  autoSaveStatus: 'idle' | 'saving' | 'saved'
}) {
  const [showAddModal, setShowAddModal] = useState(false)
  const [sourcesOpen, setSourcesOpen] = useState(true)
  const [briefOpen, setBriefOpen] = useState(false)

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="px-4 py-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold">Settings</h2>
        <div className="h-4">
          {autoSaveStatus === 'saving' && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />Saving…
            </span>
          )}
          {autoSaveStatus === 'saved' && (
            <span className="flex items-center gap-1 text-xs text-success">
              <CloudUpload className="h-3 w-3" />Saved
            </span>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-4">
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
                  <Clock className="inline h-3 w-3 mr-1" />Schedule
                </label>
                <select
                  value={SCHEDULE_PRESETS.some(p => p.cron === schedule) ? schedule : '__custom__'}
                  onChange={(e) => onScheduleChange(e.target.value === '__custom__' ? '' : e.target.value)}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-xs outline-none focus:ring-1 focus:ring-ring cursor-pointer"
                >
                  <option value="" disabled>Select schedule…</option>
                  {SCHEDULE_PRESETS.map(p => <option key={p.cron} value={p.cron}>{p.label}</option>)}
                  <option value="__custom__">Custom cron…</option>
                </select>
                <input
                  type="text"
                  value={schedule}
                  onChange={(e) => onScheduleChange(e.target.value)}
                  placeholder="0 */6 * * *"
                  readOnly={SCHEDULE_PRESETS.some(p => p.cron === schedule)}
                  className={`w-full rounded-xl border border-input bg-background px-3 py-2 text-xs font-mono outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground mt-2 ${SCHEDULE_PRESETS.some(p => p.cron === schedule) ? 'text-muted-foreground' : ''}`}
                />
              </div>

            </div>
          )}
        </section>

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
            <EditorialBriefForm initialContext={context} onSave={onContextSave} autoSave />
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

// ─── New Session modal ───────────────────────────────────────────────────────

function NewSessionModal({
  isPending,
  onConfirm,
  onClose,
}: {
  isPending: boolean
  onConfirm: () => void
  onClose: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-sm mx-4 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">New Session</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="p-5">
          <p className="text-sm text-muted-foreground">
            Create a new session for this pipeline. Sources will be collected and analyzed automatically.
          </p>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-border">
          <button
            onClick={onClose}
            className="px-3.5 py-2 rounded-xl text-sm font-medium text-muted-foreground
              hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isPending}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium
              bg-foreground text-background hover:opacity-90 transition-opacity
              disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed"
          >
            {isPending
              ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Creating…</>
              : 'Create Session'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Status dot ───────────────────────────────────────────────────────────────

const STATUS_DOT: Record<string, string> = {
  pending_review: 'bg-warning',
  approved: 'bg-success',
  producing: 'bg-info',
  published: 'bg-success/70',
  rejected: 'bg-muted-foreground/40',
  collecting: 'bg-primary',
}

// ─── Session row ──────────────────────────────────────────────────────────────

function SessionRow({
  session,
  approvingId,
  rejectingId,
  onClick,
  onApprove,
  onReject,
}: {
  session: ContentSession
  approvingId: string | null
  rejectingId: string | null
  onClick: () => void
  onApprove: (id: string) => void
  onReject: (id: string) => void
}) {
  const isApproving = approvingId === session.id
  const isRejecting = rejectingId === session.id
  const createdAt = new Date(session.created_at).toLocaleString('en-US', {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
  })

  return (
    <tr
      className={`group border-b border-border/50 last:border-b-0 hover:bg-muted/20 transition-colors cursor-pointer
        ${session.status === 'rejected' ? 'opacity-50' : ''}`}
      onClick={onClick}
    >
      {/* Session name */}
      <td className="px-4 py-3">
        <div className="flex items-center gap-2.5">
          <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${STATUS_DOT[session.status] ?? 'bg-muted'}`} />
          <div className="min-w-0">
            <span className="text-sm font-medium text-primary group-hover:underline">
              Session {session.session_number}
            </span>
            {session.analysis?.summary && (
              <p className="text-xs text-muted-foreground truncate max-w-[28rem] mt-0.5 leading-tight">
                {session.analysis.summary}
              </p>
            )}
          </div>
        </div>
      </td>

      {/* Status */}
      <td className="px-4 py-3 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
        <StatusBadge status={session.status} />
      </td>

      {/* Score */}
      <td className="px-4 py-3 whitespace-nowrap">
        {session.analysis
          ? <ScoreIndicator score={session.analysis.score} />
          : <span className="text-xs text-muted-foreground/40">—</span>}
      </td>

      {/* Created */}
      <td className="px-4 py-3 whitespace-nowrap text-xs text-muted-foreground">
        {createdAt}
      </td>

      {/* Actions */}
      <td className="px-4 py-3 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          {session.status === 'pending_review' ? (
            <>
              <button
                onClick={() => onApprove(session.id)}
                disabled={isApproving || isRejecting}
                title="Approve"
                className="p-1.5 rounded-md text-success hover:bg-success/10 transition-colors disabled:opacity-50 cursor-pointer"
              >
                {isApproving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />}
              </button>
              <button
                onClick={() => onReject(session.id)}
                disabled={isApproving || isRejecting}
                title="Reject"
                className="p-1.5 rounded-md text-destructive hover:bg-destructive/10 transition-colors disabled:opacity-50 cursor-pointer"
              >
                {isRejecting ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <X className="h-3.5 w-3.5" />}
              </button>
            </>
          ) : (session.status === 'producing' || session.status === 'published') ? (
            <div className="flex items-center gap-1.5">
              {(session.workflow_results ?? []).map((wr) => (
                <span key={wr.run_id}>
                  {wr.status === 'success'
                    ? <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                    : wr.status === 'failed'
                    ? <XCircle className="h-3.5 w-3.5 text-destructive" />
                    : <Loader2 className="h-3.5 w-3.5 text-info animate-spin" />}
                </span>
              ))}
            </div>
          ) : null}
          <button
            className="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer"
            title="More options"
          >
            <MoreHorizontal className="h-3.5 w-3.5" />
          </button>
        </div>
      </td>
    </tr>
  )
}

// ─── Session filter type ───────────────────────────────────────────────────────

type SessionFilter = 'all' | 'pending_review' | 'producing' | 'published' | 'rejected'

// ─── Page ──────────────────────────────────────────────────────────────────────

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
  const [autoSaveStatus, setAutoSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle')

  useEffect(() => {
    if (pipeline) {
      setLocalSources(pipeline.sources ?? [])
      setLocalSchedule(pipeline.schedule ?? '')
    }
  }, [pipeline?.id])

  // Refs to read latest values in effects/cleanup without stale closures
  const pipelineRef = useRef(pipeline)
  pipelineRef.current = pipeline
  const localSourcesRef = useRef(localSources)
  localSourcesRef.current = localSources
  const localScheduleRef = useRef(localSchedule)
  localScheduleRef.current = localSchedule

  // Whether local values differ from server state
  const isDirty = useMemo(() => {
    if (!pipeline) return false
    return (
      JSON.stringify(localSources) !== JSON.stringify(pipeline.sources ?? []) ||
      localSchedule !== (pipeline.schedule ?? '')
    )
  }, [localSources, localSchedule, pipeline])

  const isDirtyRef = useRef(isDirty)
  isDirtyRef.current = isDirty

  const doSave = async () => {
    const p = pipelineRef.current
    if (!p) return
    setAutoSaveStatus('saving')
    try {
      await updatePipeline(id!, { ...p, sources: localSourcesRef.current, schedule: localScheduleRef.current })
      queryClient.invalidateQueries({ queryKey: ['pipeline', id] })
      setAutoSaveStatus('saved')
      setTimeout(() => setAutoSaveStatus('idle'), 2000)
    } catch {
      setAutoSaveStatus('idle')
    }
  }

  const doSaveRef = useRef(doSave)
  doSaveRef.current = doSave

  // Debounced auto-save on change
  useEffect(() => {
    if (!isDirty) return
    const timer = setTimeout(() => { void doSaveRef.current() }, 800)
    return () => clearTimeout(timer)
  }, [localSources, localSchedule, isDirty])

  // Save on unmount if dirty
  useEffect(() => {
    return () => {
      if (isDirtyRef.current) void doSaveRef.current()
    }
  }, [])

  const collectMutation = useMutation({
    mutationFn: () => collectPipeline(id!),
    onSuccess: ({ session_id }) => {
      setShowNewSession(false)
      navigate(`/pipelines/${id}/sessions/${session_id}`)
    },
  })

  const updateContextMutation = useMutation({
    mutationFn: (ctx: PipelineContext) => updatePipeline(id!, { ...pipeline!, context: ctx }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipeline', id] }),
  })

  const [showNewSession, setShowNewSession] = useState(false)

  const [approvingId, setApprovingId] = useState<string | null>(null)
  const [rejectingId, setRejectingId] = useState<string | null>(null)
  const [activeFilter, setActiveFilter] = useState<SessionFilter>('all')
  const [search, setSearch] = useState('')

  const { data: sessions = [], isLoading: sessionsLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId: id }],
    queryFn: () => fetchContentSessions({ pipelineId: id }),
    enabled: !!id,
  })

  const handleApprove = async (sid: string) => {
    setApprovingId(sid)
    try {
      await useContentSessionStore.getState().approveSession(sid, [])
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    } finally { setApprovingId(null) }
  }

  const handleReject = async (sid: string) => {
    setRejectingId(sid)
    try {
      await useContentSessionStore.getState().rejectSession(sid)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    } finally { setRejectingId(null) }
  }

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

  const filteredSessions = sessions
    .filter(s => activeFilter === 'all' || s.status === activeFilter)
    .filter(s => {
      if (!search) return true
      const q = search.toLowerCase()
      return `session ${s.session_number}`.includes(q) ||
        s.analysis?.summary?.toLowerCase().includes(q) ||
        s.status.includes(q)
    })

  if (isLoading || !pipeline) {
    return (
      <MainLayout headerContent={<span className="font-semibold text-muted-foreground">Loading…</span>}>
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
          <span className="text-muted-foreground/40 text-xs shrink-0">/</span>
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
          onContextSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
          autoSaveStatus={autoSaveStatus}
        />
      }
    >
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto px-6 py-8">

          {/* ── Page header ─────────────────────────────────────────────── */}
          <div className="flex items-start justify-between gap-4 mb-8">
            <div>
              <h1 className="text-2xl font-bold tracking-tight">{pipeline.name}</h1>
              {pipeline.description && (
                <p className="text-sm text-muted-foreground mt-1">{pipeline.description}</p>
              )}
            </div>

            <button
              onClick={() => setShowNewSession(true)}
              className="flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-medium
                bg-foreground text-background hover:opacity-90 transition-opacity
                cursor-pointer shrink-0"
            >
              <Plus className="h-3.5 w-3.5" />New Session
            </button>
          </div>

          {/* ── Sessions section ─────────────────────────────────────────── */}
          <div className="flex items-center gap-3 mb-4">
            {/* Filter tabs */}
            <div className="flex items-center gap-0.5">
              {filterTabs.map(tab => {
                const count = filterCounts[tab.value]
                const isActive = activeFilter === tab.value
                return (
                  <button
                    key={tab.value}
                    onClick={() => setActiveFilter(tab.value)}
                    className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium transition-colors cursor-pointer
                      ${isActive
                        ? 'bg-foreground/10 text-foreground'
                        : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}`}
                  >
                    {tab.label}
                    {count > 0 && (
                      <span className={`text-[10px] font-semibold tabular-nums
                        ${tab.value === 'pending_review' && !isActive ? 'text-warning' : ''}`}>
                        {count}
                      </span>
                    )}
                  </button>
                )
              })}
            </div>

            <div className="flex-1" />

            {/* Search */}
            <div className="relative w-56">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none" />
              <input
                type="search"
                value={search}
                onChange={e => setSearch(e.target.value)}
                placeholder="Search sessions…"
                className="w-full pl-8 pr-3 py-1.5 rounded-lg border border-input bg-background
                  text-xs outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground"
              />
            </div>
          </div>

          {/* ── Table ────────────────────────────────────────────────────── */}
          {sessionsLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : sessions.length === 0 ? (
            <div className="text-center py-16 border border-dashed border-border/50 rounded-2xl">
              <p className="text-sm text-muted-foreground">No sessions yet. Click <strong>New Session</strong> to get started.</p>
            </div>
          ) : (
            <div className="border border-border/50 rounded-xl overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border/50 bg-muted/20">
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Session</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Status</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Score</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground">Created</th>
                    <th className="px-4 py-2.5 text-left text-xs font-medium text-muted-foreground w-24"></th>
                  </tr>
                </thead>
                <tbody>
                  {filteredSessions.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-4 py-10 text-center text-sm text-muted-foreground">
                        {search ? `No sessions matching "${search}"` : 'No sessions with this status.'}
                      </td>
                    </tr>
                  ) : (
                    filteredSessions.map(session => (
                      <SessionRow
                        key={session.id}
                        session={session}
                        approvingId={approvingId}
                        rejectingId={rejectingId}
                        onClick={() => navigate(`/pipelines/${id}/sessions/${session.id}`)}
                        onApprove={handleApprove}
                        onReject={handleReject}
                      />
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}

        </div>
      </div>

      {/* New Session Modal */}
      {showNewSession && (
        <NewSessionModal
          isPending={collectMutation.isPending}
          onConfirm={() => collectMutation.mutate()}
          onClose={() => setShowNewSession(false)}
        />
      )}

    </MainLayout>
  )
}
