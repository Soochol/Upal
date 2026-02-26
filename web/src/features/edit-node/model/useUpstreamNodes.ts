import { useMemo } from 'react'
import { useWorkflowStore } from '@/entities/workflow'
import type { Node, Edge } from '@xyflow/react'
import type { NodeData } from '@/entities/workflow'

export type UpstreamNode = {
  id: string
  type: string
  label: string
}

/** Pure function — usable outside React hooks (e.g. in callbacks). */
export function computeUpstreamNodes(
  nodeId: string,
  nodes: Node<NodeData>[],
  edges: Edge[],
): UpstreamNode[] {
  const sourceIds = new Set(
    edges.filter((e) => e.target === nodeId).map((e) => e.source),
  )
  return nodes
    .filter((n) => sourceIds.has(n.id) && n.type !== 'groupNode')
    .map((n) => ({
      id: n.id,
      type: n.data.nodeType as string,
      label: n.data.label,
    }))
}

export function useUpstreamNodes(nodeId: string): UpstreamNode[] {
  const edges = useWorkflowStore((s) => s.edges)
  const allNodes = useWorkflowStore((s) => s.nodes)
  return useMemo(
    () => computeUpstreamNodes(nodeId, allNodes, edges),
    [edges, allNodes, nodeId],
  )
}
