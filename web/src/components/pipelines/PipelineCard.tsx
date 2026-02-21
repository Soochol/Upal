import { Play, Trash2, Clock, Zap, RefreshCw, GitBranch, PauseCircle } from 'lucide-react'
import type { Pipeline } from '@/lib/api/types'

const stageTypeConfig: Record<string, { icon: typeof GitBranch; color: string }> = {
  workflow:  { icon: Play,        color: 'var(--info)' },
  approval:  { icon: PauseCircle, color: 'var(--warning)' },
  schedule:  { icon: Clock,       color: 'var(--success)' },
  trigger:   { icon: Zap,         color: 'var(--node-agent)' },
  transform: { icon: RefreshCw,   color: 'var(--muted-foreground)' },
}

type Props = {
  pipeline: Pipeline
  onClick: () => void
  onStart: () => void
  onDelete: () => void
}

export function PipelineCard({ pipeline, onClick, onStart, onDelete }: Props) {
  return (
    <div
      className="group relative w-full rounded-2xl overflow-hidden border border-border bg-card
        hover:border-foreground/20 hover:shadow-lg hover:shadow-black/8 hover:-translate-y-0.5
        transition-all duration-200 cursor-pointer"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter') onClick() }}
    >
      {/* Action buttons — appear on hover */}
      <div className="absolute top-2.5 right-2.5 z-10 flex items-center gap-0.5
        opacity-0 group-hover:opacity-100 transition-opacity duration-150">
        <button
          onClick={(e) => { e.stopPropagation(); onStart() }}
          className="p-1.5 rounded-lg hover:bg-info/10 text-muted-foreground hover:text-info transition-colors"
          title="Run pipeline"
        >
          <Play className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={(e) => { e.stopPropagation(); onDelete() }}
          className="p-1.5 rounded-lg hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
          title="Delete pipeline"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* ── Stage flow canvas ── */}
      <div className="relative h-[68px] border-b border-border overflow-hidden">
        {pipeline.thumbnail_svg ? (
          <div
            className="pipeline-thumbnail w-full h-full"
            dangerouslySetInnerHTML={{ __html: pipeline.thumbnail_svg }}
          />
        ) : (
          <>
            <div className="hero-dot-grid absolute inset-0 opacity-50 pointer-events-none bg-card" />
            <div className="relative flex items-center justify-center h-full px-4 gap-0 bg-card">
              {pipeline.stages.length === 0 ? (
                <span className="text-[11px] text-muted-foreground/30 italic">no stages</span>
              ) : (
                pipeline.stages.slice(0, 7).map((stage, i) => {
                  const cfg = stageTypeConfig[stage.type] ?? { icon: GitBranch, color: 'var(--muted-foreground)' }
                  const Icon = cfg.icon
                  return (
                    <div key={stage.id} className="flex items-center gap-0 shrink-0">
                      <div
                        className="w-7 h-7 rounded-lg flex items-center justify-center"
                        style={{
                          background: `color-mix(in oklch, ${cfg.color}, transparent 78%)`,
                          border: `1px solid color-mix(in oklch, ${cfg.color}, transparent 55%)`,
                        }}
                      >
                        <Icon className="w-3.5 h-3.5" style={{ color: cfg.color }} />
                      </div>
                      {i < Math.min(pipeline.stages.length, 7) - 1 && (
                        <div className="w-3 h-px bg-border" />
                      )}
                    </div>
                  )
                })
              )}
              {pipeline.stages.length > 7 && (
                <span className="text-[9px] text-muted-foreground/60 ml-2 font-mono tabular-nums">
                  +{pipeline.stages.length - 7}
                </span>
              )}
            </div>
          </>
        )}
      </div>

      {/* ── Card body ── */}
      <div className="px-4 pt-3 pb-3.5">
        <h3 className="landing-display font-semibold text-sm text-foreground truncate pr-14 leading-snug">
          {pipeline.name}
        </h3>
        {pipeline.description && (
          <p className="text-xs text-muted-foreground mt-1 line-clamp-1 leading-relaxed">
            {pipeline.description}
          </p>
        )}
        <div className="flex items-center gap-2 mt-2.5">
          <span className="text-[10px] text-muted-foreground/50 landing-body tabular-nums">
            {pipeline.stages.length} stage{pipeline.stages.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>
    </div>
  )
}
