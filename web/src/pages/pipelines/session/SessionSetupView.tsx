import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Trash2, Loader2, Play, Pencil, ChevronDown, RotateCcw, GitBranch, Power,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import { KeywordTagInput } from '@/shared/ui/KeywordTagInput'
import { FloatingConfigureChat } from '@/shared/ui/FloatingConfigureChat'
import type { ChatMessage } from '@/shared/ui/ConfigureChat'
import { AddSourceModal, STATIC_SOURCES, SIGNAL_SOURCES } from '@/features/configure-pipeline-sources/AddSourceModal'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import { WorkflowPicker } from '../WorkflowPicker'
import {
  fetchContentSession,
  updateSessionSettings,
  configureSession,
  activateSession,
  deactivateSession,
  runSessionInstance,
} from '@/entities/content-session/api'
import { sessionPollingInterval } from '@/entities/content-session/constants'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { useUIStore } from '@/entities/ui'
import type { ContentSession } from '@/entities/content-session'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/entities/pipeline'

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

const SOURCE_TYPE_MAP = Object.fromEntries(
  [...STATIC_SOURCES, ...SIGNAL_SOURCES].map(s => [s.type, s]),
)

type Props = {
  sessionId: string
}

export function SessionSetupView({ sessionId }: Props) {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)

  // ─── Data fetching ────────────────────────────────────────────────────

  const { data: session } = useQuery<ContentSession>({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId),
    enabled: !!sessionId,
    refetchInterval: (query) => sessionPollingInterval(query.state.data),
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
  const [isEditingName, setIsEditingName] = useState(false)
  const [localName, setLocalName] = useState('')

  // Effective values: session-level values only (no pipeline fallback for templates)
  const effectiveSources = session?.session_sources ?? []
  const effectiveSchedule = session?.schedule ?? ''
  const effectiveWorkflows = session?.session_workflows ?? []
  const effectiveModel = session?.model ?? ''
  const effectiveContext = session?.session_context ?? DEFAULT_CONTEXT

  // Fingerprint of server-side data — changes when refetch brings new values
  const serverFingerprint = JSON.stringify([
    session?.session_sources, session?.schedule,
    session?.session_workflows, session?.model, session?.session_context,
  ])

  // Reset editing UI on session switch
  useEffect(() => {
    setEditingField(null)
    setIsEditingName(false)
    setLocalName(session?.name ?? '')
  }, [sessionId]) // eslint-disable-line react-hooks/exhaustive-deps

  // ─── Auto-save ────────────────────────────────────────────────────────

  const settingsData = useMemo(() => ({
    sources: localSources,
    schedule: localSchedule,
    workflows: localWorkflows,
    model: localModel,
    context: localContext,
  }), [localSources, localSchedule, localWorkflows, localModel, localContext])

  const { markClean } = useAutoSave({
    data: settingsData,
    onSave: async (data) => {
      await updateSessionSettings(sessionId, data)
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    },
    delay: 2000,
    saveOnUnmount: true,
    onBeforeUnloadSave: (data) => {
      fetch(`/api/content-sessions/${sessionId}/settings`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        keepalive: true,
        body: JSON.stringify(data),
      })
    },
    onError: (err) => addToast(`Failed to save: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  // Sync server -> local state when server data changes
  // (covers both session switch AND external updates like FloatingChat)
  useEffect(() => {
    setLocalSources(effectiveSources)        // eslint-disable-line react-hooks/set-state-in-effect
    setLocalSchedule(effectiveSchedule)
    setLocalWorkflows(effectiveWorkflows)
    setLocalModel(effectiveModel)
    setLocalContext(effectiveContext)
    markClean()
  }, [serverFingerprint]) // eslint-disable-line react-hooks/exhaustive-deps

  // ─── Activate / Deactivate / Run ─────────────────────────────────────

  const toggleMutation = useMutation({
    mutationFn: () => {
      if (!session) throw new Error('Session not loaded')
      return session.status === 'active'
        ? deactivateSession(sessionId)
        : activateSession(sessionId)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
    },
    onError: (err) => addToast(`Failed to toggle session: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  const runMutation = useMutation({
    mutationFn: () => runSessionInstance(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
    },
    onError: (err) => addToast(`Failed to run session: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  const isDraft = session?.status === 'draft'
  const isActive = session?.status === 'active'

  const handleNameSave = async () => {
    setIsEditingName(false)
    const trimmed = localName.trim()
    if (!trimmed || trimmed === session?.name) return
    try {
      await updateSessionSettings(sessionId, { name: trimmed })
      queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
      queryClient.invalidateQueries({ queryKey: ['content-sessions'] })
    } catch (err) {
      addToast(`Failed to rename: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }

  // ─── AI Configure ───────────────────────────────────────────────────

  const handleConfigure = useCallback(async ({ message, model, thinking, history }: {
    message: string; model: string; thinking: boolean; history: ChatMessage[]
  }) => {
    const response = await configureSession(sessionId, {
      message,
      model: model || undefined,
      thinking,
      history,
      current_sources: localSources,
      current_schedule: localSchedule,
      current_workflows: localWorkflows,
      current_model: localModel,
      current_context: localContext,
    })

    if (response.sources) setLocalSources(response.sources)
    if (response.schedule !== undefined && response.schedule !== null) setLocalSchedule(response.schedule)
    if (response.workflows) setLocalWorkflows(response.workflows)
    if (response.model !== undefined && response.model !== null) setLocalModel(response.model)
    if (response.context) setLocalContext(response.context)

    return { explanation: response.explanation || 'Settings updated.' }
  }, [sessionId, localSources, localSchedule, localWorkflows, localModel, localContext])

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
          {isEditingName ? (
            <input
              autoFocus
              type="text"
              value={localName}
              onChange={(e) => setLocalName(e.target.value)}
              onBlur={handleNameSave}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleNameSave()
                if (e.key === 'Escape') { setLocalName(session.name ?? ''); setIsEditingName(false) }
              }}
              className="text-base font-semibold bg-transparent outline-none border-b border-primary/50 py-0.5 min-w-0 flex-1 mr-3"
            />
          ) : (
            <h2
              className="text-base font-semibold cursor-pointer hover:text-primary transition-colors group flex items-center gap-1.5"
              onClick={() => { setLocalName(session.name ?? ''); setIsEditingName(true) }}
            >
              {session.name || `Session #${session.session_number ?? '\u2014'}`}
              <Pencil className="h-3 w-3 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity" />
            </h2>
          )}
          <div className="flex items-center gap-2">
            {/* Start/Stop toggle */}
            <button
              onClick={() => toggleMutation.mutate()}
              disabled={toggleMutation.isPending || (!isDraft && !isActive)}
              className={cn(
                'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed',
                isActive
                  ? 'bg-success/10 text-success border border-success/20 hover:bg-success/20'
                  : 'bg-muted/50 text-muted-foreground border border-border hover:bg-muted',
              )}
            >
              <Power className="h-3 w-3" />
              {toggleMutation.isPending ? 'Updating...' : isActive ? 'Active' : 'Draft'}
            </button>

            {/* Run Now */}
            <button
              onClick={() => runMutation.mutate()}
              disabled={runMutation.isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <Play className="h-3 w-3" />
              {runMutation.isPending ? 'Running...' : 'Run Now'}
            </button>
          </div>
        </div>
      </div>

      {/* ── Scrollable content ── */}
      <div className="flex-1 overflow-y-auto relative">
        <div className="max-w-2xl mx-auto px-6 py-6">

          {/* ════ SOURCES ════ */}
          <section className="mb-8">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-baseline gap-2.5">
                <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
                  Sources
                </h3>
                {localSources.length > 0 && (
                  <span className="text-xs text-muted-foreground/50">
                    {[
                      signalCount > 0 && `${signalCount} signal${signalCount !== 1 ? 's' : ''}`,
                      staticCount > 0 && `${staticCount} static`,
                    ].filter(Boolean).join(' \u00b7 ')}
                  </span>
                )}
              </div>
              {localSources.length > 0 && (
                <button
                  onClick={() => setShowAddModal(true)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                >
                  <Plus className="h-3 w-3" /> Add source
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
                {localSources.map((src, i) => {
                  const typeDef = SOURCE_TYPE_MAP[src.type]
                  return (
                  <div
                    key={src.id}
                    className="group flex items-center gap-3 py-2 -mx-2 px-2 rounded-md
                      hover:bg-muted/40 transition-colors cursor-pointer"
                    onClick={() => setEditingSourceIndex(i)}
                  >
                    {typeDef ? (
                      <div className={`w-6 h-6 rounded-md ${typeDef.accent} ${typeDef.accentText} flex items-center justify-center shrink-0 [&>svg]:h-3.5 [&>svg]:w-3.5`}>
                        {typeDef.icon}
                      </div>
                    ) : (
                      <div
                        className={cn(
                          'w-1.5 h-1.5 rounded-full shrink-0',
                          src.source_type === 'signal'
                            ? 'bg-primary shadow-[0_0_4px_var(--color-primary)]'
                            : 'bg-muted-foreground/30',
                        )}
                      />
                    )}
                    <span className="text-sm flex-1 truncate">{src.label}</span>
                    <span className="text-xs text-muted-foreground/40 opacity-0 group-hover:opacity-100 transition-opacity">
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
                  )
                })}
              </div>
            )}
          </section>

          {/* ════ SCHEDULE ════ */}
          <section className="mb-8">
            <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Schedule
            </h3>
            <div className="border-t border-border/40">
              {/* Frequency */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-28 shrink-0 text-sm text-muted-foreground">Frequency</span>
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
                <span className="w-28 shrink-0 text-sm text-muted-foreground">Cron</span>
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
            <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
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
                <span className="w-28 shrink-0 text-sm text-muted-foreground pt-1.5">Focus</span>
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
                <span className="w-28 shrink-0 text-sm text-muted-foreground pt-1.5">Exclude</span>
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
                <span className="w-28 shrink-0 text-sm text-muted-foreground">Language</span>
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
            <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Processing
            </h3>
            <div className="border-t border-border/40">
              {/* Model */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-28 shrink-0 text-sm text-muted-foreground">Analysis Model</span>
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
                  <span className="w-28 shrink-0 text-sm text-muted-foreground">Workflows</span>
                  {localWorkflows.length > 0 && (
                    <button
                      onClick={() => setShowWorkflowPicker(true)}
                      className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    >
                      <Plus className="h-3 w-3" /> Add
                    </button>
                  )}
                </div>

                {localWorkflows.length === 0 ? (
                  <div className="ml-28 py-3">
                    <button
                      onClick={() => setShowWorkflowPicker(true)}
                      className="text-xs text-muted-foreground/40 hover:text-foreground transition-colors cursor-pointer"
                    >
                      + Add workflow
                    </button>
                  </div>
                ) : (
                  <div className="ml-28 mt-1.5 space-y-0.5">
                    {localWorkflows.map((wf, i) => (
                      <div key={wf.workflow_name} className="group flex items-center gap-2 py-1">
                        <div className="w-5 h-5 rounded-md bg-card border border-white/5 flex items-center justify-center shrink-0">
                          <GitBranch className="w-3 h-3 text-blue-400" />
                        </div>
                        <a
                          href={`/workflows?w=${encodeURIComponent(wf.workflow_name)}`}
                          className="text-sm flex-1 truncate hover:text-primary transition-colors"
                        >
                          {wf.workflow_name}
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
                              className="bg-transparent text-xs text-muted-foreground outline-none cursor-pointer
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

        {/* ── Floating AI Configure ── */}
        <FloatingConfigureChat
          key={sessionId}
          onSubmit={handleConfigure}
          placeholder="Describe your session settings..."
        />
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

type InlineTextFieldProps = {
  label: string
  value: string
  onChange: (v: string) => void
  placeholder?: string
  multiline?: boolean
  editing: boolean
  onStartEdit: () => void
  onEndEdit: () => void
}

function InlineTextField({
  label, value, onChange, placeholder, multiline,
  editing, onStartEdit, onEndEdit,
}: InlineTextFieldProps) {
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
        <span className="w-28 shrink-0 text-sm text-muted-foreground pt-0.5">{label}</span>
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
      <span className="w-28 shrink-0 text-sm text-muted-foreground">{label}</span>
      {value ? (
        <span className="text-sm flex-1 truncate">{value}</span>
      ) : (
        <span className="text-sm flex-1 text-muted-foreground/30 italic">{placeholder}</span>
      )}
      <Pencil className="h-3 w-3 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-opacity shrink-0 ml-2" />
    </div>
  )
}
