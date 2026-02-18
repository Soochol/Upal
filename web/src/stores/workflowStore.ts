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

export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'tool' | 'output'
  config: Record<string, unknown>
}

export type RunEvent = {
  type: string
  data: Record<string, unknown>
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

  onNodesChange: (changes) => {
    set({ nodes: applyNodeChanges(changes, get().nodes) })
  },
  onEdgesChange: (changes) => {
    set({ edges: applyEdgeChanges(changes, get().edges) })
  },
  onConnect: (connection) => {
    set({ edges: addEdge(connection, get().edges) })
  },
  addNode: (type, position) => {
    const id = getId()
    const labels: Record<string, string> = {
      input: 'User Input',
      agent: 'Agent',
      tool: 'Tool',
      output: 'Output',
    }
    const newNode: Node<NodeData> = {
      id,
      type: 'upalNode',
      position,
      data: {
        label: labels[type] || type,
        nodeType: type,
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
}))
