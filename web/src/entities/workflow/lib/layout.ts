import Dagre from '@dagrejs/dagre'
import type { Node, Edge } from '@xyflow/react'

type LayoutDirection = 'LR' | 'TB'

export function getLayoutedElements<T extends Node>(
  nodes: T[],
  edges: Edge[],
  direction: LayoutDirection = 'LR',
): { nodes: T[]; edges: Edge[] } {
  const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}))

  g.setGraph({
    rankdir: direction,
    nodesep: 80,
    ranksep: 120,
    marginx: 50,
    marginy: 50,
  })

  nodes.forEach((node) => {
    g.setNode(node.id, { width: 280, height: 80 })
  })

  edges.forEach((edge) => {
    g.setEdge(edge.source, edge.target)
  })

  Dagre.layout(g)

  const layoutedNodes = nodes.map((node) => {
    const pos = g.node(node.id)
    return {
      ...node,
      position: { x: pos.x - 140, y: pos.y - 40 },
    } as T
  })

  return { nodes: layoutedNodes, edges }
}
