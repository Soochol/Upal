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
import { getLayoutedElements } from '../lib/layout'
import { getNodeDefinition } from '@/entities/node'
import type { NodeType } from '@/entities/node'
import type { NodeData } from '../types'

// Node types whose prompt field should receive auto-inserted template references on connect
const AUTO_PROMPT_TYPES = new Set(['agent', 'output'])

type WorkflowState = {
  nodes: Node<NodeData>[]
  edges: Edge[]
  onNodesChange: OnNodesChange<Node<NodeData>>
  onEdgesChange: OnEdgesChange
  onConnect: OnConnect
  addNode: (type: NodeData['nodeType'], position: { x: number; y: number }, initialConfig?: Record<string, unknown>) => void
  removeNode: (id: string) => void
  getNode: (id: string) => Node<NodeData> | undefined
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
  createGroup: (nodeIds: string[]) => string | undefined
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
  },
  onEdgesChange: (changes) => {
    // Auto-remove {{source}} template references from target node prompts on edge removal
    const removals = changes.filter((c) => c.type === 'remove')
    if (removals.length > 0) {
      const { edges, nodes } = get()
      const removedEdges = removals
        .map((c) => edges.find((e) => e.id === c.id))
        .filter(Boolean) as Edge[]

      let updatedNodes = nodes
      for (const edge of removedEdges) {
        const target = updatedNodes.find((n) => n.id === edge.target)
        if (!target || !AUTO_PROMPT_TYPES.has(target.data.nodeType)) continue
        const prompt = (target.data.config.prompt as string) ?? ''
        if (!prompt) continue

        const ref = `{{${edge.source}}}`
        if (!prompt.includes(ref)) continue

        // Remove the reference and clean up extra blank lines
        const newPrompt = prompt
          .split('\n')
          .filter((line) => line.trim() !== ref)
          .join('\n')

        updatedNodes = updatedNodes.map((n) =>
          n.id === edge.target
            ? { ...n, data: { ...n.data, config: { ...n.data.config, prompt: newPrompt } } }
            : n,
        )
      }
      if (updatedNodes !== nodes) {
        set({ nodes: updatedNodes })
      }
    }
    set({ edges: applyEdgeChanges(changes, get().edges) })
  },
  onConnect: (connection) => {
    set({ edges: addEdge({ ...connection, type: 'default' }, get().edges) })

    // Auto-insert {{source}} template reference into target node's prompt
    if (!connection.source || !connection.target) return
    const targetNode = get().nodes.find((n) => n.id === connection.target)
    if (!targetNode || !AUTO_PROMPT_TYPES.has(targetNode.data.nodeType)) return

    const currentPrompt = (targetNode.data.config.prompt as string) ?? ''
    const ref = `{{${connection.source}}}`
    if (currentPrompt.includes(ref)) return

    const newPrompt = currentPrompt ? `${currentPrompt}\n${ref}` : ref
    set({
      nodes: get().nodes.map((n) =>
        n.id === connection.target
          ? { ...n, data: { ...n.data, config: { ...n.data.config, prompt: newPrompt } } }
          : n,
      ),
    })
  },
  addNode: (type, position, initialConfig) => {
    const id = getId()
    let label: string = type
    let description = ''
    // 'group' is not a registry NodeType — only look up registered types
    if (type !== 'group') {
      try {
        const def = getNodeDefinition(type as NodeType)
        label = def.label
        description = def.description
      } catch {
        // Unknown type — use raw type string as label
      }
    }
    const newNode: Node<NodeData> = {
      id,
      type: 'upalNode',
      position,
      data: {
        label,
        nodeType: type,
        description,
        config: initialConfig ?? {},
      },
    }
    set({ nodes: [...get().nodes, newNode] })
  },
  removeNode: (id) => set({ nodes: get().nodes.filter((n) => n.id !== id) }),
  getNode: (id) => get().nodes.find((n) => n.id === id),
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
    if (nodeIds.length === 0) return undefined
    const groupId = getId()
    const { nodes } = get()
    const selectedNodes = nodes.filter((n) => nodeIds.includes(n.id))
    if (selectedNodes.length === 0) return undefined

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
    return groupId
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
