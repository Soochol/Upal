import { Trash2 } from 'lucide-react'
import type { WorkflowDefinition } from '@/lib/serializer'
import { NODE_TYPES } from '@/lib/nodeTypes'
import { WorkflowMiniGraph } from '@/components/WorkflowMiniGraph'

type WorkflowCardProps = {
  workflow: WorkflowDefinition
  description?: string
  onClick: () => void
  onDelete?: () => void
  isRunning?: boolean
}

export function WorkflowCard({
  workflow,
  description,
  onClick,
  onDelete,
  isRunning,
}: WorkflowCardProps) {
  const typeCounts: Record<string, number> = {}
  for (const n of workflow.nodes) {
    typeCounts[n.type] = (typeCounts[n.type] || 0) + 1
  }

  return (
    <div
      className={`group relative text-left w-full rounded-2xl overflow-hidden transition-all duration-200
        bg-card border border-border hover:border-foreground/20
        hover:shadow-lg hover:shadow-black/8 hover:-translate-y-0.5 cursor-pointer
        ${isRunning ? 'workflow-card-running' : ''}`}
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter') onClick() }}
    >
      {onDelete && (
        <button
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          className="absolute top-2.5 right-2.5 z-10 p-1.5 rounded-lg opacity-0 group-hover:opacity-100
            hover:bg-destructive/10 text-muted-foreground hover:text-destructive
            transition-all duration-150 cursor-pointer"
          title="Delete workflow"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      )}

      {/* ── Canvas preview area ── */}
      <div className="relative h-[68px] border-b border-border overflow-hidden">
        {workflow.thumbnail_svg ? (
          <div
            className="workflow-thumbnail w-full h-full"
            dangerouslySetInnerHTML={{ __html: workflow.thumbnail_svg }}
          />
        ) : (
          <WorkflowMiniGraph
            nodes={workflow.nodes}
            edges={workflow.edges}
            uid={workflow.name}
          />
        )}
      </div>

      {/* ── Card body ── */}
      <div className="px-4 pt-3 pb-3.5">
        {/* Name */}
        <h3 className="landing-display font-semibold text-sm text-foreground truncate pr-5 leading-snug">
          {workflow.name}
        </h3>

        {/* Description */}
        {description && (
          <p className="text-xs text-muted-foreground mt-1 line-clamp-1 leading-relaxed">
            {description}
          </p>
        )}

        {/* Node type badges */}
        <div className="flex items-center gap-1.5 mt-2.5 flex-wrap">
          {Object.entries(typeCounts).map(([type, count]) => {
            const cssVar = NODE_TYPES[type as keyof typeof NODE_TYPES]?.cssVar || 'var(--muted-foreground)'
            return (
              <span
                key={type}
                className="inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full landing-body font-medium"
                style={{
                  background: `color-mix(in oklch, ${cssVar}, transparent 88%)`,
                  color: cssVar,
                }}
              >
                <span className="w-1 h-1 rounded-full shrink-0" style={{ background: cssVar }} />
                {count} {type}
              </span>
            )
          })}
          <span className="text-[10px] text-muted-foreground/40 ml-auto landing-body tabular-nums">
            {workflow.edges.length}e
          </span>
        </div>

      </div>
    </div>
  )
}
