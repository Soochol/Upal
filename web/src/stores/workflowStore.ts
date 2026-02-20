import { create } from 'zustand'
import {
  type Node,
  type Edge,
  addEdge,
  applyNodeChanges,
  applyEdgeChanges,
  type OnNodesChange,
  type OnEdgesChange,
  type OnConnect,
} from '@xyflow/react'
import { getLayoutedElements } from '@/lib/layout'
import { NODE_TYPES, type NodeType } from '@/lib/nodeTypes'
import { useUIStore } from './uiStore'

// Re-export types and stores for backward compatibility
export type { RunEvent, NodeRunStatus } from './executionStore'
export { useExecutionStore } from './executionStore'
export { useUIStore } from './uiStore'

export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'output' | 'group'
  description: string
  config: Record<string, unknown>
}

type WorkflowState = {
  nodes: Node<NodeData>[]
  edges: Edge[]
  onNodesChange: OnNodesChange<Node<NodeData>>
  onEdgesChange: OnEdgesChange
  onConnect: OnConnect
  addNode: (type: NodeData['nodeType'], position: { x: number; y: number }) => void
  updateNodeConfig: (nodeId: string, config: Record<string, unknown>) => void
  updateNodeLabel: (nodeId: string, label: string) => void
  updateNodeDescription: (nodeId: string, description: string) => void
  applyAutoLayout: () => void

  // Workflow identity
  workflowName: string
  originalName: string // name at load/save time — empty for unsaved workflows
  setWorkflowName: (name: string) => void
  setOriginalName: (name: string) => void

  // Group management
  createGroup: (nodeIds: string[]) => void
  removeGroup: (groupId: string) => void
  updateGroupLabel: (groupId: string, label: string) => void
  updateGroupColor: (groupId: string, color: string) => void
}

let nodeId = 0
const getId = () => `node_${++nodeId}`

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  nodes: [],
  edges: [],

  // Workflow identity
  workflowName: '',
  originalName: '',

  onNodesChange: (changes) => {
    set({ nodes: applyNodeChanges(changes, get().nodes) })
    // Clear right-panel selection if the selected node was removed
    const removals = changes.filter((c) => c.type === 'remove').map((c) => c.id)
    if (removals.length > 0) {
      const selectedNodeId = useUIStore.getState().selectedNodeId
      if (selectedNodeId && removals.includes(selectedNodeId)) {
        useUIStore.getState().selectNode(null)
      }
    }
  },
  onEdgesChange: (changes) => {
    set({ edges: applyEdgeChanges(changes, get().edges) })
  },
  onConnect: (connection) => {
    set({ edges: addEdge({ ...connection, type: 'default' }, get().edges) })
  },
  addNode: (type, position) => {
    const id = getId()
    const ntCfg = NODE_TYPES[type as NodeType]
    const newNode: Node<NodeData> = {
      id,
      type: 'upalNode',
      position,
      data: {
        label: ntCfg?.label || type,
        nodeType: type,
        description: ntCfg?.description || '',
        config: {},
      },
    }
    set({ nodes: [...get().nodes, newNode] })
  },
  updateNodeConfig: (nodeId, config) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === nodeId
          ? { ...n, data: { ...n.data, config: { ...n.data.config, ...config } } }
          : n,
      ),
    })
  },
  updateNodeLabel: (nodeId, label) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === nodeId ? { ...n, data: { ...n.data, label } } : n,
      ),
    })
  },
  updateNodeDescription: (nodeId, description) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === nodeId ? { ...n, data: { ...n.data, description } } : n,
      ),
    })
  },
  applyAutoLayout: () => {
    const { nodes, edges } = get()
    if (nodes.length === 0) return
    const { nodes: layouted } = getLayoutedElements<Node<NodeData>>(nodes, edges, 'LR')
    set({ nodes: layouted })
  },
  setWorkflowName: (name) => {
    set({ workflowName: name })
  },
  setOriginalName: (name) => {
    set({ originalName: name })
  },

  createGroup: (nodeIds) => {
    if (nodeIds.length === 0) return
    const groupId = getId()
    const { nodes } = get()
    const selectedNodes = nodes.filter((n) => nodeIds.includes(n.id))
    if (selectedNodes.length === 0) return

    // Compute bounding box of selected nodes
    const padding = 40
    const xs = selectedNodes.map((n) => n.position.x)
    const ys = selectedNodes.map((n) => n.position.y)
    const minX = Math.min(...xs) - padding
    const minY = Math.min(...ys) - padding
    const maxX = Math.max(...xs) + 320 + padding // approximate node width
    const maxY = Math.max(...ys) + 100 + padding // approximate node height

    const groupNode: Node<NodeData> = {
      id: groupId,
      type: 'groupNode',
      position: { x: minX, y: minY },
      style: { width: maxX - minX, height: maxY - minY },
      data: { label: 'Group', nodeType: 'group', description: '', config: { color: 'purple' } },
    }

    // Reparent selected nodes relative to the group
    const updatedNodes = nodes.map((n) => {
      if (nodeIds.includes(n.id)) {
        return {
          ...n,
          parentId: groupId,
          position: { x: n.position.x - minX, y: n.position.y - minY },
          extent: 'parent' as const,
        }
      }
      return n
    })

    // Group node must appear before its children in the array
    set({ nodes: [groupNode, ...updatedNodes] })
    useUIStore.getState().selectNode(groupId)
  },

  removeGroup: (groupId) => {
    const { nodes } = get()
    const groupNode = nodes.find((n) => n.id === groupId)
    if (!groupNode || groupNode.type !== 'groupNode') return

    const updatedNodes = nodes
      .filter((n) => n.id !== groupId)
      .map((n) => {
        if (n.parentId === groupId) {
          return {
            ...n,
            parentId: undefined,
            extent: undefined,
            position: {
              x: n.position.x + groupNode.position.x,
              y: n.position.y + groupNode.position.y,
            },
          }
        }
        return n
      })

    set({ nodes: updatedNodes })
    useUIStore.getState().selectNode(null)
  },

  updateGroupLabel: (groupId, label) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === groupId ? { ...n, data: { ...n.data, label } } : n,
      ),
    })
  },

  updateGroupColor: (groupId, color) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === groupId
          ? { ...n, data: { ...n.data, config: { ...n.data.config, color } } }
          : n,
      ),
    })
  },
}))
