// web/src/pages/Pipelines.tsx — Pipeline settings page
import { useState, useEffect, useCallback, useMemo } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus, Loader2, ArrowLeft, Settings,
  X,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { fetchPipelines, fetchPipeline, collectPipeline } from '@/entities/pipeline'
import { fetchContentSessions, updateSessionSettings } from '@/entities/content-session/api'
import { fetchPublishChannels } from '@/entities/publish-channel/api'
import { PipelineSidebar } from '@/pages/pipelines/PipelineSidebar'
import { PipelineSettingsPanel } from '@/pages/pipelines/PipelineSettingsPanel'
import { useUIStore } from '@/entities/ui'
import { useAutoSave } from '@/shared/hooks/useAutoSave'
import type { PipelineSource, PipelineWorkflow } from '@/entities/pipeline'

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
  const navigate = useNavigate()
  const addToast = useUIStore((s) => s.addToast)

  const selectedPipelineId = searchParams.get('p')

  const selectPipeline = useCallback((id: string) => {
    setSearchParams({ p: id })
  }, [setSearchParams])

  const [showNewSession, setShowNewSession] = useState(false)

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

  // Fetch template session for the selected pipeline (holds settings)
  const { data: templateSessions = [] } = useQuery({
    queryKey: ['content-sessions', { pipelineId: selectedPipelineId, templateOnly: true }],
    queryFn: () => fetchContentSessions({ pipelineId: selectedPipelineId!, templateOnly: true }),
    enabled: !!selectedPipelineId,
  })
  const templateSession = templateSessions[0] ?? null

  const collectMutation = useMutation({
    mutationFn: (config?: { isTest: boolean; limit: number }) =>
      collectPipeline(selectedPipelineId!, config),
    onSuccess: (newSession) => {
      setShowNewSession(false)
      navigate(newSession?.session_id ? `/inbox?s=${newSession.session_id}` : '/inbox')
    },
  })

  // ─── Local settings state + auto-save (reads from template session) ───

  const [localSources, setLocalSources] = useState<PipelineSource[]>([])
  const [localSchedule, setLocalSchedule] = useState('')
  const [localWorkflows, setLocalWorkflows] = useState<PipelineWorkflow[]>([])
  const [localModel, setLocalModel] = useState('')

  // Bundle settings into one object for auto-save
  const settingsData = useMemo(() => ({
    sources: localSources,
    schedule: localSchedule,
    workflows: localWorkflows,
    model: localModel,
  }), [localSources, localSchedule, localWorkflows, localModel])

  const { saveStatus: autoSaveStatus, markClean } = useAutoSave({
    data: settingsData,
    onSave: async (data) => {
      if (!templateSession) return
      await updateSessionSettings(templateSession.id, data)
      queryClient.invalidateQueries({
        queryKey: ['content-sessions', { pipelineId: selectedPipelineId, templateOnly: true }],
      })
    },
    delay: 2000,
    enabled: !!templateSession,
    saveOnUnmount: true,
    onBeforeUnloadSave: (data) => {
      if (!templateSession) return
      fetch(`/api/content-sessions/${templateSession.id}/settings`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        keepalive: true,
        body: JSON.stringify(data),
      })
    },
    onError: (err) =>
      addToast(`Failed to save session settings: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  // Sync server → local state when template session changes
  useEffect(() => {
    if (templateSession) {
      setLocalSources(templateSession.session_sources ?? []) // eslint-disable-line react-hooks/set-state-in-effect
      setLocalSchedule(templateSession.schedule ?? '')
      setLocalWorkflows(templateSession.session_workflows ?? [])
      setLocalModel(templateSession.model ?? '')
      markClean()
    }
  }, [templateSession?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-select first pipeline on load
  useEffect(() => {
    if (!selectedPipelineId && pipelines.length > 0) {
      selectPipeline(pipelines[0].id)
    }
  }, [pipelines, selectedPipelineId, selectPipeline])

  // ─── Mobile level ──────────────────────────────────────────────────────

  type MobileLevel = 'pipelines' | 'detail'
  const mobileLevel: MobileLevel = selectedPipelineId ? 'detail' : 'pipelines'

  const goBackToPipelines = () => setSearchParams({})

  // ─── Render ────────────────────────────────────────────────────────────

  return (
    <MainLayout
      headerContent={<span className="font-semibold tracking-tight">Pipelines</span>}
    >
      <div className="flex h-full w-full overflow-hidden bg-background">

        {/* ── Pipeline List Sidebar ── */}
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
          />
        </div>

        {/* ── Main content: pipeline header + settings ── */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col',
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
                </div>
              </div>

              {/* Settings panel as main content */}
              <div className="flex-1 overflow-hidden">
                <PipelineSettingsPanel
                  pipelineId={selectedPipelineId}
                  sources={localSources}
                  schedule={localSchedule}
                  context={templateSession?.context}
                  workflows={localWorkflows}
                  model={localModel}
                  channels={channels}
                  onSourcesChange={setLocalSources}
                  onScheduleChange={setLocalSchedule}
                  onContextSave={async (ctx) => {
                    if (!templateSession) return
                    await updateSessionSettings(templateSession.id, { context: ctx })
                    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: selectedPipelineId, templateOnly: true }] })
                  }}
                  onWorkflowsChange={setLocalWorkflows}
                  onModelChange={setLocalModel}
                  autoSaveStatus={autoSaveStatus}
                />
              </div>
            </>
          ) : (
            // No pipeline selected
            <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
              <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                <Settings className="w-6 h-6 opacity-30" />
              </div>
              <div className="text-center">
                <p className="font-medium text-foreground">Select a pipeline</p>
                <p className="text-sm">Choose a pipeline from the list to configure its settings.</p>
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
