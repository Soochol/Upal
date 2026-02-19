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

export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'tool' | 'output' | 'external'
  description: string
  config: Record<string, unknown>
}

export type RunEvent = {
  type: string
  data: Record<string, unknown>
}

export type NodeRunStatus = 'idle' | 'running' | 'completed' | 'error'

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
  selectedNodeId: string | null
  selectNode: (id: string | null) => void

  // Workflow identity
  workflowName: string
  setWorkflowName: (name: string) => void

  // Execution state
  isRunning: boolean
  setIsRunning: (running: boolean) => void
  runEvents: RunEvent[]
  addRunEvent: (event: RunEvent) => void
  clearRunEvents: () => void

  // Node run status tracking
  nodeStatuses: Record<string, NodeRunStatus>
  setNodeStatus: (nodeId: string, status: NodeRunStatus) => void
  clearNodeStatuses: () => void
}

let nodeId = 0
const getId = () => `node_${++nodeId}`

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  nodes: [],
  edges: [],
  selectedNodeId: null,

  // Workflow identity
  workflowName: '',

  // Execution state
  isRunning: false,
  runEvents: [],
  nodeStatuses: {},

  onNodesChange: (changes) => {
    set({ nodes: applyNodeChanges(changes, get().nodes) })
  },
  onEdgesChange: (changes) => {
    set({ edges: applyEdgeChanges(changes, get().edges) })
  },
  onConnect: (connection) => {
    set({ edges: addEdge({ ...connection, type: 'smoothstep' }, get().edges) })
    get().applyAutoLayout()
  },
  addNode: (type, position) => {
    const id = getId()
    const labels: Record<string, string> = {
      input: 'User Input',
      agent: 'Agent',
      tool: 'Tool',
      output: 'Output',
      external: 'External Agent',
    }
    const descriptions: Record<string, string> = {
      input: 'User-provided data entry point',
      agent: 'AI model processing step',
      tool: 'External tool or function call',
      output: 'Workflow result endpoint',
      external: 'External A2A-compatible agent',
    }
    const newNode: Node<NodeData> = {
      id,
      type: 'upalNode',
      position,
      data: {
        label: labels[type] || type,
        nodeType: type,
        description: descriptions[type] || '',
        config: {},
      },
    }
    set({ nodes: [...get().nodes, newNode] })
    get().applyAutoLayout()
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
  selectNode: (id) => {
    set({ selectedNodeId: id })
  },
  setWorkflowName: (name) => {
    set({ workflowName: name })
  },
  setIsRunning: (running) => {
    set({ isRunning: running })
  },
  addRunEvent: (event) => {
    set({ runEvents: [...get().runEvents, event] })
  },
  clearRunEvents: () => {
    set({ runEvents: [] })
  },
  setNodeStatus: (nodeId, status) => {
    set({ nodeStatuses: { ...get().nodeStatuses, [nodeId]: status } })
  },
  clearNodeStatuses: () => {
    set({ nodeStatuses: {} })
  },
}))
