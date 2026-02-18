import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { useWorkflowStore } from '../../../stores/workflowStore'
import type { NodeData } from '../../../stores/workflowStore'

const colorMap: Record<string, string> = {
  input: 'border-yellow-500 bg-yellow-500/10',
  agent: 'border-blue-500 bg-blue-500/10',
  tool: 'border-red-500 bg-red-500/10',
  output: 'border-green-500 bg-green-500/10',
}

const iconMap: Record<string, string> = {
  input: '\u{1F7E1}',
  agent: '\u{1F535}',
  tool: '\u{1F534}',
  output: '\u{1F7E2}',
}

export function UpalNode({ id, data }: NodeProps<Node<NodeData>>) {
  const selectedNodeId = useWorkflowStore((s) => s.selectedNodeId)
  const selectNode = useWorkflowStore((s) => s.selectNode)
  const isSelected = selectedNodeId === id

  const handleClick = () => {
    selectNode(isSelected ? null : id)
  }

  return (
    <div
      className={`rounded-lg border-2 px-4 py-3 min-w-[200px] cursor-pointer ${colorMap[data.nodeType] || 'border-zinc-600'} ${isSelected ? 'ring-2 ring-zinc-400/50' : ''}`}
      onClick={handleClick}
    >
      <Handle type="target" position={Position.Top} className="!bg-zinc-400" />
      <div className="flex items-center gap-2">
        <span>{iconMap[data.nodeType]}</span>
        <span className="font-medium text-zinc-100">{data.label}</span>
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-zinc-400" />
    </div>
  )
}
