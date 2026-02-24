// web/src/pages/Pipelines.tsx — Pipeline listing page (inbox-style layout)
import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Loader2, ArrowLeft, Search,
  X, PanelRightClose, PanelRightOpen,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { fetchPipelines, fetchPipeline, updatePipeline, collectPipeline } from '@/entities/pipeline'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { PipelineSidebar } from '@/pages/pipelines/PipelineSidebar'
import { SessionListPanel } from '@/pages/pipelines/SessionListPanel'
import { SessionDetailPreview } from '@/pages/pipelines/session/SessionDetailPreview'
import { PipelineSettingsPanel } from '@/pages/pipelines/PipelineSettingsPanel'
import { useUIStore } from '@/entities/ui'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import type { PipelineSource, PipelineContext, PipelineWorkflow } from '@/shared/types'

// ─── New Session modal ──────────────────────────────────────────────────────

function NewSessionModal({
  isPending,
  onConfirm,
  onClose,
}: {
  isPending: boolean
  onConfirm: (config: { isTest: boolean; limit: number }) => void
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
          <p className="text-sm text-muted-foreground mb-4">Start a new session to collect and analyze sources.</p>
          <div className="space-y-4 bg-muted/20 p-4 rounded-xl border border-border/50">
            <label className="flex items-start gap-3 cursor-pointer group">
              <input type="checkbox" checked={isTest} onChange={(e) => setIsTest(e.target.checked)} className="mt-1 flex-shrink-0 cursor-pointer accent-primary" />
              <div>
                <span className="text-sm font-medium group-hover:text-foreground transition-colors">Test Run (Dry Run)</span>
                <p className="text-xs text-muted-foreground mt-0.5 leading-relaxed">Limit sources fetched to quickly test without consuming too many credits.</p>
              </div>
            </label>
            {isTest && (
              <div className="pl-7 pt-1 animate-in fade-in slide-in-from-top-2">
                <label className="text-xs font-medium text-muted-foreground mb-1.5 block">Item limit per source</label>
                <input type="number" min={1} max={50} value={limit} onChange={(e) => setLimit(Number(e.target.value))} className="w-24 rounded-lg bg-background border border-input px-3 py-1.5 text-sm focus:ring-1 focus:ring-ring outline-none" />
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-border bg-muted/10">
          <button onClick={onClose} className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer">Cancel</button>
          <button onClick={() => onConfirm({ isTest, limit })} disabled={isPending} className="inline-flex items-center gap-1.5 px-5 py-2 rounded-xl text-sm font-semibold bg-primary text-primary-foreground hover:bg-primary/90 transition-opacity disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed shadow-md shadow-primary/20">
            {isPending ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Starting…</> : 'Start Session'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function PipelinesPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()
  const addToast = useUIStore((s) => s.addToast)

  const selectedPipelineId = searchParams.get('p')
  const selectedSessionId = searchParams.get('s')

  const selectPipeline = useCallback((id: string) => {
    setSearchParams({ p: id })
  }, [setSearchParams])

  const selectSession = useCallback((id: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (id) {
        next.set('s', id)
      } else {
        next.delete('s')
      }
      return next
    })
  }, [setSearchParams])

  const [showNewSession, setShowNewSession] = useState(false)
  const [isSidebarOpen, setIsSidebarOpen] = useState(true)
  const { size: rightPanelWidth, handleMouseDown: onRightPanelDrag } = useResizeDrag({
    direction: 'horizontal',
    min: 260,
    max: 700,
    initial: 320,
  })

  // ─── Data fetching ─────────────────────────────────────────────────────

  const { data: pipelines = [], isLoading } = useQuery({
    queryKey: ['pipelines'],
    queryFn: fetchPipelines,
  })

  const { data: selectedPipeline } = useQuery({
    queryKey: ['pipeline', selectedPipelineId],
    queryFn: () => fetchPipeline(selectedPipelineId!),
    enabled: !!selectedPipelineId,
  })

  const { data: channels = [] } = useQuery({
    queryKey: ['publish-channels'],
    queryFn: fetchPublishChannels,
  })

  const collectMutation = useMutation({
    mutationFn: (config?: { isTest: boolean; limit: number }) =>
      collectPipeline(selectedPipelineId!, config),
    onSuccess: () => {
      setShowNewSession(false)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: selectedPipelineId }] })
    },
  })

  const updateContextMutation = useMutation({
    mutationFn: (ctx: PipelineContext) =>
      updatePipeline(selectedPipelineId!, { ...selectedPipeline!, context: ctx }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pipeline', selectedPipelineId] }),
  })

  // ─── Local settings state + auto-save ─────────────────────────────────

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')
  const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
  const [localModel, setLocalModel] = useState('')
  const [autoSaveStatus, setAutoSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle')

  // Sync server → local state when switching pipelines
  useEffect(() => {
    if (selectedPipeline) {
      setLocalSources(selectedPipeline.sources ?? []) // eslint-disable-line react-hooks/set-state-in-effect
      setLocalSchedule(selectedPipeline.schedule ?? '')
      setLocalWorkflows(selectedPipeline.workflows ?? [])
      setLocalModel(selectedPipeline.model ?? '')
    }
  }, [selectedPipeline?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Refs to read latest values in effects/cleanup without stale closures
  const pipelineRef = useRef(selectedPipeline)
  const localSourcesRef = useRef(localSources)
  const localScheduleRef = useRef(localSchedule)
  const localWorkflowsRef = useRef(localWorkflows)
  const localModelRef = useRef(localModel)

  useEffect(() => {
    pipelineRef.current = selectedPipeline
    localSourcesRef.current = localSources
    localScheduleRef.current = localSchedule
    localWorkflowsRef.current = localWorkflows
    localModelRef.current = localModel
  })

  const isDirty = useMemo(() => {
    if (!selectedPipeline) return false
    return (
      JSON.stringify(localSources) !== JSON.stringify(selectedPipeline.sources ?? []) ||
      localSchedule !== (selectedPipeline.schedule ?? '') ||
      JSON.stringify(localWorkflows) !== JSON.stringify(selectedPipeline.workflows ?? []) ||
      localModel !== (selectedPipeline.model ?? '')
    )
  }, [localSources, localSchedule, localWorkflows, localModel, selectedPipeline])

  const isDirtyRef = useRef(isDirty)
  useEffect(() => { isDirtyRef.current = isDirty })

  const doSave = async () => {
    const p = pipelineRef.current
    if (!p) return
    setAutoSaveStatus('saving')
    try {
      await updatePipeline(selectedPipelineId!, {
        ...p,
        sources: localSourcesRef.current,
        schedule: localScheduleRef.current,
        workflows: localWorkflowsRef.current,
        model: localModelRef.current,
      })
      queryClient.invalidateQueries({ queryKey: ['pipeline', selectedPipelineId] })
      setAutoSaveStatus('saved')
      setTimeout(() => setAutoSaveStatus('idle'), 2000)
    } catch (err) {
      setAutoSaveStatus('idle')
      addToast(`Failed to save pipeline settings: ${err instanceof Error ? err.message : 'unknown error'}`)
    }
  }

  const doSaveRef = useRef(doSave)
  useEffect(() => { doSaveRef.current = doSave })

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

  // Auto-select first pipeline on load
  useEffect(() => {
    if (!selectedPipelineId && pipelines.length > 0) {
      selectPipeline(pipelines[0].id)
    }
  }, [pipelines, selectedPipelineId, selectPipeline])

  // ─── Mobile level ──────────────────────────────────────────────────────

  type MobileLevel = 'pipelines' | 'sessions' | 'detail'
  const mobileLevel: MobileLevel = selectedSessionId
    ? 'detail'
    : selectedPipelineId
      ? 'sessions'
      : 'pipelines'

  const goBackToPipelines = () => setSearchParams({})
  const goBackToSessions = () => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.delete('s')
      return next
    })
  }

  const toggleSettings = useCallback(() => {
    setIsSidebarOpen(v => !v)
  }, [])

  const openSettings = useCallback(() => {
    setIsSidebarOpen(true)
  }, [])

  // ─── Render ────────────────────────────────────────────────────────────

  return (
    <MainLayout
      headerContent={<span className="font-semibold tracking-tight">Pipelines</span>}
    >
      <div className="flex h-full w-full overflow-hidden bg-background">

        {/* ── Level 1: Pipeline List Sidebar ── */}
        <div className={cn(
          'w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border',
          'bg-sidebar/30 backdrop-blur-xl z-20 flex flex-col',
          'md:shadow-[4px_0_24px_-12px_rgba(0,0,0,0.5)]',
          mobileLevel === 'pipelines' ? 'flex' : 'hidden md:flex',
        )}>
          <PipelineSidebar
            pipelines={pipelines}
            selectedId={selectedPipelineId}
            onSelect={selectPipeline}
            isLoading={isLoading}
            onSettingsOpen={openSettings}
          />
        </div>

        {/* ── Level 2+3: Sessions + Session Detail ── */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col relative',
          mobileLevel === 'pipelines' ? 'hidden md:flex' : 'flex',
        )}>
          {selectedPipelineId ? (
            <>
              {/* Pipeline header strip */}
              <div className="px-4 md:px-6 py-3 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex items-center justify-between gap-3">
                {/* Mobile back button */}
                <button
                  onClick={goBackToPipelines}
                  className="md:hidden flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors shrink-0"
                >
                  <ArrowLeft className="w-4 h-4" />
                </button>

                <div className="flex-1 min-w-0">
                  <h2 className="text-lg font-bold tracking-tight truncate">
                    {selectedPipeline?.name ?? 'Loading...'}
                  </h2>
                  {selectedPipeline?.description && (
                    <p className="text-xs text-muted-foreground truncate mt-0.5">{selectedPipeline.description}</p>
                  )}
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  <button
                    onClick={() => setShowNewSession(true)}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    <span className="hidden sm:inline">Start Session</span>
                  </button>
                  <button
                    onClick={toggleSettings}
                    className="hidden md:flex p-2 rounded-xl border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
                    title="Toggle Settings"
                  >
                    {isSidebarOpen ? <PanelRightClose className="w-4 h-4" /> : <PanelRightOpen className="w-4 h-4" />}
                  </button>
                </div>
              </div>

              {/* Inner split: Session list + Session detail + Settings panel */}
              <div className="flex flex-1 overflow-hidden">
                {/* Session list */}
                <SessionListPanel
                  pipelineId={selectedPipelineId}
                  selectedSessionId={selectedSessionId}
                  onSelectSession={selectSession}
                  className={cn(
                    'w-full md:w-[300px] 2xl:w-[340px] shrink-0 md:border-r border-border bg-sidebar/30',
                    mobileLevel === 'sessions' ? 'flex' : 'hidden md:flex',
                  )}
                />

                {/* Session detail */}
                <div className={cn(
                  'flex-1 min-w-0 flex flex-col bg-grid-pattern',
                  mobileLevel === 'detail' ? 'flex' : 'hidden md:flex',
                )}>
                  {/* Mobile back to sessions */}
                  {selectedSessionId && (
                    <button
                      onClick={goBackToSessions}
                      className="md:hidden flex items-center gap-2 px-4 py-3 border-b border-border text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                    >
                      <ArrowLeft className="w-4 h-4" />
                      Back to sessions
                    </button>
                  )}
                  {selectedSessionId ? (
                    <SessionDetailPreview pipelineId={selectedPipelineId} sessionId={selectedSessionId} />
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

                {/* Right Settings Panel — inside content area */}
                {isSidebarOpen && (
                  <div className="hidden md:contents">
                    <div
                      onMouseDown={onRightPanelDrag}
                      className="w-1 shrink-0 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-30 relative
                        after:absolute after:inset-y-0 after:-left-1 after:-right-1"
                    />
                    <aside
                      style={{ width: rightPanelWidth }}
                      className="border-l border-border bg-sidebar/95 backdrop-blur-md shadow-2xl z-30 flex flex-col shrink-0"
                    >
                      <PipelineSettingsPanel
                        pipelineId={selectedPipelineId}
                        sources={localSources}
                        schedule={localSchedule}
                        context={selectedPipeline?.context}
                        workflows={localWorkflows}
                        model={localModel}
                        channels={channels}
                        onSourcesChange={setLocalSources}
                        onScheduleChange={setLocalSchedule}
                        onContextSave={async (ctx) => { await updateContextMutation.mutateAsync(ctx) }}
                        onWorkflowsChange={setLocalWorkflows}
                        onModelChange={setLocalModel}
                        autoSaveStatus={autoSaveStatus}
                      />
                    </aside>
                  </div>
                )}
              </div>
            </>
          ) : (
            // No pipeline selected
            <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
              <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                <Search className="w-6 h-6 opacity-30" />
              </div>
              <div className="text-center">
                <p className="font-medium text-foreground">Select a pipeline</p>
                <p className="text-sm">Choose a pipeline from the list to view its sessions.</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* New Session Modal */}
      {showNewSession && selectedPipelineId && (
        <NewSessionModal
          isPending={collectMutation.isPending}
          onConfirm={(config) => collectMutation.mutate(config)}
          onClose={() => setShowNewSession(false)}
        />
      )}
    </MainLayout>
  )
}
