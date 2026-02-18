import type { Node, Edge } from '@xyflow/react'
import type { NodeData } from '../stores/workflowStore'

export type WorkflowDefinition = {
  name: string
  version: number
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
}

type WorkflowNode = {
  id: string
  type: string
  config: Record<string, unknown>
}

type WorkflowEdge = {
  from: string
  to: string
  loop?: { max_iterations: number; exit_when: string }
}

export function serializeWorkflow(
  name: string,
  nodes: Node<NodeData>[],
  edges: Edge[],
): WorkflowDefinition {
  return {
    name,
    version: 1,
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.data.nodeType,
      config: n.data.config,
    })),
    edges: edges.map((e) => ({
      from: e.source,
      to: e.target,
    })),
  }
}

export function deserializeWorkflow(
  wf: WorkflowDefinition,
): { nodes: Node<NodeData>[]; edges: Edge[] } {
  const labels: Record<string, string> = {
    input: 'User Input',
    agent: 'Agent',
    tool: 'Tool',
    output: 'Output',
  }

  const nodes: Node<NodeData>[] = wf.nodes.map((n, i) => ({
    id: n.id,
    type: 'upalNode',
    position: { x: 250, y: i * 150 + 50 },
    data: {
      label: labels[n.type] || n.type,
      nodeType: n.type as NodeData['nodeType'],
      config: n.config,
    },
  }))

  const edges: Edge[] = wf.edges.map((e, i) => ({
    id: `edge-${i}`,
    source: e.from,
    target: e.to,
  }))

  return { nodes, edges }
}
