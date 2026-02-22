import { Trash2 } from 'lucide-react'
import type { WorkflowDefinition } from '@/entities/workflow'
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
  return (
    <div
      className={`group relative text-left w-full rounded-2xl overflow-hidden transition-all duration-200
        bg-card border border-border hover:border-foreground/20
        hover:shadow-lg hover:shadow-black/8 hover:-translate-y-0.5 cursor-pointer min-h-[164px]
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
        <div className="flex items-center gap-2 mt-2.5">
          <span className="text-[10px] text-muted-foreground/50 landing-body tabular-nums">
            {workflow.nodes.length} node{workflow.nodes.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>
    </div>
  )
}
