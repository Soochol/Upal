import { Trash2, Clock } from 'lucide-react'
import type { WorkflowDefinition } from '@/lib/serializer'
import { NODE_TYPES } from '@/lib/nodeTypes'

type WorkflowCardProps = {
  workflow: WorkflowDefinition
  description?: string
  onClick: () => void
  onDelete?: () => void
  scheduleCount?: number
  nextRun?: string
  isRunning?: boolean
}

export function WorkflowCard({
  workflow,
  description,
  onClick,
  onDelete,
  scheduleCount,
  nextRun,
  isRunning,
}: WorkflowCardProps) {
  // Count nodes by type
  const typeCounts: Record<string, number> = {}
  for (const n of workflow.nodes) {
    typeCounts[n.type] = (typeCounts[n.type] || 0) + 1
  }

  return (
    <div
      className={`group relative text-left w-full rounded-2xl p-5 transition-all duration-200
        bg-card/60 border border-border hover:border-foreground/20
        hover:bg-card hover:shadow-lg hover:shadow-black/10
        hover:-translate-y-0.5 cursor-pointer${isRunning ? ' workflow-card-running' : ''}`}
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter') onClick() }}
    >
      {onDelete && (
        <button
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          className="absolute top-3 right-3 p-1.5 rounded-lg opacity-0 group-hover:opacity-100
            hover:bg-destructive/10 text-muted-foreground hover:text-destructive
            transition-all duration-150 cursor-pointer"
          title="Delete workflow"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      )}

      {/* Mini node graph preview */}
      <div className="flex items-center gap-1.5 mb-4">
        {workflow.nodes.slice(0, 6).map((n, i) => (
          <div key={n.id} className="flex items-center gap-1.5">
            <div
              className="w-6 h-6 rounded-lg flex items-center justify-center"
              style={{
                background: `color-mix(in oklch, ${NODE_TYPES[n.type as keyof typeof NODE_TYPES]?.cssVar || 'var(--muted)'}, transparent 80%)`,
              }}
            >
              {(() => {
                const ntCfg = NODE_TYPES[n.type as keyof typeof NODE_TYPES]
                return ntCfg ? (
                  <span style={{ color: ntCfg.cssVar }}><ntCfg.icon className="w-3 h-3" /></span>
                ) : null
              })()}
            </div>
            {i < Math.min(workflow.nodes.length, 6) - 1 && (
              <div className="w-3 h-px bg-border" />
            )}
          </div>
        ))}
        {workflow.nodes.length > 6 && (
          <span className="text-[10px] text-muted-foreground ml-1">
            +{workflow.nodes.length - 6}
          </span>
        )}
      </div>

      {/* Name */}
      <h3 className="landing-display font-semibold text-base text-foreground group-hover:text-foreground truncate">
        {workflow.name}
      </h3>

      {/* Description */}
      {description && (
        <p className="text-sm text-muted-foreground mt-1.5 line-clamp-2 leading-relaxed">
          {description}
        </p>
      )}

      {/* Node type badges */}
      <div className="flex items-center gap-2 mt-4 flex-wrap">
        {Object.entries(typeCounts).map(([type, count]) => (
          <span
            key={type}
            className="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full landing-body"
            style={{
              background: `color-mix(in oklch, ${NODE_TYPES[type as keyof typeof NODE_TYPES]?.cssVar || 'var(--muted)'}, transparent 88%)`,
              color: NODE_TYPES[type as keyof typeof NODE_TYPES]?.cssVar || 'var(--muted-foreground)',
            }}
          >
            <span
              className="w-1.5 h-1.5 rounded-full"
              style={{ background: NODE_TYPES[type as keyof typeof NODE_TYPES]?.cssVar || 'var(--muted-foreground)' }}
            />
            {count} {type}
          </span>
        ))}
        <span className="text-[11px] text-muted-foreground/60 ml-auto landing-body">
          {workflow.edges.length} edge{workflow.edges.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Schedule badge */}
      {scheduleCount && scheduleCount > 0 && (
        <div className="flex items-center gap-1.5 mt-3 text-[11px] text-muted-foreground landing-body">
          <Clock className="w-3 h-3" />
          <span>{scheduleCount} schedule{scheduleCount !== 1 ? 's' : ''}</span>
          {nextRun && (
            <span className="ml-auto text-muted-foreground/60">
              next: {new Date(nextRun).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}
            </span>
          )}
        </div>
      )}
    </div>
  )
}
