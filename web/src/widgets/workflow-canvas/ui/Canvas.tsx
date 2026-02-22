import { useCallback, useEffect, useRef, type DragEvent } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  SelectionMode,
  useOnSelectionChange,
  useReactFlow,
  type OnSelectionChangeParams,
  type IsValidConnection,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { useWorkflowStore } from '@/entities/workflow'
import { uploadFile } from '@/shared/api'
import { UpalNode } from './UpalNode'
import { GroupNode } from './GroupNode'
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
  exposeGetViewportCenter?: (fn: () => { x: number; y: number }) => void
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

export function Canvas({ onAddFirstNode, onDropNode, onPromptSubmit, isGenerating, exposeGetViewportCenter }: CanvasProps) {
  const { nodes, edges, onNodesChange, onEdgesChange, onConnect, addNode } =
    useWorkflowStore()
  const { screenToFlowPosition } = useReactFlow()

  // Expose viewport center calculator to parent (for click-to-add nodes)
  useEffect(() => {
    exposeGetViewportCenter?.(() => {
      const el = document.querySelector('.react-flow')
      if (!el) return { x: 250, y: 150 }
      const rect = el.getBoundingClientRect()
      return screenToFlowPosition({
        x: rect.left + rect.width / 2,
        y: rect.top + rect.height / 2,
      })
    })
  }, [exposeGetViewportCenter, screenToFlowPosition])

  const isEmpty = nodes.length === 0

  /** Prevent self-connections at the React Flow level */
  const isValidConnection: IsValidConnection = useCallback(
    (connection) => connection.source !== connection.target,
    [],
  )

  /** When a connection drag ends outside a handle, check if it landed on a node body */
  const onConnectEnd = useCallback(
    (event: MouseEvent | TouchEvent) => {
      // Get cursor position (mouse or touch)
      const clientX = 'changedTouches' in event
        ? event.changedTouches[0].clientX
        : (event as MouseEvent).clientX
      const clientY = 'changedTouches' in event
        ? event.changedTouches[0].clientY
        : (event as MouseEvent).clientY

      // Find the node element under the cursor via DOM hit-testing
      const targetEl = document
        .elementFromPoint(clientX, clientY)
        ?.closest('.react-flow__node')
      if (!targetEl) return

      const targetNodeId = targetEl.getAttribute('data-id')
      if (!targetNodeId) return

      const { nodes: currentNodes, edges: currentEdges, onConnect: connect } =
        useWorkflowStore.getState()

      // Find the source node: the one whose source handle started the drag.
      // React Flow adds a "connecting" class to the source node during drag.
      const sourceEl = document.querySelector('.react-flow__node.connecting')
      const sourceNodeId = sourceEl?.getAttribute('data-id')
      if (!sourceNodeId) return

      // Prevent self-connections
      if (targetNodeId === sourceNodeId) return

      // Skip group nodes (no handles)
      const targetNode = currentNodes.find((n) => n.id === targetNodeId)
      if (!targetNode || targetNode.type === 'groupNode') return

      // Skip if already connected
      const alreadyConnected = currentEdges.some(
        (e) => e.source === sourceNodeId && e.target === targetNodeId,
      )
      if (alreadyConnected) return

      connect({
        source: sourceNodeId,
        target: targetNodeId,
        sourceHandle: null,
        targetHandle: null,
      })
    },
    [],
  )

  const onDragOver = useCallback((e: DragEvent) => {
    if (e.dataTransfer.types.includes('Files')) {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'copy'
    } else if (e.dataTransfer.types.includes('application/upal-node-type')) {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'move'
    }
  }, [])

  const onDrop = useCallback(
    async (e: DragEvent) => {
      e.preventDefault()

      // File drop: upload each file and create an asset node
      if (e.dataTransfer.files.length > 0) {
        const files = Array.from(e.dataTransfer.files)
        const basePosition = screenToFlowPosition({ x: e.clientX, y: e.clientY })
        for (let i = 0; i < files.length; i++) {
          try {
            const info = await uploadFile(files[i])
            addNode('asset', { x: basePosition.x + i * 20, y: basePosition.y + i * 20 }, {
              file_id: info.id,
              filename: info.filename,
              content_type: info.content_type,
              preview_text: info.preview_text,
            })
          } catch (err) {
            console.error('Failed to upload file:', err)
          }
        }
        return
      }

      // Node-type drop from the palette
      const type = e.dataTransfer.getData('application/upal-node-type')
      if (!type) return
      const position = screenToFlowPosition({ x: e.clientX, y: e.clientY })
      onDropNode(type, position)
    },
    [onDropNode, screenToFlowPosition, addNode],
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
        onConnectEnd={onConnectEnd}
        isValidConnection={isValidConnection}
        connectionRadius={80}
        nodeTypes={nodeTypes}
        fitView
        className="bg-background"
        multiSelectionKeyCode={['Shift', 'Control', 'Meta']}
        selectionOnDrag
        selectionMode={SelectionMode.Partial}
        panOnDrag={[1, 2]}
        deleteKeyCode={['Delete', 'Backspace']}
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
