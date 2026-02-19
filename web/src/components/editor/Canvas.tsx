import { useCallback, type DragEvent } from 'react'
import { ReactFlow, Background, Controls, MiniMap } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useWorkflowStore } from '@/stores/workflowStore'
import { UpalNode } from './nodes/UpalNode'
import { EmptyState } from './EmptyState'

const nodeTypes = {
  upalNode: UpalNode,
}

type CanvasProps = {
  onAddFirstNode: () => void
  onGenerate: () => void
  onDropNode: (type: string, position: { x: number; y: number }) => void
}

export function Canvas({ onAddFirstNode, onGenerate, onDropNode }: CanvasProps) {
  const { nodes, edges, onNodesChange, onEdgesChange, onConnect } =
    useWorkflowStore()

  const isEmpty = nodes.length === 0

  const onDragOver = useCallback((e: DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }, [])

  const onDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      const type = e.dataTransfer.getData('application/upal-node-type')
      if (!type) return
      const bounds = e.currentTarget.getBoundingClientRect()
      const position = {
        x: e.clientX - bounds.left,
        y: e.clientY - bounds.top,
      }
      onDropNode(type, position)
    },
    [onDropNode],
  )

  return (
    <div className="h-full w-full relative bg-background" onDrop={onDrop} onDragOver={onDragOver}>
      {isEmpty && (
        <EmptyState onAddNode={onAddFirstNode} onGenerate={onGenerate} />
      )}
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
        className="bg-background"
      >
        <Background color="var(--border)" gap={20} size={1} />
        <Controls className="!bg-card !border-border !shadow-sm" />
        <MiniMap
          nodeColor="var(--muted)"
          maskColor="var(--background)"
          className="!bg-card !border-border !rounded-lg !shadow-sm"
        />
      </ReactFlow>
    </div>
  )
}
