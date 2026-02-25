import { ConfigureChat } from '@/shared/ui/ConfigureChat'
import type { ChatMessage } from '@/shared/ui/ConfigureChat'
import { configurePipeline } from '../api'
import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/entities/pipeline'

type PipelineChatEditorProps = {
  pipelineId: string
  currentSources: PipelineSource[]
  currentSchedule: string
  currentWorkflows: PipelineWorkflow[]
  currentModel: string
  currentContext?: PipelineContext
  onSourcesChange: (s: PipelineSource[]) => void
  onScheduleChange: (s: string) => void
  onWorkflowsChange: (w: PipelineWorkflow[]) => void
  onModelChange: (m: string) => void
  onContextSave: (ctx: PipelineContext) => Promise<void>
}

export function PipelineChatEditor({
  pipelineId,
  currentSources, currentSchedule, currentWorkflows, currentModel, currentContext,
  onSourcesChange, onScheduleChange, onWorkflowsChange, onModelChange, onContextSave,
}: PipelineChatEditorProps) {
  const handleSubmit = async ({ message, model, thinking, history }: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => {
    const response = await configurePipeline(pipelineId, {
      message,
      model: model || undefined,
      thinking,
      history,
      current_sources: currentSources,
      current_schedule: currentSchedule,
      current_workflows: currentWorkflows,
      current_model: currentModel,
      current_context: currentContext,
    })

    if (response.sources) onSourcesChange(response.sources)
    if (response.schedule !== undefined && response.schedule !== null) onScheduleChange(response.schedule)
    if (response.workflows) onWorkflowsChange(response.workflows)
    if (response.model !== undefined && response.model !== null) onModelChange(response.model)
    if (response.context) await onContextSave(response.context)

    return { explanation: response.explanation || 'Settings updated.' }
  }

  return (
    <ConfigureChat
      onSubmit={handleSubmit}
      placeholder="Describe your pipeline settings..."
      loadingText="Configuring..."
    />
  )
}
