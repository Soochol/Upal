import type { Node, Edge } from '@xyflow/react'
import type { NodeData } from '../stores/workflowStore'
import { getLayoutedElements } from '@/lib/layout'
import { NODE_TYPES, type NodeType } from '@/lib/nodeTypes'

export type WorkflowGroup = {
  id: string
  label: string
  color?: string
}

export type WorkflowDefinition = {
  name: string
  version: number
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  groups?: WorkflowGroup[]
}

type WorkflowNode = {
  id: string
  type: string
  config: Record<string, unknown>
  group?: string
}

type WorkflowEdge = {
  from: string
  to: string
  loop?: { max_iterations: number; exit_when: string }
  condition?: string
  trigger_rule?: 'on_success' | 'on_failure' | 'always'
}

export function serializeWorkflow(
  name: string,
  nodes: Node<NodeData>[],
  edges: Edge[],
): WorkflowDefinition {
  // Separate group nodes from regular nodes
  const groupNodes = nodes.filter((n) => n.type === 'groupNode')
  const regularNodes = nodes.filter((n) => n.type !== 'groupNode')

  const groups: WorkflowGroup[] = groupNodes.map((g) => ({
    id: g.id,
    label: g.data.label || 'Group',
    color: (g.data.config.color as string) || undefined,
  }))

  const wf: WorkflowDefinition = {
    name,
    version: 1,
    nodes: regularNodes.map((n) => {
      const config = n.data.description
        ? { ...n.data.config, description: n.data.description }
        : n.data.config
      const node: WorkflowNode = {
        id: n.id,
        type: n.data.nodeType,
        config,
      }
      if (n.parentId) {
        node.group = n.parentId
      }
      return node
    }),
    edges: edges.map((e) => {
      const edge: WorkflowEdge = {
        from: e.source,
        to: e.target,
      }
      if (e.data?.condition) {
        edge.condition = e.data.condition as string
      }
      if (e.data?.trigger_rule && e.data.trigger_rule !== 'on_success') {
        edge.trigger_rule = e.data.trigger_rule as WorkflowEdge['trigger_rule']
      }
      return edge
    }),
  }

  if (groups.length > 0) {
    wf.groups = groups
  }

  return wf
}

/** Convert a snake_case / kebab-case node ID into a Title Case label. */
function humanizeId(id: string): string {
  return id
    .replace(/[_-]/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}

export function deserializeWorkflow(
  wf: WorkflowDefinition,
): { nodes: Node<NodeData>[]; edges: Edge[] } {
  // Fixed labels for structural node types; agent derives from node ID
  const fixedLabels: Record<string, string> = {
    input: 'User Input',
    output: 'Output',
  }

  // Create group nodes first (they must precede children in the array)
  const groupNodes: Node<NodeData>[] = (wf.groups ?? []).map((g) => ({
    id: g.id,
    type: 'groupNode',
    position: { x: 0, y: 0 },
    style: { width: 400, height: 300 },
    data: {
      label: g.label,
      nodeType: 'group' as const,
      description: '',
      config: { color: g.color ?? 'purple' },
    },
  }))

  // Build lookup for nodes that belong to groups
  const groupMembership = new Map<string, string>()
  for (const n of wf.nodes) {
    if (n.group) {
      groupMembership.set(n.id, n.group)
    }
  }

  const regularNodes: Node<NodeData>[] = wf.nodes.map((n, i) => {
    // Migrate legacy output node config fields
    if (n.type === 'output') {
      if (n.config.layout_model && !n.config.model) {
        n.config.model = n.config.layout_model
      }
      delete n.config.layout_model
      delete n.config.display_mode
    }

    const node: Node<NodeData> = {
      id: n.id,
      type: 'upalNode',
      position: { x: i * 350, y: 0 },
      data: {
        label: (n.config.label as string) || fixedLabels[n.type] || humanizeId(n.id),
        nodeType: n.type as NodeData['nodeType'],
        description: (n.config.description as string) || NODE_TYPES[n.type as NodeType]?.description || '',
        config: n.config,
      },
    }
    const parentGroup = groupMembership.get(n.id)
    if (parentGroup) {
      node.parentId = parentGroup
      node.extent = 'parent'
    }
    return node
  })

  const allNodes: Node<NodeData>[] = [...groupNodes, ...regularNodes]

  const edges: Edge[] = wf.edges.map((e, i) => {
    const edge: Edge = {
      id: `edge-${i}`,
      source: e.from,
      target: e.to,
      type: 'default',
    }
    if (e.condition || (e.trigger_rule && e.trigger_rule !== 'on_success')) {
      edge.data = {
        ...(e.condition && { condition: e.condition }),
        ...(e.trigger_rule && { trigger_rule: e.trigger_rule }),
      }
    }
    return edge
  })

  // Only apply auto-layout to non-grouped nodes
  if (groupNodes.length === 0) {
    return getLayoutedElements<Node<NodeData>>(allNodes, edges, 'LR')
  }

  // When groups exist, layout only ungrouped nodes and position grouped nodes within their groups
  const ungroupedNodes = allNodes.filter((n) => !n.parentId && n.type !== 'groupNode')
  const { nodes: layoutedUngrouped } = getLayoutedElements<Node<NodeData>>(ungroupedNodes, edges, 'LR')

  // Position grouped children within their parent group
  const groupedChildren = allNodes.filter((n) => n.parentId)
  const byGroup = new Map<string, Node<NodeData>[]>()
  for (const n of groupedChildren) {
    const list = byGroup.get(n.parentId!) ?? []
    list.push(n)
    byGroup.set(n.parentId!, list)
  }
  const layoutedGrouped: Node<NodeData>[] = []
  for (const [, children] of byGroup) {
    children.forEach((n, i) => {
      layoutedGrouped.push({ ...n, position: { x: 40, y: 50 + i * 120 } })
    })
  }

  // Recompute group sizes based on children count
  const finalGroupNodes = groupNodes.map((g) => {
    const count = byGroup.get(g.id)?.length ?? 0
    return {
      ...g,
      style: { width: 400, height: Math.max(300, 50 + count * 120 + 40) },
    }
  })

  return {
    nodes: [...finalGroupNodes, ...layoutedUngrouped, ...layoutedGrouped],
    edges,
  }
}
