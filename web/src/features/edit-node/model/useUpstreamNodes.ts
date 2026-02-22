import { useMemo } from 'react'
import { useWorkflowStore } from '@/entities/workflow'

export type UpstreamNode = {
  id: string
  type: string
  label: string
}

export function useUpstreamNodes(nodeId: string): UpstreamNode[] {
  const edges = useWorkflowStore((s) => s.edges)
  const allNodes = useWorkflowStore((s) => s.nodes)
  return useMemo(() => {
    const sourceIds = new Set(
      edges.filter((e) => e.target === nodeId).map((e) => e.source),
    )
    return allNodes
      .filter((n) => sourceIds.has(n.id) && n.type !== 'groupNode')
      .map((n) => ({
        id: n.id,
        type: n.data.nodeType as string,
        label: n.data.label,
      }))
  }, [edges, allNodes, nodeId])
}
