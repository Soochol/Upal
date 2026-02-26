// web/src/pages/Pipelines.tsx — Pipeline management page (3-column layout)
import { useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Settings, ArrowLeft } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { fetchPipelines, fetchPipeline } from '@/entities/pipeline'
import { createDraftSession } from '@/entities/content-session/api'
import { useUIStore } from '@/entities/ui'
import { PipelineSidebar } from '@/pages/pipelines/PipelineSidebar'
import { SessionListPanel } from '@/pages/pipelines/SessionListPanel'
import { SessionSetupView } from '@/pages/pipelines/session/SessionSetupView'

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

  const newSessionMutation = useMutation({
    mutationFn: () => createDraftSession({ pipeline_id: selectedPipelineId!, is_template: true }),
    onSuccess: (session) => {
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: selectedPipelineId, templateOnly: true }] })
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

    </MainLayout>
  )
}
