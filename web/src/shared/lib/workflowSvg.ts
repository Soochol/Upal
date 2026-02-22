/** DAG layout for WorkflowMiniGraph — no external dependencies */

export type LayoutNode = { id: string; type: string }
export type LayoutEdge = { from: string; to: string }
export type NodePos = { x: number; y: number }

const W = 300
const H = 68
const MARGIN_X = 22
const MARGIN_Y = 12
const MAX_LAYERS = 9

/**
 * Assigns topological layers then positions nodes within each layer.
 * Returns a Map<nodeId, {x, y}> for the 300×68 viewBox.
 */
export function computeLayout(
  nodes: LayoutNode[],
  edges: LayoutEdge[],
): Map<string, NodePos> {
  const positions = new Map<string, NodePos>()
  if (nodes.length === 0) return positions

  // ── 1. Build adjacency ──────────────────────────────────────────────────
  const inDegree = new Map<string, number>()
  const adj = new Map<string, string[]>()
  for (const n of nodes) {
    inDegree.set(n.id, 0)
    adj.set(n.id, [])
  }
  for (const e of edges) {
    if (!adj.has(e.from) || !adj.has(e.to)) continue
    adj.get(e.from)!.push(e.to)
    inDegree.set(e.to, (inDegree.get(e.to) ?? 0) + 1)
  }

  // ── 2. Assign layers (longest-path BFS) ────────────────────────────────
  const layer = new Map<string, number>()
  // Start with source nodes (in-degree 0)
  const queue: string[] = []
  for (const n of nodes) {
    if ((inDegree.get(n.id) ?? 0) === 0) {
      layer.set(n.id, 0)
      queue.push(n.id)
    }
  }
  // BFS: assign each successor the max layer of its predecessors + 1
  let head = 0
  while (head < queue.length) {
    const id = queue[head++]
    const l = layer.get(id) ?? 0
    for (const next of (adj.get(id) ?? [])) {
      const newL = Math.max((layer.get(next) ?? 0), l + 1)
      layer.set(next, newL)
      queue.push(next)
    }
  }
  // Isolated nodes not reached
  for (const n of nodes) {
    if (!layer.has(n.id)) layer.set(n.id, 0)
  }

  // ── 3. Group by layer ──────────────────────────────────────────────────
  const layerGroups = new Map<number, string[]>()
  for (const [id, l] of layer.entries()) {
    const capped = Math.min(l, MAX_LAYERS - 1)
    if (!layerGroups.has(capped)) layerGroups.set(capped, [])
    layerGroups.get(capped)!.push(id)
  }

  // ── 4. Compute positions ───────────────────────────────────────────────
  const numLayers = Math.max(...layerGroups.keys()) + 1
  // Distribute columns across usable width
  const colSpan = numLayers <= 1
    ? 0
    : (W - 2 * MARGIN_X) / (numLayers - 1)

  for (const [l, ids] of layerGroups.entries()) {
    const x = numLayers === 1 ? W / 2 : MARGIN_X + l * colSpan
    const usableH = H - 2 * MARGIN_Y
    const rowSpan = ids.length <= 1 ? 0 : usableH / (ids.length - 1)
    ids.forEach((id, i) => {
      const y = ids.length === 1 ? H / 2 : MARGIN_Y + i * rowSpan
      positions.set(id, { x, y })
    })
  }

  return positions
}

/** Cubic bezier path between two points with horizontal control handles */
export function edgePath(x1: number, y1: number, x2: number, y2: number): string {
  const cx = (x1 + x2) / 2
  return `M${x1},${y1} C${cx},${y1} ${cx},${y2} ${x2},${y2}`
}

/** Pick the CSS variable of the most-common node type */
export function dominantCssVar(
  nodes: LayoutNode[],
  cssVarByType: Record<string, string>,
  fallback = 'var(--node-agent)',
): string {
  if (nodes.length === 0) return fallback
  const counts: Record<string, number> = {}
  for (const n of nodes) counts[n.type] = (counts[n.type] ?? 0) + 1
  const top = Object.entries(counts).sort((a, b) => b[1] - a[1])[0][0]
  return cssVarByType[top] ?? fallback
}
