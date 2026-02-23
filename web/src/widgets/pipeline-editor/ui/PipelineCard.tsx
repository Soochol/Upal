import { Play, Trash2, Clock, Zap, RefreshCw, GitBranch, PauseCircle, Loader2, Rss } from 'lucide-react'
import type { Pipeline } from '@/shared/types'
import { humanReadableCron } from '@/shared/lib/cron'

const stageTypeConfig: Record<string, { icon: typeof GitBranch; color: string }> = {
  workflow: { icon: Play, color: 'var(--info)' },
  approval: { icon: PauseCircle, color: 'var(--warning)' },
  schedule: { icon: Clock, color: 'var(--success)' },
  trigger: { icon: Zap, color: 'var(--node-agent)' },
  transform: { icon: RefreshCw, color: 'var(--muted-foreground)' },
  collect: { icon: Rss, color: 'var(--info)' },
}

type Props = {
  pipeline: Pipeline
  onClick: () => void
  onStart: () => void
  onCollect?: () => void
  onDelete: () => void
  isCollecting?: boolean
}

export function PipelineCard({ pipeline, onClick, onStart, onCollect, onDelete, isCollecting }: Props) {
  const hasContentMeta = (pipeline.sources && pipeline.sources.length > 0) || !!pipeline.schedule
  const pendingCount = pipeline.pending_session_count ?? 0

  return (
    <div
      className="group relative w-full rounded-2xl overflow-hidden glass-panel border border-white/5
        hover:border-white/10 hover:bg-white/5 hover:-translate-y-1 hover:shadow-[0_12px_24px_rgba(0,0,0,0.2)]
        transition-all duration-300 cursor-pointer"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter') onClick() }}
    >
      {/* Action buttons — appear on hover */}
      <div className="absolute top-2.5 right-2.5 z-10 flex items-center gap-0.5
        opacity-0 group-hover:opacity-100 transition-opacity duration-150">
        {onCollect && (
          <button
            onClick={(e) => { e.stopPropagation(); onCollect() }}
            disabled={isCollecting}
            className="p-1.5 rounded-lg hover:bg-primary/10 text-muted-foreground hover:text-primary transition-colors disabled:opacity-60"
            title="Collect now"
          >
            {isCollecting
              ? <Loader2 className="w-3.5 h-3.5 animate-spin" />
              : <Rss className="w-3.5 h-3.5" />
            }
          </button>
        )}
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
      <div className="relative h-[68px] border-b border-white/5 bg-black/10 dark:bg-black/20 overflow-hidden">
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
      <div className="px-4 pt-3 pb-3.5 min-h-[96px] flex flex-col">
        <h3 className="landing-display font-semibold text-sm text-foreground truncate pr-14 leading-snug">
          {pipeline.name}
        </h3>
        <p className="text-xs text-muted-foreground mt-1 line-clamp-1 leading-relaxed min-h-[1.125rem]">
          {pipeline.description || '\u00A0'}
        </p>

        <div className="flex items-center gap-2 mt-auto pt-2 flex-wrap">
          <span className="text-[10px] text-muted-foreground/50 landing-body tabular-nums">
            {pipeline.stages.length} stage{pipeline.stages.length !== 1 ? 's' : ''}
          </span>

          {/* Content pipeline metadata */}
          {hasContentMeta && (
            <>
              {pipeline.schedule && (
                <span className="inline-flex items-center gap-0.5 text-[10px] text-muted-foreground/50">
                  <Clock className="h-2.5 w-2.5" />
                  {humanReadableCron(pipeline.schedule)}
                </span>
              )}
              {(pipeline.sources?.length ?? 0) > 0 && (
                <span className="text-[10px] text-muted-foreground/50">
                  {pipeline.sources!.length} source{pipeline.sources!.length !== 1 ? 's' : ''}
                </span>
              )}
            </>
          )}

          {/* Pending review badge */}
          {pendingCount > 0 && (
            <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full
              bg-warning/10 text-warning text-[10px] font-medium border border-warning/20">
              {pendingCount} pending
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
