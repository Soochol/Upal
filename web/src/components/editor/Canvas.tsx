import { ReactFlow, Background, Controls, MiniMap } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useWorkflowStore } from '../../stores/workflowStore'
import { UpalNode } from './nodes/UpalNode'

const nodeTypes = {
  upalNode: UpalNode,
}

export function Canvas() {
  const { nodes, edges, onNodesChange, onEdgesChange, onConnect } =
    useWorkflowStore()

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        nodeTypes={nodeTypes}
        fitView
      >
        <Background color="#333" gap={20} />
        <Controls />
        <MiniMap />
      </ReactFlow>
    </div>
  )
}
