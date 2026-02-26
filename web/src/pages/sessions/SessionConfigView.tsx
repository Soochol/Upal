import { useState, useEffect, useMemo } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Trash2, Loader2, Play, Pencil, ChevronDown, ChevronRight, GitBranch, Power,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { AddSourceModal, STATIC_SOURCES, SIGNAL_SOURCES } from '@/features/configure-pipeline-sources/AddSourceModal'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import { WorkflowPicker } from '@/features/workflow-picker'
import {
  fetchSession,
  updateSession,
  activateSession,
  deactivateSession,
} from '@/entities/session'
import { createRun } from '@/entities/session-run'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { useUIStore } from '@/entities/ui'
import { SCHEDULE_PRESETS } from '@/entities/session'
import type { Session, SessionSource, SessionWorkflow, SessionContext } from '@/entities/session'

const DEFAULT_CONTEXT: SessionContext = {
  prompt: '', language: 'Korean',
}

type Props = {
  sessionId: string
}

export function SessionConfigView({ sessionId }: Props) {
  const queryClient = useQueryClient()
  const addToast = useUIStore((s) => s.addToast)

  // ─── Data fetching ────────────────────────────────────────────────────

  const { data: session } = useQuery<Session>({
    queryKey: ['session', sessionId],
    queryFn: () => fetchSession(sessionId),
    enabled: !!sessionId,
  })

  const { data: channels = [] } = useQuery({
    queryKey: ['publish-channels'],
    queryFn: fetchPublishChannels,
  })

  // ─── Local settings state ─────────────────────────────────────────────

  const [localSources, setLocalSources] = useState<SessionSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')
  const [localWorkflows, setLocalWorkflows] = useState<SessionWorkflow[]>([])
  const [localModel, setLocalModel] = useState('')
  const [localContext, setLocalContext] = useState<SessionContext>(DEFAULT_CONTEXT)
  const [showAddModal, setShowAddModal] = useState(false)
  const [editingSourceIndex, setEditingSourceIndex] = useState<number | null>(null)
  const [showWorkflowPicker, setShowWorkflowPicker] = useState(false)
  const [showAdvancedSources, setShowAdvancedSources] = useState(false)
  const [isEditingName, setIsEditingName] = useState(false)
  const [localName, setLocalName] = useState('')

  // Effective values: session-level values only
  const effectiveSources = session?.sources ?? []
  const effectiveSchedule = session?.schedule ?? ''
  const effectiveWorkflows = session?.workflows ?? []
  const effectiveModel = session?.model ?? ''
  const effectiveContext = session?.context ?? DEFAULT_CONTEXT

  // Fingerprint of server-side data — changes when refetch brings new values
  const serverFingerprint = JSON.stringify([
    session?.sources, session?.schedule,
    session?.workflows, session?.model, session?.context,
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
      await updateSession(sessionId, data)
      queryClient.invalidateQueries({ queryKey: ['session', sessionId] })
    },
    delay: 2000,
    saveOnUnmount: true,
    onBeforeUnloadSave: (data) => {
      fetch(`/api/sessions/${encodeURIComponent(sessionId)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        keepalive: true,
        body: JSON.stringify(data),
      })
    },
    onError: (err) => addToast(`Failed to save: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  // Sync server -> local state when server data changes
  // (covers both session switch AND external updates)
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
      queryClient.invalidateQueries({ queryKey: ['session', sessionId] })
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
    },
    onError: (err) => addToast(`Failed to toggle session: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  const runMutation = useMutation({
    mutationFn: () => createRun(sessionId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
      queryClient.invalidateQueries({ queryKey: ['session-runs', sessionId] })
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
      await updateSession(sessionId, { name: trimmed })
      queryClient.invalidateQueries({ queryKey: ['session', sessionId] })
      queryClient.invalidateQueries({ queryKey: ['sessions'] })
    } catch (err) {
      addToast(`Failed to rename: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }

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
              {session.name || 'Untitled Session'}
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

          {/* RESEARCH */}
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
                            {(() => {
                              const def = [...STATIC_SOURCES, ...SIGNAL_SOURCES].find(s => s.type === src.type)
                              return def ? <span className={`shrink-0 ${def.accentText}`}>{def.icon}</span> : null
                            })()}
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

          {/* TASK */}
          <section className="mb-8">
            <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Task
            </h3>
            <div className="border-t border-border/40">
              <div className="flex items-start py-2.5 -mx-2 px-2 rounded-md hover:bg-muted/30 transition-colors">
                <span className="w-28 shrink-0 text-sm text-muted-foreground pt-0.5">Prompt</span>
                <textarea
                  value={localContext.prompt || ''}
                  onChange={(e) => setLocalContext({ ...localContext, prompt: e.target.value })}
                  placeholder="Tell the AI what to do..."
                  rows={4}
                  className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground/30 resize-none"
                />
              </div>
            </div>
          </section>

          {/* PROCESSING */}
          <section className="mb-8">
            <h3 className="text-xs font-semibold uppercase tracking-widest text-muted-foreground mb-2">
              Processing
            </h3>
            <div className="border-t border-border/40">
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

          {/* SCHEDULE */}
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
