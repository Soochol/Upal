import { useWorkflowStore } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { configureNode } from '../api'
import { useUpstreamNodes } from '../model/useUpstreamNodes'
import { ConfigureChat } from '@/shared/ui/ConfigureChat'
import type { ChatMessage } from '@/shared/ui/ConfigureChat'

type AIChatEditorProps = {
  nodeId: string
  data: NodeData
}

export function AIChatEditor({ nodeId, data }: AIChatEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)
  const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)
  const upstreamNodes = useUpstreamNodes(nodeId)

  const handleSubmit = async (params: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => {
    const response = await configureNode({
      node_type: data.nodeType,
      node_id: nodeId,
      current_config: data.config,
      label: data.label,
      description: data.description ?? '',
      message: params.message,
      model: params.model || undefined,
      thinking: params.thinking,
      history: params.history,
      upstream_nodes: upstreamNodes,
    })

    // Apply returned config updates
    if (response.config && Object.keys(response.config).length > 0) {
      updateNodeConfig(nodeId, response.config)
    }
    if (response.label) {
      updateNodeLabel(nodeId, response.label)
    }
    const desc = response.description || (response.config?.description as string)
    if (desc) {
      updateNodeDescription(nodeId, desc)
    }

    return { explanation: response.explanation }
  }

  return <ConfigureChat onSubmit={handleSubmit} />
}
