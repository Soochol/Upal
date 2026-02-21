import { useMemo } from 'react'
import { NODE_TYPES } from '@/lib/nodeTypes'
import { computeLayout, edgePath, dominantCssVar } from '@/lib/workflowSvg'
import type { LayoutNode, LayoutEdge } from '@/lib/workflowSvg'

type Props = {
  nodes: LayoutNode[]
  edges: LayoutEdge[]
  /** Unique string used as SVG element ID prefix (prevents clashes between cards) */
  uid: string
}

// Map node type → CSS variable for dominantCssVar lookup
const CSS_VARS: Record<string, string> = Object.fromEntries(
  Object.entries(NODE_TYPES).map(([t, cfg]) => [t, cfg.cssVar]),
)

export function WorkflowMiniGraph({ nodes, edges, uid }: Props) {
  const positions = useMemo(() => computeLayout(nodes, edges), [nodes, edges])
  const domVar = useMemo(() => dominantCssVar(nodes, CSS_VARS), [nodes])

  // Sanitize uid for use in SVG id attributes
  const safeId = uid.replace(/[^a-zA-Z0-9-_]/g, '_')
  const dotsId = `dots-${safeId}`
  const glowId = `glow-${safeId}`

  return (
    <svg
      width="100%"
      height="68"
      viewBox="0 0 300 68"
      preserveAspectRatio="xMidYMid meet"
      aria-hidden="true"
    >
      <defs>
        {/* Subtle dot pattern */}
        <pattern id={dotsId} width="16" height="16" patternUnits="userSpaceOnUse">
          <circle cx="1.5" cy="1.5" r="0.75" fill="var(--foreground)" fillOpacity="0.055" />
        </pattern>

        {/* Radial glow from dominant node color */}
        <radialGradient id={glowId} cx="50%" cy="50%" r="55%">
          <stop offset="0%" stopColor={domVar} stopOpacity="0.14" />
          <stop offset="100%" stopColor="var(--card)" stopOpacity="0" />
        </radialGradient>
      </defs>

      {/* ── Background ── */}
      <rect width="300" height="68" fill="var(--card)" />
      <rect width="300" height="68" fill={`url(#${dotsId})`} />
      <rect width="300" height="68" fill={`url(#${glowId})`} />

      {/* ── Edges ── */}
      {edges.map((e) => {
        const src = positions.get(e.from)
        const tgt = positions.get(e.to)
        if (!src || !tgt) return null
        return (
          <path
            key={`${e.from}-${e.to}`}
            d={edgePath(src.x, src.y, tgt.x, tgt.y)}
            stroke="var(--border)"
            strokeWidth="1.2"
            fill="none"
            strokeLinecap="round"
            opacity="0.8"
          />
        )
      })}

      {/* ── Nodes ── */}
      {nodes.map((n) => {
        const pos = positions.get(n.id)
        if (!pos) return null
        const cssVar = NODE_TYPES[n.type as keyof typeof NODE_TYPES]?.cssVar ?? 'var(--muted-foreground)'
        return (
          <g key={n.id}>
            {/* Soft halo */}
            <circle cx={pos.x} cy={pos.y} r={8.5} fill={cssVar} fillOpacity={0.1} />
            {/* Main dot */}
            <circle
              cx={pos.x}
              cy={pos.y}
              r={5}
              fill={cssVar}
              fillOpacity={0.28}
              stroke={cssVar}
              strokeOpacity={0.65}
              strokeWidth={1.3}
            />
          </g>
        )
      })}

      {/* ── Empty state ── */}
      {nodes.length === 0 && (
        <text
          x="150"
          y="37"
          textAnchor="middle"
          dominantBaseline="middle"
          fontSize="11"
          fill="var(--muted-foreground)"
          fillOpacity="0.3"
          fontStyle="italic"
        >
          empty
        </text>
      )}
    </svg>
  )
}
