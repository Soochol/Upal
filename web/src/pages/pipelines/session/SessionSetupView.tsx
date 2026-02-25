import { useState, useEffect, useRef, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Trash2, Clock, Loader2, CloudUpload, Play,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { EditorialBriefForm } from '@/features/define-editorial-brief/EditorialBriefForm'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { WorkflowPicker } from '../WorkflowPicker'
import {
  fetchContentSession,
  updateSessionSettings,
  collectSession,
} from '@/entities/content-session/api'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { fetchPipeline } from '@/entities/pipeline'
import { useUIStore } from '@/entities/ui'
import type { ContentSession } from '@/entities/content-session'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/shared/types'

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const SCHEDULE_PRESETS: { label: string; cron: string }[] = [
  { label: 'Every hour', cron: '0 * * * *' },
  { label: 'Every 6 hours', cron: '0 */6 * * *' },
  { label: 'Every 12 hours', cron: '0 */12 * * *' },
  { label: 'Daily at 09:00', cron: '0 9 * * *' },
  { label: 'Weekdays at 09:00', cron: '0 9 * * 1-5' },
  { label: 'Weekly (Mon 09:00)', cron: '0 9 * * 1' },
  { label: 'Monthly (1st 09:00)', cron: '0 9 1 * *' },
]

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

type Props = {
  sessionId: string
  pipelineId: string
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function SessionSetupView({ sessionId, pipelineId }: Props) {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)

  // ─── Data fetching ────────────────────────────────────────────────────

  const { data: session } = useQuery<ContentSession>({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId),
    enabled: !!sessionId,
    refetchInterval: (query) => {
      const s = query.state.data
      if (!s) return false
      if (s.status === 'collecting' || s.status === 'analyzing' || s.status === 'producing') return 3000
      if (s.status === 'approved' && (!s.workflow_results || s.workflow_results.length === 0)) return 3000
      return false
    },
  })

  const { data: pipeline } = useQuery({
    queryKey: ['pipeline', pipelineId],
    queryFn: () => fetchPipeline(pipelineId),
    enabled: !!pipelineId,
  })

  const { data: channels = [] } = useQuery({
    queryKey: ['publish-channels'],
    queryFn: fetchPublishChannels,
  })

  // ─── Local settings state ─────────────────────────────────────────────

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')
  const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
  const [localModel, setLocalModel] = useState('')
  const [autoSaveStatus, setAutoSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle')

  const [showAddModal, setShowAddModal] = useState(false)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)

  // Determine effective values: session-level overrides, fallback to pipeline defaults
  const effectiveSources = useMemo(
    () => session?.session_sources ?? pipeline?.sources ?? [],
    [session?.session_sources, pipeline?.sources],
  )
  const effectiveSchedule = session?.schedule ?? pipeline?.schedule ?? ''
  const effectiveWorkflows = useMemo(
    () => session?.session_workflows ?? pipeline?.workflows ?? [],
    [session?.session_workflows, pipeline?.workflows],
  )
  const effectiveModel = session?.model ?? pipeline?.model ?? ''

  // Sync server -> local state when session changes
  useEffect(() => {
    setLocalSources(effectiveSources) // eslint-disable-line react-hooks/set-state-in-effect
    setLocalSchedule(effectiveSchedule)
    setLocalWorkflows(effectiveWorkflows)
    setLocalModel(effectiveModel)
  }, [sessionId]) // eslint-disable-line react-hooks/exhaustive-deps

  // Refs to read latest values in effects without stale closures
  const localSourcesRef = useRef(localSources)
  const localScheduleRef = useRef(localSchedule)
  const localWorkflowsRef = useRef(localWorkflows)
  const localModelRef = useRef(localModel)

  useEffect(() => {
    localSourcesRef.current = localSources
    localScheduleRef.current = localSchedule
    localWorkflowsRef.current = localWorkflows
    localModelRef.current = localModel
  })

  const isDirty = useMemo(() => {
    return (
      JSON.stringify(localSources) !== JSON.stringify(effectiveSources) ||
      localSchedule !== effectiveSchedule ||
      JSON.stringify(localWorkflows) !== JSON.stringify(effectiveWorkflows) ||
      localModel !== effectiveModel
    )
  }, [localSources, localSchedule, localWorkflows, localModel, effectiveSources, effectiveSchedule, effectiveWorkflows, effectiveModel])

  const isDirtyRef = useRef(isDirty)
  useEffect(() => { isDirtyRef.current = isDirty })

  // ─── Auto-save ────────────────────────────────────────────────────────

  const doSave = async () => {
    setAutoSaveStatus('saving')
    try {
      await updateSessionSettings(sessionId, {
        sources: localSourcesRef.current,
        schedule: localScheduleRef.current,
        workflows: localWorkflowsRef.current,
        model: localModelRef.current,
      })
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
      setAutoSaveStatus('saved')
      setTimeout(() => setAutoSaveStatus('idle'), 2000)
    } catch (err) {
      setAutoSaveStatus('idle')
      addToast(`Failed to save session settings: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }

  const doSaveRef = useRef(doSave)
  useEffect(() => { doSaveRef.current = doSave })

  // Debounced auto-save on change (800ms)
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

  // ─── Context save (editorial brief) ───────────────────────────────────

  const handleContextSave = async (ctx: PipelineContext) => {
    await updateSessionSettings(sessionId, { context: ctx })
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
  }

  // ─── Start session (collect) ──────────────────────────────────────────

  const collectMutation = useMutation({
    mutationFn: () => collectSession(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
    },
  })

  const isDraft = session?.status === 'draft'

  // ─── Render ───────────────────────────────────────────────────────────

  if (!session) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden bg-background">
      {/* Save status indicator */}
      <div className="px-6 py-2 border-b border-border/50 bg-background/80 backdrop-blur-sm flex items-center justify-between shrink-0">
        <span className="text-xs text-muted-foreground">
          Session #{session.session_number ?? '—'}
        </span>
        <div className="h-4">
          {autoSaveStatus === 'saving' && (
            <span className="flex items-center gap-1 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />Saving...
            </span>
          )}
          {autoSaveStatus === 'saved' && (
            <span className="flex items-center gap-1 text-xs text-success">
              <CloudUpload className="h-3 w-3" />Saved
            </span>
          )}
        </div>
      </div>

      {/* Scrollable sections */}
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto px-6 py-8 space-y-8">

          {/* ── Section 1: Sources & Schedule ── */}
          <section>
            <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-4">
              Sources &amp; Schedule
            </h3>
            <div className="rounded-xl border border-border/50 bg-card/50 p-5 space-y-4">
              {localSources.length === 0 ? (
                <div className="py-6 text-center rounded-xl border border-dashed border-border">
                  <p className="text-sm text-muted-foreground mb-3">No sources configured.</p>
                  <button
                    onClick={() => setShowAddModal(true)}
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                  >
                    <Plus className="h-3.5 w-3.5" /> Add source
                  </button>
                </div>
              ) : (
                <>
                  <div className="rounded-xl border border-border overflow-hidden">
                    {localSources.map((src, i) => (
                      <div key={src.id} className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0">
                        <SourceTypeBadge type={src.source_type} />
                        <span className="text-sm font-medium flex-1 truncate">{src.label}</span>
                        <button
                          onClick={() => setLocalSources(localSources.filter((_, j) => j !== i))}
                          className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                  <button
                    onClick={() => setShowAddModal(true)}
                    className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                  >
                    <Plus className="h-3 w-3" /> Add source
                  </button>
                </>
              )}

              {/* Schedule */}
              <div className="pt-2 border-t border-border/50">
                <label className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground mb-2">
                  <Clock className="h-3 w-3" />Schedule
                </label>
                <select
                  value={SCHEDULE_PRESETS.some(p => p.cron === localSchedule) ? localSchedule : '__custom__'}
                  onChange={(e) => setLocalSchedule(e.target.value === '__custom__' ? '' : e.target.value)}
                  className="w-full rounded-xl border border-input bg-background px-3 py-2 text-sm outline-none focus:ring-1 focus:ring-ring cursor-pointer"
                >
                  <option value="" disabled>Select schedule...</option>
                  {SCHEDULE_PRESETS.map(p => <option key={p.cron} value={p.cron}>{p.label}</option>)}
                  <option value="__custom__">Custom cron...</option>
                </select>
                <input
                  type="text"
                  value={localSchedule}
                  onChange={(e) => setLocalSchedule(e.target.value)}
                  placeholder="0 */6 * * *"
                  readOnly={SCHEDULE_PRESETS.some(p => p.cron === localSchedule)}
                  className={cn(
                    'w-full rounded-xl border border-input bg-background px-3 py-2 text-sm font-mono outline-none focus:ring-1 focus:ring-ring placeholder:text-muted-foreground mt-2',
                    SCHEDULE_PRESETS.some(p => p.cron === localSchedule) && 'text-muted-foreground',
                  )}
                />
              </div>
            </div>
          </section>

          {/* ── Section 2: Editorial Brief ── */}
          <section>
            <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-4">
              Editorial Brief
            </h3>
            <div className="rounded-xl border border-border/50 bg-card/50 p-5">
              <EditorialBriefForm
                initialContext={session.context ?? pipeline?.context}
                onSave={handleContextSave}
                autoSave
              />
            </div>
          </section>

          {/* ── Section 3: Analysis Model ── */}
          <section>
            <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-4">
              Analysis Model
            </h3>
            <div className="rounded-xl border border-border/50 bg-card/50 p-5 space-y-3">
              <p className="text-xs text-muted-foreground">
                Model used for content analysis. Leave empty for system default.
              </p>
              <ModelSelector
                key={localModel || '__default__'}
                value={localModel}
                onChange={setLocalModel}
                placeholder="System Default"
              />
              {localModel && (
                <button
                  onClick={() => setLocalModel('')}
                  className="text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  Reset to default
                </button>
              )}
            </div>
          </section>

          {/* ── Section 4: Workflows ── */}
          <section>
            <h3 className="text-xs font-bold uppercase tracking-widest text-muted-foreground mb-4">
              Workflows
            </h3>
            <div className="rounded-xl border border-border/50 bg-card/50 p-5 space-y-4">
              {localWorkflows.length === 0 ? (
                <div className="py-6 text-center rounded-xl border border-dashed border-border">
                  <p className="text-sm text-muted-foreground mb-3">No workflows configured.</p>
                  <button
                    onClick={() => setShowWorkflowPicker(true)}
                    className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                  >
                    <Plus className="h-3.5 w-3.5" /> Add workflow
                  </button>
                </div>
              ) : (
                <>
                  <div className="rounded-xl border border-border overflow-hidden">
                    {localWorkflows.map((wf, i) => (
                      <div key={wf.workflow_name} className="flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0">
                        <a
                          href={`/workflows?w=${encodeURIComponent(wf.workflow_name)}`}
                          className="text-sm font-medium flex-1 truncate hover:text-primary hover:underline transition-colors"
                        >
                          {wf.label || wf.workflow_name}
                        </a>
                        {channels.length > 0 && (
                          <select
                            value={wf.channel_id || ''}
                            onChange={(e) => {
                              const updated = [...localWorkflows]
                              updated[i] = { ...updated[i], channel_id: e.target.value || undefined }
                              setLocalWorkflows(updated)
                            }}
                            className="w-32 rounded-lg border border-input bg-background px-2 py-1.5 text-xs outline-none focus:ring-1 focus:ring-ring cursor-pointer"
                          >
                            <option value="">No channel</option>
                            {channels.map(ch => (
                              <option key={ch.id} value={ch.id}>{ch.name}</option>
                            ))}
                          </select>
                        )}
                        <button
                          onClick={() => setLocalWorkflows(localWorkflows.filter((_, j) => j !== i))}
                          className="text-muted-foreground hover:text-destructive transition-colors cursor-pointer"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                  <button
                    onClick={() => setShowWorkflowPicker(true)}
                    className="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                  >
                    <Plus className="h-3 w-3" /> Add workflow
                  </button>
                </>
              )}
            </div>
          </section>

        </div>
      </div>

      {/* ── Sticky bottom: Start Session ── */}
      {isDraft && (
        <div className="shrink-0 px-6 py-4 border-t border-border bg-background/80 backdrop-blur-sm">
          <div className="max-w-3xl mx-auto flex items-center justify-between">
            <p className="text-xs text-muted-foreground">
              {localSources.length === 0
                ? 'Add at least one source to start collection.'
                : `${localSources.length} source${localSources.length !== 1 ? 's' : ''} configured`}
            </p>
            <button
              onClick={() => collectMutation.mutate()}
              disabled={localSources.length === 0 || collectMutation.isPending}
              className="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold bg-foreground text-background hover:opacity-90 transition-opacity disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed shadow-md"
            >
              {collectMutation.isPending ? (
                <><Loader2 className="h-4 w-4 animate-spin" />Starting...</>
              ) : (
                <><Play className="h-4 w-4" />Start Session</>
              )}
            </button>
          </div>
        </div>
      )}

      {/* ── Modals ── */}
      {showAddModal && (
        <AddSourceModal
          onAdd={(src) => setLocalSources([...localSources, src])}
          onClose={() => setShowAddModal(false)}
        />
      )}
      {showWorkflowPicker && (
        <WorkflowPicker
          existingWorkflows={localWorkflows}
          onAdd={(wf) => setLocalWorkflows([...localWorkflows, wf])}
          onClose={() => setShowWorkflowPicker(false)}
        />
      )}
    </div>
  )
}
