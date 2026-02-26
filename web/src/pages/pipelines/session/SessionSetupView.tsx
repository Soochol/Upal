import { useState, useEffect, useMemo, useCallback } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Trash2, Loader2, Play, Pencil, ChevronDown, ChevronRight, RotateCcw, GitBranch, Power,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { ModelSelector } from '@/shared/ui/ModelSelector'
import type { ChatMessage } from '@/entities/ui/model/chatStore'
import { useRegisterChatHandler } from '@/shared/hooks/useRegisterChatHandler'
import { AddSourceModal } from '@/features/configure-pipeline-sources/AddSourceModal'
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

const DEFAULT_CONTEXT: PipelineContext = {
  description: '', prompt: '', language: 'Korean',
}

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
  const [showAdvancedSources, setShowAdvancedSources] = useState(false)
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

  // Register global chat bar handler
  useRegisterChatHandler(handleConfigure, 'Describe your session settings...', 'Session')

  // ─── Derived ──────────────────────────────────────────────────────────

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
      <div className="flex-1 overflow-y-auto min-h-0">
        <div className="max-w-2xl mx-auto px-6 py-6">

          {/* ════ RESEARCH ════ */}
          <section className="mb-8">
            <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Research
            </h3>
            <div className="border-t border-border/40">
              {/* Mode */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-28 shrink-0 text-sm text-muted-foreground">Mode</span>
                <div className="flex gap-1">
                  {(['light', 'deep'] as const).map((depth) => (
                    <button
                      key={depth}
                      onClick={() => setLocalContext({ ...localContext, research_depth: depth })}
                      className={cn(
                        'px-3 py-1 rounded-md text-xs font-medium transition-all cursor-pointer',
                        (localContext.research_depth || 'deep') === depth
                          ? 'bg-foreground text-background'
                          : 'bg-muted/50 text-muted-foreground hover:bg-muted',
                      )}
                    >
                      {depth === 'light' ? 'Search' : 'Deep Research'}
                    </button>
                  ))}
                </div>
              </div>
              {/* Research Model */}
              <div className="flex items-center py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-28 shrink-0 text-sm text-muted-foreground">Model</span>
                <div className="flex-1 min-w-0">
                  <ModelSelector
                    key={localContext.research_model || '__research_default__'}
                    value={localContext.research_model || ''}
                    onChange={(v) => setLocalContext({ ...localContext, research_model: v })}
                    placeholder="Same as analysis"
                  />
                </div>
                {localContext.research_model && (
                  <button
                    onClick={() => setLocalContext({ ...localContext, research_model: undefined })}
                    className="ml-2 text-muted-foreground/50 hover:text-foreground transition-colors cursor-pointer"
                    title="Reset to default"
                  >
                    <RotateCcw className="h-3 w-3" />
                  </button>
                )}
              </div>
              {/* Advanced sources toggle */}
              <div className="pt-2">
                <button
                  onClick={() => setShowAdvancedSources(!showAdvancedSources)}
                  className="flex items-center gap-1 text-xs text-muted-foreground/60 hover:text-muted-foreground transition-colors cursor-pointer"
                >
                  <ChevronRight className={cn('h-3 w-3 transition-transform', showAdvancedSources && 'rotate-90')} />
                  Additional sources
                  {localSources.length > 0 && (
                    <span className="text-muted-foreground/40 ml-1">({localSources.length})</span>
                  )}
                </button>
                {showAdvancedSources && (
                  <div className="mt-2 ml-4">
                    {localSources.length > 0 && (
                      <div className="space-y-0.5 mb-2">
                        {localSources.map((src, i) => (
                          <div
                            key={src.id}
                            className="group flex items-center gap-2 py-1.5 -mx-2 px-2 rounded-md
                              hover:bg-muted/40 transition-colors cursor-pointer"
                            onClick={() => setEditingSourceIndex(i)}
                          >
                            <span className="text-sm flex-1 truncate">{src.label}</span>
                            <span className="text-xs text-muted-foreground/40">{src.type}</span>
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
                    <button
                      onClick={() => setShowAddModal(true)}
                      className="flex items-center gap-1 text-xs text-muted-foreground/40 hover:text-foreground transition-colors cursor-pointer"
                    >
                      <Plus className="h-3 w-3" /> Add source (RSS, URL...)
                    </button>
                  </div>
                )}
              </div>
            </div>
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

          {/* ════ DESCRIPTION & PROMPT ════ */}
          {(localContext.description || localContext.prompt) && (
            <section className="mb-8">
              <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
                Task
              </h3>
              <div className="border-t border-border/40">
                {localContext.description && (
                  <div className="flex items-start py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                    <span className="w-28 shrink-0 text-sm text-muted-foreground">Description</span>
                    <span className="text-sm flex-1">{localContext.description}</span>
                  </div>
                )}
                {localContext.prompt && (
                  <div className="flex items-start py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                    <span className="w-28 shrink-0 text-sm text-muted-foreground">Prompt</span>
                    <span className="text-sm flex-1 whitespace-pre-wrap">{localContext.prompt}</span>
                  </div>
                )}
              </div>
            </section>
          )}

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

