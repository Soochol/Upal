import { useCallback, useEffect, useRef, type DragEvent } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useOnSelectionChange,
  type OnSelectionChangeParams,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useWorkflowStore } from '@/stores/workflowStore'
import { UpalNode } from './nodes/UpalNode'
import { GroupNode } from './nodes/GroupNode'
import { EmptyState } from './EmptyState'
import { CanvasPromptBar } from './CanvasPromptBar'

const nodeTypes = {
  upalNode: UpalNode,
  groupNode: GroupNode,
}

type CanvasProps = {
  onAddFirstNode: () => void
  onDropNode: (type: string, position: { x: number; y: number }) => void
  onPromptSubmit: (description: string) => void
  isGenerating: boolean
}

/** Inner component that uses React Flow hooks (must be inside ReactFlow). */
function SelectionGrouper() {
  const createGroup = useWorkflowStore((s) => s.createGroup)
  const selectedNodeIdsRef = useRef<string[]>([])

  const onSelectionChange = useCallback(
    ({ nodes: selectedNodes }: OnSelectionChangeParams) => {
      selectedNodeIdsRef.current = selectedNodes
        .filter((n) => n.type !== 'groupNode')
        .map((n) => n.id)
    },
    [],
  )

  useOnSelectionChange({ onChange: onSelectionChange })

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'g') {
        e.preventDefault()
        const ids = selectedNodeIdsRef.current
        if (ids.length >= 2) {
          createGroup(ids)
        }
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [createGroup])

  return null
}

export function Canvas({ onAddFirstNode, onDropNode, onPromptSubmit, isGenerating }: CanvasProps) {
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
        <EmptyState onAddNode={onAddFirstNode} />
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
        multiSelectionKeyCode={['Shift', 'Control', 'Meta']}
        selectionOnDrag
        proOptions={{ hideAttribution: true }}
      >
        <SelectionGrouper />
        <Background color="var(--border)" gap={20} size={1} />
        <Controls className="!bg-card !border-border !shadow-sm" />
        {!isEmpty && (
          <MiniMap
            nodeColor="var(--muted)"
            maskColor="var(--background)"
            className="!bg-card !border-border !rounded-lg !shadow-sm"
          />
        )}
      </ReactFlow>
      <CanvasPromptBar
        onSubmit={onPromptSubmit}
        isGenerating={isGenerating}
        hasNodes={!isEmpty}
      />
    </div>
  )
}
