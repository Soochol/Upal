import { useState, useEffect, useRef, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Trash2, Loader2, Play, Pencil, ChevronDown, RotateCcw,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
import { WorkflowPicker } from '../WorkflowPicker'
import {
  fetchContentSession,
  updateSessionSettings,
  collectSession,
} from '@/entities/content-session/api'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
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

const LANGUAGE_OPTIONS = ['Korean', 'English', 'Japanese', 'Chinese']

const DEFAULT_CONTEXT: PipelineContext = {
  purpose: '', target_audience: '', tone_style: '',
  focus_keywords: [], exclude_keywords: [], language: 'Korean',
}

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


  const { data: channels = [] } = useQuery({
    queryKey: ['publish-channels'],
    queryFn: fetchPublishChannels,
  })

  // ─── Local settings state ─────────────────────────────────────────────

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')
  const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
  const [localModel, setLocalModel] = useState('')
  const [localContext, setLocalContext] = useState<PipelineContext>(DEFAULT_CONTEXT)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editingSourceIndex, setEditingSourceIndex] = useState<number | null>(null)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)
  const [editingField, setEditingField] = useState<string | null>(null)

  // Effective values: session-level values only (no pipeline fallback for templates)
  const effectiveSources = useMemo(
    () => session?.session_sources ?? [],
    [session?.session_sources],
  )
  const effectiveSchedule = session?.schedule ?? ''
  const effectiveWorkflows = useMemo(
    () => session?.session_workflows ?? [],
    [session?.session_workflows],
  )
  const effectiveModel = session?.model ?? ''
  const effectiveContext = session?.context ?? DEFAULT_CONTEXT

  // Sync server -> local state when session changes
  useEffect(() => {
    setLocalSources(effectiveSources)        // eslint-disable-line react-hooks/set-state-in-effect
    setLocalSchedule(effectiveSchedule)
    setLocalWorkflows(effectiveWorkflows)
    setLocalModel(effectiveModel)
    setLocalContext(effectiveContext)
    setEditingField(null)
  }, [sessionId]) // eslint-disable-line react-hooks/exhaustive-deps

  // Refs for latest values in effects
  const localSourcesRef = useRef(localSources)
  const localScheduleRef = useRef(localSchedule)
  const localWorkflowsRef = useRef(localWorkflows)
  const localModelRef = useRef(localModel)
  const localContextRef = useRef(localContext)

  useEffect(() => {
    localSourcesRef.current = localSources
    localScheduleRef.current = localSchedule
    localWorkflowsRef.current = localWorkflows
    localModelRef.current = localModel
    localContextRef.current = localContext
  })

  const isDirty = useMemo(() => {
    return (
      JSON.stringify(localSources) !== JSON.stringify(effectiveSources) ||
      localSchedule !== effectiveSchedule ||
      JSON.stringify(localWorkflows) !== JSON.stringify(effectiveWorkflows) ||
      localModel !== effectiveModel ||
      JSON.stringify(localContext) !== JSON.stringify(effectiveContext)
    )
  }, [localSources, localSchedule, localWorkflows, localModel, localContext, effectiveSources, effectiveSchedule, effectiveWorkflows, effectiveModel, effectiveContext])

  const isDirtyRef = useRef(isDirty)
  useEffect(() => { isDirtyRef.current = isDirty })

  // ─── Auto-save ────────────────────────────────────────────────────────

  const doSave = async () => {
    try {
      await updateSessionSettings(sessionId, {
        sources: localSourcesRef.current,
        schedule: localScheduleRef.current,
        workflows: localWorkflowsRef.current,
        model: localModelRef.current,
        context: localContextRef.current,
      })
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    } catch (err) {
      addToast(`Failed to save: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }

  const doSaveRef = useRef(doSave)
  useEffect(() => { doSaveRef.current = doSave })

  // Debounced auto-save (800ms)
  useEffect(() => {
    if (!isDirty) return
    const timer = setTimeout(() => { void doSaveRef.current() }, 800)
    return () => clearTimeout(timer)
  }, [localSources, localSchedule, localWorkflows, localModel, localContext, isDirty])

  // Save on unmount if dirty
  useEffect(() => {
    return () => { if (isDirtyRef.current) void doSaveRef.current() }
  }, [])

  // ─── Start session (collect) ──────────────────────────────────────────

  const collectMutation = useMutation({
    mutationFn: () => collectSession(sessionId),
    onSuccess: () => {
      addToast('수집을 시작했습니다. Inbox에서 결과를 확인하세요.')
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
    },
  })

  const isDraft = session?.status === 'draft'

  // ─── Derived ──────────────────────────────────────────────────────────

  const signalCount = localSources.filter(s => s.source_type === 'signal').length
  const staticCount = localSources.filter(s => s.source_type === 'static').length
  const isPreset = SCHEDULE_PRESETS.some(p => p.cron === localSchedule)

  // ─── Render ───────────────────────────────────────────────────────────

  if (!session) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">

      {/* ── Header ── */}
      <div className="p-4 border-b border-border/50 bg-background/50 sticky top-0 z-10 shrink-0">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">
            {session.name || `Session #${session.session_number ?? '\u2014'}`}
          </h2>
          {isDraft && (
            <button
              onClick={() => collectMutation.mutate()}
              disabled={localSources.length === 0 || collectMutation.isPending}
              className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium
                bg-foreground text-background hover:opacity-90 transition-opacity
                disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed shrink-0"
            >
              {collectMutation.isPending ? (
                <><Loader2 className="h-3 w-3 animate-spin" />Starting...</>
              ) : (
                <><Play className="h-3 w-3" />Start Session</>
              )}
            </button>
          )}
        </div>
      </div>

      {/* ── Scrollable content ── */}
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto px-6 py-6">

          {/* ════ SOURCES ════ */}
          <section className="mb-8">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-baseline gap-2.5">
                <h3 className="text-[11px] font-semibold uppercase tracking-widest text-muted-foreground">
                  Sources
                </h3>
                {localSources.length > 0 && (
                  <span className="text-[11px] text-muted-foreground/50">
                    {signalCount > 0 && `${signalCount} signal${signalCount !== 1 ? 's' : ''}`}
                    {signalCount > 0 && staticCount > 0 && ' \u00b7 '}
                    {staticCount > 0 && `${staticCount} static`}
                  </span>
                )}
              </div>
              {localSources.length > 0 && (
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  <Plus className="h-3 w-3" /> Add
                </button>
              )}
            </div>

            {localSources.length === 0 ? (
              <div className="border-t border-border/40 py-10 text-center">
                <button
                  onClick={() => setShowAddModal(true)}
                  className="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-medium rounded-lg
                    bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                >
                  <Plus className="h-3 w-3" /> Add source
                </button>
              </div>
            ) : (
              <div className="border-t border-border/40">
                {localSources.map((src, i) => (
                  <div
                    key={src.id}
                    className="group flex items-center gap-3 py-2 -mx-2 px-2 rounded-md
                      hover:bg-muted/40 transition-colors cursor-pointer"
                    onClick={() => setEditingSourceIndex(i)}
                  >
                    <div
                      className={cn(
                        'w-1.5 h-1.5 rounded-full shrink-0',
                        src.source_type === 'signal'
                          ? 'bg-primary shadow-[0_0_4px_var(--color-primary)]'
                          : 'bg-muted-foreground/30',
                      )}
                    />
                    <span className="text-[13px] flex-1 truncate">{src.label}</span>
                    <span className="text-[10px] text-muted-foreground/40 opacity-0 group-hover:opacity-100 transition-opacity">
                      {src.source_type}
                    </span>
                    <button
                      onClick={(e) => { e.stopPropagation(); setLocalSources(localSources.filter((_, j) => j !== i)) }}
                      className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive
                        transition-all cursor-pointer p-0.5"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </section>

          {/* ════ SCHEDULE ════ */}
          <section className="mb-8">
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Schedule
            </h3>
            <div className="border-t border-border/40">
              {/* Frequency */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-24 shrink-0 text-xs text-muted-foreground">Frequency</span>
                <div className="relative flex-1">
                  <select
                    value={isPreset ? localSchedule : '__custom__'}
                    onChange={(e) => setLocalSchedule(e.target.value === '__custom__' ? '' : e.target.value)}
                    className="w-full bg-transparent text-sm outline-none cursor-pointer
                      appearance-none [-webkit-appearance:none] pr-5"
                  >
                    <option value="" disabled>Select...</option>
                    {SCHEDULE_PRESETS.map(p => <option key={p.cron} value={p.cron}>{p.label}</option>)}
                    <option value="__custom__">Custom cron</option>
                  </select>
                  <ChevronDown className="absolute right-0 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground/40 pointer-events-none" />
                </div>
              </div>
              {/* Cron expression */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-24 shrink-0 text-xs text-muted-foreground">Cron</span>
                <input
                  type="text"
                  value={localSchedule}
                  onChange={(e) => setLocalSchedule(e.target.value)}
                  placeholder="0 */6 * * *"
                  readOnly={isPreset}
                  className={cn(
                    'flex-1 bg-transparent text-sm font-mono outline-none placeholder:text-muted-foreground/30',
                    isPreset && 'text-muted-foreground cursor-default',
                  )}
                />
              </div>
            </div>
          </section>

          {/* ════ EDITORIAL BRIEF ════ */}
          <section className="mb-8">
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Editorial Brief
            </h3>
            <div className="border-t border-border/40">
              <InlineTextField
                label="Purpose"
                value={localContext.purpose}
                onChange={(v) => setLocalContext({ ...localContext, purpose: v })}
                placeholder="Topic and goal of this pipeline..."
                multiline
                editing={editingField === 'purpose'}
                onStartEdit={() => setEditingField('purpose')}
                onEndEdit={() => setEditingField(null)}
              />
              <InlineTextField
                label="Audience"
                value={localContext.target_audience}
                onChange={(v) => setLocalContext({ ...localContext, target_audience: v })}
                placeholder="Target audience..."
                editing={editingField === 'audience'}
                onStartEdit={() => setEditingField('audience')}
                onEndEdit={() => setEditingField(null)}
              />
              <InlineTextField
                label="Tone"
                value={localContext.tone_style}
                onChange={(v) => setLocalContext({ ...localContext, tone_style: v })}
                placeholder="Tone and style..."
                editing={editingField === 'tone'}
                onStartEdit={() => setEditingField('tone')}
                onEndEdit={() => setEditingField(null)}
              />
              {/* Focus keywords */}
              <div className="flex items-start py-2.5 -mx-2 px-2">
                <span className="w-24 shrink-0 text-xs text-muted-foreground pt-1.5">Focus</span>
                <div className="flex-1 min-w-0">
                  <KeywordTagInput
                    keywords={localContext.focus_keywords ?? []}
                    onChange={(kws) => setLocalContext({ ...localContext, focus_keywords: kws })}
                    placeholder="Add focus keywords..."
                    className="border-0 bg-transparent px-0 py-0 min-h-[28px] rounded-none"
                  />
                </div>
              </div>
              {/* Exclude keywords */}
              <div className="flex items-start py-2.5 -mx-2 px-2">
                <span className="w-24 shrink-0 text-xs text-muted-foreground pt-1.5">Exclude</span>
                <div className="flex-1 min-w-0">
                  <KeywordTagInput
                    keywords={localContext.exclude_keywords ?? []}
                    onChange={(kws) => setLocalContext({ ...localContext, exclude_keywords: kws })}
                    placeholder="Add exclude keywords..."
                    className="border-0 bg-transparent px-0 py-0 min-h-[28px] rounded-none"
                  />
                </div>
              </div>
              {/* Language */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-24 shrink-0 text-xs text-muted-foreground">Language</span>
                <div className="relative flex-1">
                  <select
                    value={localContext.language}
                    onChange={(e) => setLocalContext({ ...localContext, language: e.target.value })}
                    className="w-full bg-transparent text-sm outline-none cursor-pointer
                      appearance-none [-webkit-appearance:none] pr-5"
                  >
                    {LANGUAGE_OPTIONS.map((lang) => (
                      <option key={lang} value={lang}>{lang}</option>
                    ))}
                  </select>
                  <ChevronDown className="absolute right-0 top-1/2 -translate-y-1/2 h-3 w-3 text-muted-foreground/40 pointer-events-none" />
                </div>
              </div>
            </div>
          </section>

          {/* ════ PROCESSING ════ */}
          <section className="mb-8">
            <h3 className="text-[11px] font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Processing
            </h3>
            <div className="border-t border-border/40">
              {/* Model */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-24 shrink-0 text-xs text-muted-foreground">Analysis Model</span>
                <div className="flex-1 min-w-0">
                  <ModelSelector
                    key={localModel || '__default__'}
                    value={localModel}
                    onChange={setLocalModel}
                    placeholder="System Default"
                  />
                </div>
                {localModel && (
                  <button
                    onClick={() => setLocalModel('')}
                    className="ml-2 text-muted-foreground/50 hover:text-foreground transition-colors cursor-pointer"
                    title="Reset to default"
                  >
                    <RotateCcw className="h-3 w-3" />
                  </button>
                )}
              </div>

              {/* Workflows */}
              <div className="py-2.5 -mx-2 px-2">
                <div className="flex items-center justify-between">
                  <span className="w-24 shrink-0 text-xs text-muted-foreground">Workflows</span>
                  {localWorkflows.length > 0 && (
                    <button
                      onClick={() => setShowWorkflowPicker(true)}
                      className="flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    >
                      <Plus className="h-3 w-3" /> Add
                    </button>
                  )}
                </div>

                {localWorkflows.length === 0 ? (
                  <div className="ml-24 py-3">
                    <button
                      onClick={() => setShowWorkflowPicker(true)}
                      className="text-[11px] text-muted-foreground/40 hover:text-foreground transition-colors cursor-pointer"
                    >
                      + Add workflow
                    </button>
                  </div>
                ) : (
                  <div className="ml-24 mt-1.5 space-y-0.5">
                    {localWorkflows.map((wf, i) => (
                      <div key={wf.workflow_name} className="group flex items-center gap-2 py-1">
                        <a
                          href={`/workflows?w=${encodeURIComponent(wf.workflow_name)}`}
                          className="text-[13px] flex-1 truncate hover:text-primary transition-colors"
                        >
                          {wf.label || wf.workflow_name}
                        </a>
                        {channels.length > 0 && (
                          <div className="relative">
                            <select
                              value={wf.channel_id || ''}
                              onChange={(e) => {
                                const updated = [...localWorkflows]
                                updated[i] = { ...updated[i], channel_id: e.target.value || undefined }
                                setLocalWorkflows(updated)
                              }}
                              className="bg-transparent text-[11px] text-muted-foreground outline-none cursor-pointer
                                appearance-none [-webkit-appearance:none] pr-4"
                            >
                              <option value="">No channel</option>
                              {channels.map(ch => (
                                <option key={ch.id} value={ch.id}>{ch.name}</option>
                              ))}
                            </select>
                            <ChevronDown className="absolute right-0 top-1/2 -translate-y-1/2 h-2.5 w-2.5 text-muted-foreground/40 pointer-events-none" />
                          </div>
                        )}
                        <button
                          onClick={() => setLocalWorkflows(localWorkflows.filter((_, j) => j !== i))}
                          className="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive
                            transition-all cursor-pointer p-0.5"
                        >
                          <Trash2 className="h-3 w-3" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </section>

        </div>
      </div>

      {/* ── Modals ── */}
      {showAddModal && (
        <AddSourceModal
          onAdd={(src) => setLocalSources([...localSources, src])}
          onClose={() => setShowAddModal(false)}
        />
      )}
      {editingSourceIndex !== null && (
        <AddSourceModal
          editSource={localSources[editingSourceIndex]}
          onAdd={(updated) => {
            const next = [...localSources]
            next[editingSourceIndex] = updated
            setLocalSources(next)
          }}
          onClose={() => setEditingSourceIndex(null)}
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

// ---------------------------------------------------------------------------
// InlineTextField — property row with click-to-edit
// ---------------------------------------------------------------------------

function InlineTextField({
  label,
  value,
  onChange,
  placeholder,
  multiline,
  editing,
  onStartEdit,
  onEndEdit,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  multiline?: boolean
  editing: boolean
  onStartEdit: () => void
  onEndEdit: () => void
}) {
  const ref = useRef<HTMLTextAreaElement & HTMLInputElement>(null)

  useEffect(() => {
    if (editing && ref.current) {
      ref.current.focus()
      const len = ref.current.value.length
      ref.current.setSelectionRange(len, len)
    }
  }, [editing])

  if (editing) {
    const sharedProps = {
      ref,
      value,
      onChange: (e: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>) => onChange(e.target.value),
      onBlur: onEndEdit,
      onKeyDown: (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') onEndEdit()
        if (e.key === 'Enter' && !multiline) onEndEdit()
      },
      placeholder,
      className: 'flex-1 bg-transparent text-sm outline-none resize-none placeholder:text-muted-foreground/30',
    }

    return (
      <div className="flex items-start py-2.5 -mx-2 px-2 bg-muted/30 rounded-md">
        <span className="w-24 shrink-0 text-xs text-muted-foreground pt-0.5">{label}</span>
        {multiline ? <textarea rows={3} {...sharedProps} /> : <input type="text" {...sharedProps} />}
      </div>
    )
  }

  return (
    <div
      className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30
        transition-colors cursor-pointer group"
      onClick={onStartEdit}
    >
      <span className="w-24 shrink-0 text-xs text-muted-foreground">{label}</span>
      {value ? (
        <span className="text-sm flex-1 truncate">{value}</span>
      ) : (
        <span className="text-sm flex-1 text-muted-foreground/30 italic">{placeholder}</span>
      )}
      <Pencil className="h-3 w-3 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity shrink-0 ml-2" />
    </div>
  )
}
