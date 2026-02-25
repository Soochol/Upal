// web/src/pages/Pipelines.tsx — Pipeline listing page (inbox-style layout)
import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Loader2, ArrowLeft, Search, X,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { MainLayout } from '@/app/layout'
import { fetchPipelines, fetchPipeline } from '@/entities/pipeline'
import { createDraftSession } from '@/entities/content-session/api'
import { PipelineSidebar } from '@/pages/pipelines/PipelineSidebar'
import { SessionListPanel } from '@/pages/pipelines/SessionListPanel'
import { SessionSetupView } from '@/pages/pipelines/session/SessionSetupView'
import { FloatingChat } from '@/widgets/floating-chat/FloatingChat'

// ─── New Session modal ──────────────────────────────────────────────────────

function NewSessionModal({
  isPending,
  onConfirm,
  onClose,
}: {
  isPending: boolean
  onConfirm: (name: string) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-2xl shadow-xl w-full max-w-sm mx-4 overflow-hidden">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <h2 className="text-sm font-semibold">New Session</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="p-5 space-y-3">
          <label className="block">
            <span className="text-xs font-medium text-muted-foreground">Session Name</span>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Weekly AI Trends"
              autoFocus
              onKeyDown={(e) => { if (e.key === 'Enter' && name.trim()) onConfirm(name.trim()) }}
              className="mt-1.5 w-full h-10 px-3 rounded-xl border border-input bg-background text-sm outline-none focus:ring-1 focus:ring-ring transition-shadow placeholder:text-muted-foreground/50"
            />
          </label>
          <p className="text-xs text-muted-foreground">You can configure sources, schedule, and workflows after creation.</p>
        </div>
        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-border bg-muted/10">
          <button onClick={onClose} className="px-4 py-2 rounded-xl text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer">Cancel</button>
          <button onClick={() => onConfirm(name.trim())} disabled={isPending || !name.trim()} className="inline-flex items-center gap-1.5 px-5 py-2 rounded-xl text-sm font-semibold bg-primary text-primary-foreground hover:bg-primary/90 transition-opacity disabled:opacity-60 cursor-pointer disabled:cursor-not-allowed shadow-md shadow-primary/20">
            {isPending ? <><Loader2 className="h-3.5 w-3.5 animate-spin" />Creating...</> : 'Create Session'}
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

  const createSessionMutation = useMutation({
    mutationFn: (name: string) => createDraftSession(selectedPipelineId!, name),
    onSuccess: (session) => {
      setShowNewSession(false)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId: selectedPipelineId }] })
      selectSession(session.id)
    },
  })

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
            onDelete={(id) => {
              if (selectedPipelineId === id) setSearchParams({})
            }}
          />
        </div>

        {/* ── Level 2+3: Sessions + Session Detail ── */}
        <div className={cn(
          'flex-1 min-w-0 flex flex-col relative',
          mobileLevel === 'pipelines' ? 'hidden md:flex' : 'flex',
        )}>
          {selectedPipelineId ? (
            <>
              {/* Pipeline header strip (mobile only — on desktop, selected pipeline is visible in sidebar) */}
              <div className="md:hidden px-4 py-3 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex items-center gap-3">
                <button
                  onClick={goBackToPipelines}
                  className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors shrink-0"
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
              </div>

              {/* Inner split: Session list + Session detail */}
              <div className="flex flex-1 overflow-hidden">
                {/* Session list */}
                <SessionListPanel
                  pipelineId={selectedPipelineId}
                  selectedSessionId={selectedSessionId}
                  onSelectSession={selectSession}
                  onNewSession={() => setShowNewSession(true)}
                  onDeleteSession={(id) => {
                    if (selectedSessionId === id) {
                      setSearchParams((prev) => {
                        const next = new URLSearchParams(prev)
                        next.delete('s')
                        return next
                      })
                    }
                  }}
                  className={cn(
                    'w-full md:w-[300px] 2xl:w-[340px] shrink-0 md:border-r border-border bg-sidebar/30',
                    mobileLevel === 'sessions' ? 'flex' : 'hidden md:flex',
                  )}
                />

                {/* Session detail / setup */}
                <div className={cn(
                  'flex-1 min-w-0 flex flex-col',
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
                    <SessionSetupView key={selectedSessionId} pipelineId={selectedPipelineId} sessionId={selectedSessionId} />
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
          isPending={createSessionMutation.isPending}
          onConfirm={(name) => createSessionMutation.mutate(name)}
          onClose={() => setShowNewSession(false)}
        />
      )}

      {/* Floating AI Assistant */}
      <FloatingChat
        pipelineId={selectedPipelineId ?? undefined}
        sessionId={selectedSessionId ?? undefined}
      />
    </MainLayout>
  )
}
