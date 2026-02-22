import type { Node, Edge } from '@xyflow/react'
import type { NodeData } from '@/entities/workflow'

/** Return input nodes in DAG topological order (Kahn's algorithm). */
export function getInputNodesInOrder(
  nodes: Node<NodeData>[],
  edges: Edge[],
): Node<NodeData>[] {
  const regularNodes = nodes.filter((n) => n.type !== 'groupNode')
  const inDegree = new Map<string, number>()
  const adj = new Map<string, string[]>()

  for (const node of regularNodes) {
    inDegree.set(node.id, 0)
    adj.set(node.id, [])
  }

  for (const edge of edges) {
    if (adj.has(edge.source)) {
      adj.get(edge.source)!.push(edge.target)
      inDegree.set(edge.target, (inDegree.get(edge.target) ?? 0) + 1)
    }
  }

  const queue: string[] = []
  for (const [id, deg] of inDegree) {
    if (deg === 0) queue.push(id)
  }

  const sorted: string[] = []
  while (queue.length > 0) {
    const id = queue.shift()!
    sorted.push(id)
    for (const next of adj.get(id) ?? []) {
      const newDeg = (inDegree.get(next) ?? 1) - 1
      inDegree.set(next, newDeg)
      if (newDeg === 0) queue.push(next)
    }
  }

  const nodeMap = new Map(regularNodes.map((n) => [n.id, n]))
  return sorted
    .filter((id) => nodeMap.get(id)?.data.nodeType === 'input')
    .map((id) => nodeMap.get(id)!)
}
