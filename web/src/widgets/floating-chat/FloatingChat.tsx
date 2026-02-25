import { useState, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { MessageSquare, X } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { fetchPipeline, updatePipeline } from '@/entities/pipeline'
import { PipelineChatEditor } from '@/features/configure-pipeline/ui/PipelineChatEditor'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/shared/types'

interface FloatingChatProps {
  pipelineId?: string
  sessionId?: string
}

export function FloatingChat({ pipelineId }: FloatingChatProps) {
  const [isOpen, setIsOpen] = useState(false)
  const queryClient = useQueryClient()

  const { data: pipeline } = useQuery({
    queryKey: ['pipeline', pipelineId],
    queryFn: () => fetchPipeline(pipelineId!),
    enabled: !!pipelineId && isOpen,
  })

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['pipeline', pipelineId] })
    queryClient.invalidateQueries({ queryKey: ['pipelines'] })
  }, [queryClient, pipelineId])

  const handleSourcesChange = useCallback(
    (sources: PipelineSource[]) => {
      if (!pipelineId || !pipeline) return
      updatePipeline(pipelineId, { ...pipeline, sources }).then(invalidate)
    },
    [pipelineId, pipeline, invalidate],
  )

  const handleScheduleChange = useCallback(
    (schedule: string) => {
      if (!pipelineId || !pipeline) return
      updatePipeline(pipelineId, { ...pipeline, schedule }).then(invalidate)
    },
    [pipelineId, pipeline, invalidate],
  )

  const handleWorkflowsChange = useCallback(
    (workflows: PipelineWorkflow[]) => {
      if (!pipelineId || !pipeline) return
      updatePipeline(pipelineId, { ...pipeline, workflows }).then(invalidate)
    },
    [pipelineId, pipeline, invalidate],
  )

  const handleModelChange = useCallback(
    (model: string) => {
      if (!pipelineId || !pipeline) return
      updatePipeline(pipelineId, { ...pipeline, model }).then(invalidate)
    },
    [pipelineId, pipeline, invalidate],
  )

  const handleContextSave = useCallback(
    async (ctx: PipelineContext) => {
      if (!pipelineId || !pipeline) return
      await updatePipeline(pipelineId, { ...pipeline, context: ctx })
      invalidate()
    },
    [pipelineId, pipeline, invalidate],
  )

  // Don't render if there's no pipeline selected
  if (!pipelineId) return null

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
          {pipeline ? (
            <PipelineChatEditor
              pipelineId={pipelineId}
              currentSources={pipeline.sources ?? []}
              currentSchedule={pipeline.schedule ?? ''}
              currentWorkflows={pipeline.workflows ?? []}
              currentModel={pipeline.model ?? ''}
              currentContext={pipeline.context}
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
