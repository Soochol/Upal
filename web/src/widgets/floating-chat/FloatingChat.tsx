import { useState, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { MessageSquare, X } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { fetchContentSession, updateSessionSettings } from '@/entities/content-session/api'
import { PipelineChatEditor } from '@/features/configure-pipeline/ui/PipelineChatEditor'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/shared/types'

interface FloatingChatProps {
  pipelineId?: string
  sessionId?: string
}

export function FloatingChat({ pipelineId, sessionId }: FloatingChatProps) {
  const [isOpen, setIsOpen] = useState(false)
  const queryClient = useQueryClient()

  const { data: session } = useQuery({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId!),
    enabled: !!sessionId && isOpen,
  })

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
  }, [queryClient, sessionId, pipelineId])

  const handleSourcesChange = useCallback(
    (sources: PipelineSource[]) => {
      if (!sessionId) return
      updateSessionSettings(sessionId, { sources }).then(invalidate)
    },
    [sessionId, invalidate],
  )

  const handleScheduleChange = useCallback(
    (schedule: string) => {
      if (!sessionId) return
      updateSessionSettings(sessionId, { schedule }).then(invalidate)
    },
    [sessionId, invalidate],
  )

  const handleWorkflowsChange = useCallback(
    (workflows: PipelineWorkflow[]) => {
      if (!sessionId) return
      updateSessionSettings(sessionId, { workflows }).then(invalidate)
    },
    [sessionId, invalidate],
  )

  const handleModelChange = useCallback(
    (model: string) => {
      if (!sessionId) return
      updateSessionSettings(sessionId, { model }).then(invalidate)
    },
    [sessionId, invalidate],
  )

  const handleContextSave = useCallback(
    async (ctx: PipelineContext) => {
      if (!sessionId) return
      await updateSessionSettings(sessionId, { context: ctx })
      invalidate()
    },
    [sessionId, invalidate],
  )

  // Don't render if there's no session selected
  if (!sessionId) return null

  return (
    <>
      {/* Expanded panel */}
      <div
        className={cn(
          'fixed bottom-20 right-6 z-50 w-[400px] h-[500px]',
          'bg-card border border-border rounded-2xl shadow-2xl',
          'flex flex-col overflow-hidden',
          'transition-all duration-200 ease-out origin-bottom-right',
          isOpen
            ? 'scale-100 opacity-100 pointer-events-auto'
            : 'scale-95 opacity-0 pointer-events-none',
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-background/80 backdrop-blur-sm shrink-0">
          <span className="text-sm font-semibold">AI Assistant</span>
          <button
            onClick={() => setIsOpen(false)}
            className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Chat body */}
        <div className="flex-1 overflow-y-auto">
          {session ? (
            <PipelineChatEditor
              pipelineId={pipelineId ?? session.pipeline_id}
              currentSources={session.session_sources ?? []}
              currentSchedule={session.schedule ?? ''}
              currentWorkflows={session.session_workflows ?? []}
              currentModel={session.model ?? ''}
              currentContext={session.context}
              onSourcesChange={handleSourcesChange}
              onScheduleChange={handleScheduleChange}
              onWorkflowsChange={handleWorkflowsChange}
              onModelChange={handleModelChange}
              onContextSave={handleContextSave}
            />
          ) : (
            <div className="flex items-center justify-center h-full text-sm text-muted-foreground">
              Loading...
            </div>
          )}
        </div>
      </div>

      {/* Floating toggle button */}
      <button
        onClick={() => setIsOpen((v) => !v)}
        className={cn(
          'fixed bottom-6 right-6 z-50',
          'h-12 w-12 rounded-full',
          'bg-primary text-primary-foreground',
          'shadow-lg hover:shadow-xl',
          'flex items-center justify-center',
          'transition-all duration-200 cursor-pointer',
          'hover:scale-105 active:scale-95',
        )}
      >
        {isOpen ? (
          <X className="h-5 w-5" />
        ) : (
          <MessageSquare className="h-5 w-5" />
        )}
      </button>
    </>
  )
}
