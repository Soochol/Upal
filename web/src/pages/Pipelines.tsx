// web/src/pages/Pipelines.tsx — Pipeline management page (3-column layout)
import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2, X, Settings, ArrowLeft } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { fetchPipelines, fetchPipeline, collectPipeline } from '@/entities/pipeline'
import { createDraftSession } from '@/entities/content-session/api'
import { useUIStore } from '@/entities/ui'
import { PipelineSidebar } from '@/pages/pipelines/PipelineSidebar'
import { SessionListPanel } from '@/pages/pipelines/SessionListPanel'
import { SessionSetupView } from '@/pages/pipelines/session/SessionSetupView'

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
    setSearchParams({ p: id }) // clears ?s= when switching pipeline
  }, [setSearchParams])

  const selectSession = useCallback((id: string) => {
    if (!selectedPipelineId) return
    if (id) {
      setSearchParams({ p: selectedPipelineId, s: id })
    } else {
      setSearchParams({ p: selectedPipelineId })
    }
  }, [selectedPipelineId, setSearchParams])

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

  const collectMutation = useMutation({
    mutationFn: (config?: { isTest: boolean; limit: number }) =>
      collectPipeline(selectedPipelineId!, config),
    onSuccess: (newSession) => {
      setShowNewSession(false)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: selectedPipelineId }] })
      if (newSession?.session_id && selectedPipelineId) {
        setSearchParams({ p: selectedPipelineId, s: newSession.session_id })
      }
    },
    onError: (err) => addToast(`Failed to start session: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  const newSessionMutation = useMutation({
    mutationFn: () => createDraftSession({ pipeline_id: selectedPipelineId! }),
    onSuccess: (session) => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: selectedPipelineId }] })
      if (selectedPipelineId) {
        setSearchParams({ p: selectedPipelineId, s: session.id })
      }
    },
    onError: (err) => addToast(`Failed to create session: ${err instanceof Error ? err.message : 'unknown error'}`),
  })

  // Auto-select first pipeline on load
  useEffect(() => {
    if (!selectedPipelineId && pipelines.length > 0) {
      selectPipeline(pipelines[0].id)
    }
  }, [pipelines, selectedPipelineId, selectPipeline])

  // ─── Mobile level ──────────────────────────────────────────────────────

  type MobileLevel = 'pipelines' | 'sessions' | 'detail'
  const mobileLevel: MobileLevel =
    selectedSessionId ? 'detail' :
    selectedPipelineId ? 'sessions' : 'pipelines'

  const goBackToPipelines = () => setSearchParams({})
  const goBackToSessions = () => {
    if (selectedPipelineId) setSearchParams({ p: selectedPipelineId })
  }

  // ─── Render ────────────────────────────────────────────────────────────

  return (
    <MainLayout
      headerContent={<span className="font-semibold tracking-tight">Pipelines</span>}
    >
      <div className="flex h-full w-full overflow-hidden bg-background">

        {/* ── Column 1: Pipeline List ── */}
        <div className={cn(
          'w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border',
          'bg-sidebar/30 backdrop-blur-xl flex flex-col',
          mobileLevel === 'pipelines' ? 'flex' : 'hidden md:flex',
        )}>
          <PipelineSidebar
            pipelines={pipelines}
            selectedId={selectedPipelineId}
            onSelect={selectPipeline}
            onDeselect={() => setSearchParams({})}
            isLoading={isLoading}
          />
        </div>

        {/* ── Column 2: Session List ── */}
        <div className={cn(
          'w-full md:w-[340px] 2xl:w-[400px] shrink-0 md:border-r border-border',
          'bg-sidebar/30 backdrop-blur-xl flex flex-col',
          mobileLevel === 'sessions' ? 'flex' : 'hidden md:flex',
        )}>
          {selectedPipelineId ? (
            <SessionListPanel
              pipelineId={selectedPipelineId}
              pipelineName={selectedPipeline?.name}
              isContentPipeline={(selectedPipeline?.stages ?? []).some(s => s.type === 'collect')}
              selectedSessionId={selectedSessionId}
              onSelectSession={selectSession}
              onNewSession={() => newSessionMutation.mutate()}
              onStartSession={() => setShowNewSession(true)}
              onBack={goBackToPipelines}
              className="h-full"
            />
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground p-4">
              <p className="text-sm text-center">Select a pipeline</p>
            </div>
          )}
        </div>

        {/* ── Column 3: Session Settings ── */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col',
          mobileLevel === 'detail' ? 'flex' : 'hidden md:flex',
        )}>
          {selectedSessionId && selectedPipelineId ? (
            <>
              {/* Mobile back button */}
              <div className="md:hidden px-4 py-2 border-b border-border/50 shrink-0">
                <button
                  onClick={goBackToSessions}
                  className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
                >
                  <ArrowLeft className="w-4 h-4" />
                  Back
                </button>
              </div>
              <SessionSetupView
                sessionId={selectedSessionId}
                pipelineId={selectedPipelineId}
              />
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground flex-col gap-3">
              <div className="size-14 rounded-full bg-muted/30 flex items-center justify-center shrink-0 border border-border/50">
                <Settings className="w-6 h-6 opacity-30" />
              </div>
              <div className="text-center">
                <p className="font-medium text-foreground">Select a session</p>
                <p className="text-sm">Choose a session to view and edit its settings.</p>
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
