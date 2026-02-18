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

type WorkflowState = {
  nodes: Node<NodeData>[]
  edges: Edge[]
  onNodesChange: OnNodesChange<Node<NodeData>>
  onEdgesChange: OnEdgesChange
  onConnect: OnConnect
  addNode: (type: NodeData['nodeType'], position: { x: number; y: number }) => void
}

let nodeId = 0
const getId = () => `node_${++nodeId}`

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  nodes: [],
  edges: [],
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
}))
