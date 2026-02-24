import { useState, useEffect, useRef, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Loader2, Trash2, Plus, Clock,
  X, Search, ChevronDown, CloudUpload, PanelRightClose, PanelRightOpen,
} from 'lucide-react'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { MainLayout } from '@/app/layout'
import { fetchPipeline, updatePipeline, collectPipeline } from '@/entities/pipeline'
import { fetchContentSessions } from '@/entities/content-session/api'
import { WorkflowPicker } from './WorkflowPicker'
import { SessionDetailPreview } from './session/SessionDetailPreview'
import type { PipelineSource, PipelineContext, PipelineWorkflow } from '@/shared/types'

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
  sources, schedule, context, workflows, model,
  onSourcesChange, onScheduleChange, onContextSave, onWorkflowsChange, onModelChange, autoSaveStatus,
}: {
  sources: PipelineSource[]
  schedule: string
  context: PipelineContext | undefined
  workflows: PipelineWorkflow[]
  model: string
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (cron: string) => void
  onContextSave: (ctx: PipelineContext) => Promise<void>
  onWorkflowsChange: (w: PipelineWorkflow[]) => void
  onModelChange: (model: string) => void
  autoSaveStatus: 'idle' | 'saving' | 'saved'
}) {
  const [showAddModal, setShowAddModal] = useState(false)
  const [sourcesOpen, setSourcesOpen] = useState(true)
  const [briefOpen, setBriefOpen] = useState(false)
  const [workflowsOpen, setWorkflowsOpen] = useState(false)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)
  const [modelOpen, setModelOpen] = useState(false)

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

        <section>
          <button
            onClick={() => setModelOpen(v => !v)}
            className="w-full flex items-center justify-between mb-2 cursor-pointer"
          >
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">
              Analysis Model
            </span>
            <ChevronDown className={`h-3.5 w-3.5 text-muted-foreground transition-transform duration-200 ${modelOpen ? 'rotate-180' : ''}`} />
          </button>
          {modelOpen && (
            <div className="space-y-2">
              <p className="text-[11px] text-muted-foreground">
                Model used for content analysis. Leave empty for system default.
              </p>
              <ModelSelector
                value={model}
                onChange={onModelChange}
                placeholder="System Default"
              />
              {model && (
                <button
                  onClick={() => onModelChange('')}
                  className="text-[11px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  Reset to default
                </button>
              )}
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
  onConfirm: (config: { isTest: boolean, limit: number }) => void
  onClose: () => void
}) {
  const [isTest, setIsTest] = useState(true)
  const [limit, setLimit] = useState(5)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-sm mx-4 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">Start Session</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="p-5">
          <p className="text-sm text-muted-foreground mb-4">
            Start a new session to collect and analyze sources.
          </p>

          <div className="space-y-4 bg-muted/20 p-4 rounded-xl border border-border/50">
            <label className="flex items-start gap-3 cursor-pointer group">
              <input
                type="checkbox"
                checked={isTest}
                onChange={e => setIsTest(e.target.checked)}
                className="mt-1 flex-shrink-0 cursor-pointer accent-primary"
              />
              <div>
                <span className="text-sm font-medium group-hover:text-foreground transition-colors">Test Run (Dry Run)</span>
                <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">Limit the number of sources fetched to quickly test the pipeline without consuming too many credits.</p>
              </div>
            </label>

            {isTest && (
              <div className="pl-7 pt-1 animate-in fade-in slide-in-from-top-2">
                <label className="text-xs font-medium text-muted-foreground mb-1.5 block">Item limit per source</label>
                <input
                  type="number"
                  min={1}
                  max={50}
                  value={limit}
                  onChange={e => setLimit(Number(e.target.value))}
                  className="w-24 rounded-lg bg-background border border-input px-3 py-1.5 text-sm focus:ring-1 focus:ring-ring outline-none"
                />
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-border bg-muted/10">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground
              hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            onClick={() => onConfirm({ isTest, limit })}
            disabled={isPending}
            className="inline-flex items-center gap-1.5 px-5 py-2 rounded-xl text-sm font-semibold
              bg-primary text-primary-foreground hover:bg-primary/90 transition-opacity
              disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed shadow-md shadow-primary/20"
          >
            {isPending
              ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Starting…</>
              : 'Start Session'}
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
  const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
  const [localModel, setLocalModel] = useState('')
  const [autoSaveStatus, setAutoSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle')

  useEffect(() => {
    if (pipeline) {
      setLocalSources(pipeline.sources ?? [])
      setLocalSchedule(pipeline.schedule ?? '')
      setLocalWorkflows(pipeline.workflows ?? [])
      setLocalModel(pipeline.model ?? '')
    }
  }, [pipeline?.id])

  // Refs to read latest values in effects/cleanup without stale closures
  const pipelineRef = useRef(pipeline)
  pipelineRef.current = pipeline
  const localSourcesRef = useRef(localSources)
  localSourcesRef.current = localSources
  const localScheduleRef = useRef(localSchedule)
  localScheduleRef.current = localSchedule
  const localWorkflowsRef = useRef(localWorkflows)
  localWorkflowsRef.current = localWorkflows
  const localModelRef = useRef(localModel)
  localModelRef.current = localModel

  // Whether local values differ from server state
  const isDirty = useMemo(() => {
    if (!pipeline) return false
    return (
      JSON.stringify(localSources) !== JSON.stringify(pipeline.sources ?? []) ||
      localSchedule !== (pipeline.schedule ?? '') ||
      JSON.stringify(localWorkflows) !== JSON.stringify(pipeline.workflows ?? []) ||
      localModel !== (pipeline.model ?? '')
    )
  }, [localSources, localSchedule, localWorkflows, localModel, pipeline])

  const isDirtyRef = useRef(isDirty)
  isDirtyRef.current = isDirty

  const doSave = async () => {
    const p = pipelineRef.current
    if (!p) return
    setAutoSaveStatus('saving')
    try {
      await updatePipeline(id!, { ...p, sources: localSourcesRef.current, schedule: localScheduleRef.current, workflows: localWorkflowsRef.current, model: localModelRef.current })
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
  }, [localSources, localSchedule, localWorkflows, localModel, isDirty])

  // Save on unmount if dirty
  useEffect(() => {
    return () => {
      if (isDirtyRef.current) void doSaveRef.current()
    }
  }, [])

  const collectMutation = useMutation({
    mutationFn: (config?: { isTest: boolean, limit: number }) => collectPipeline(id!, config),
    onSuccess: () => {
      setShowNewSession(false)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: id }] })
    },
  })

  const updateContextMutation = useMutation({
    mutationFn: (ctx: PipelineContext) => updatePipeline(id!, { ...pipeline!, context: ctx }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipeline', id] }),
  })

  const [showNewSession, setShowNewSession] = useState(false)
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)

  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null)
  const [activeFilter, setActiveFilter] = useState<SessionFilter>('all')
  const [search, setSearch] = useState('')

  const { data: sessions = [], isLoading: sessionsLoading } = useQuery({
    queryKey: ['content-sessions', { pipelineId: id }],
    queryFn: () => fetchContentSessions({ pipelineId: id }),
    enabled: !!id,
  })

  const filterTabs: { value: SessionFilter; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'pending_review', label: 'Pending' },
    { value: 'producing', label: 'Producing' },
    { value: 'published', label: 'Published' },
    { value: 'rejected', label: 'Rejected' },
  ]

  const filterCounts = useMemo(() => {
    const counts: Record<SessionFilter, number> = { all: sessions.length, pending_review: 0, producing: 0, published: 0, rejected: 0 }
    for (const s of sessions) {
      if (s.status in counts) counts[s.status as SessionFilter]++
    }
    return counts
  }, [sessions])

  const filteredSessions = sessions
    .filter(s => activeFilter === 'all' || s.status === activeFilter)
    .filter(s => {
      if (!search) return true
      const q = search.toLowerCase()
      return `session ${s.session_number}`.includes(q) ||
        s.analysis?.summary?.toLowerCase().includes(q) ||
        s.status.includes(q)
    })

  // Auto-select first session if none is selected
  useEffect(() => {
    if (!selectedSessionId && filteredSessions.length > 0) {
      // Prefer pending_review if viewing all, else just the first one
      const pending = filteredSessions.find(s => s.status === 'pending_review')
      setSelectedSessionId((pending ?? filteredSessions[0]).id)
    }
  }, [filteredSessions, selectedSessionId])

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
        isSidebarOpen ? (
          <PipelineSettingsPanel
            sources={localSources}
            schedule={localSchedule}
            context={pipeline.context}
            workflows={localWorkflows}
            onSourcesChange={setLocalSources}
            onScheduleChange={setLocalSchedule}
            onContextSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
            onWorkflowsChange={setLocalWorkflows}
            model={localModel}
            onModelChange={setLocalModel}
            autoSaveStatus={autoSaveStatus}
          />
        ) : null
      }
    >
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto px-6 py-8">

          {/* ── Page header ─────────────────────────────────────────────── */}
          <div className="flex items-start justify-between gap-4 mb-4">
            <div>
              <h1 className="text-xl font-bold tracking-tight">{pipeline.name}</h1>
              {pipeline.description && (
                <p className="text-xs text-muted-foreground mt-0.5">{pipeline.description}</p>
              )}
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => setShowNewSession(true)}
                className="flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-medium
                  bg-foreground text-background hover:opacity-90 transition-opacity
                  cursor-pointer shrink-0"
              >
                <Plus className="h-3.5 w-3.5" />Start Session
              </button>
              <button
                onClick={() => setIsSidebarOpen(!isSidebarOpen)}
                className="p-2 rounded-xl border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                title="Toggle Settings Sidebar"
              >
                {isSidebarOpen ? <PanelRightClose className="w-4 h-4" /> : <PanelRightOpen className="w-4 h-4" />}
              </button>
            </div>
          </div>

          {/* ── Split View Layout ─────────────────────────────────────── */}
          <div className="flex h-[calc(100vh-160px)] -mx-6 mb-[-2rem] border-t border-border overflow-hidden">
            {/* Left Sidebar (Session List) */}
            <div className="w-[320px] 2xl:w-[380px] shrink-0 border-r border-border bg-sidebar/30 backdrop-blur-xl z-20 flex flex-col shadow-[4px_0_24px_-12px_rgba(0,0,0,0.5)]">
              <div className="p-4 border-b border-border/50 bg-background/50 sticky top-0 z-10">
                <div className="relative">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <input
                    type="search"
                    placeholder="Search sessions..."
                    className="w-full h-9 pl-9 pr-4 rounded-lg bg-background border border-input text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                  />
                </div>
                {/* Filter Tabs */}
                <div className="flex items-center gap-1 mt-3 overflow-x-auto pb-1 scrollbar-none">
                  {filterTabs.map(tab => {
                    const count = filterCounts[tab.value]
                    const isActive = activeFilter === tab.value
                    return (
                      <button
                        key={tab.value}
                        onClick={() => setActiveFilter(tab.value)}
                        className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium transition-colors cursor-pointer whitespace-nowrap shrink-0
                           ${isActive
                            ? 'bg-foreground text-background'
                            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}`}
                      >
                        {tab.label}
                        {count > 0 && (
                          <span className={`text-[9px] font-bold tabular-nums px-1 rounded-full
                             ${tab.value === 'pending_review' ? 'bg-warning/20 text-warning' : 'bg-muted-foreground/20'}`}>
                            {count}
                          </span>
                        )}
                      </button>
                    )
                  })}
                </div>
              </div>

              <div className="flex-1 overflow-y-auto w-full p-2 space-y-1">
                {sessionsLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <Loader2 className="w-5 h-5 animate-spin text-muted-foreground" />
                  </div>
                ) : filteredSessions.length === 0 ? (
                  <div className="text-center py-12 px-4">
                    <p className="text-xs text-muted-foreground">No sessions found.</p>
                  </div>
                ) : (
                  filteredSessions.map((s) => {
                    const isSelected = selectedSessionId === s.id
                    return (
                      <button
                        key={s.id}
                        onClick={() => setSelectedSessionId(s.id)}
                        className={`w-full text-left p-3 rounded-xl transition-all duration-200 cursor-pointer border ${isSelected
                          ? 'bg-primary/5 border-primary/20 shadow-sm'
                          : 'bg-transparent border-transparent hover:bg-muted/50'
                          }`}
                      >
                        <div className="flex items-start justify-between gap-2 mb-1">
                          <div className="flex items-center gap-1.5 min-w-0">
                            <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${STATUS_DOT[s.status] ?? 'bg-muted'}`} />
                            <span className={`text-xs font-semibold truncate ${isSelected ? 'text-primary' : 'text-foreground'}`}>
                              Session {s.session_number}
                            </span>
                          </div>
                          <span className="text-[10px] text-muted-foreground/60 whitespace-nowrap shrink-0">
                            {new Date(s.created_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
                          </span>
                        </div>
                        {s.analysis?.summary ? (
                          <p className={`text-xs line-clamp-2 ${isSelected ? 'text-foreground/80' : 'text-muted-foreground'}`}>
                            {s.analysis.summary}
                          </p>
                        ) : (
                          <p className="text-[10px] text-muted-foreground/50 italic">Processing...</p>
                        )}
                        <div className="flex items-center gap-2 mt-2">
                          <StatusBadge status={s.status} />
                          {s.status === 'pending_review' && (
                            <span className="flex h-2 w-2 rounded-full bg-warning animate-pulse ml-auto" title="Needs Review" />
                          )}
                        </div>
                      </button>
                    )
                  })
                )}
              </div>
            </div>

            {/* Right Preview */}
            <div className="flex-1 min-w-0 flex flex-col bg-grid-pattern relative">
              {selectedSessionId ? (
                <SessionDetailPreview pipelineId={id!} sessionId={selectedSessionId} />
              ) : (
                <div className="flex-1 flex flex-col items-center justify-center text-muted-foreground p-8">
                  <div className="w-16 h-16 rounded-3xl glass-panel flex items-center justify-center mb-4 shadow-xl">
                    <Search className="w-6 h-6 opacity-50" />
                  </div>
                  <p className="text-sm font-medium">No session selected</p>
                  <p className="text-xs opacity-60 mt-1 max-w-[250px] text-center">Select a session from the list to view details and approve analysis.</p>
                </div>
              )}
            </div>
          </div>

        </div>
      </div>

      {/* New Session Modal */}
      {showNewSession && (
        <NewSessionModal
          isPending={collectMutation.isPending}
          onConfirm={(config) => collectMutation.mutate(config)}
          onClose={() => setShowNewSession(false)}
        />
      )}

    </MainLayout>
  )
}
