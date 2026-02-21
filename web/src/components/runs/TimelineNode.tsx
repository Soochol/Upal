import { CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react'
import type { NodeRunRecord } from '@/lib/api'

type Props = {
  nodeRun: NodeRunRecord
  durationMs: number
  maxDurationMs: number
  isSelected: boolean
  onClick: () => void
}

const statusStyles: Record<string, { icon: typeof Clock; color: string; bgColor: string }> = {
  pending:   { icon: Clock,        color: 'text-warning',     bgColor: 'bg-warning/10' },
  running:   { icon: Loader2,      color: 'text-info',        bgColor: 'bg-info/10' },
  completed: { icon: CheckCircle2, color: 'text-success',     bgColor: 'bg-success/10' },
  success:   { icon: CheckCircle2, color: 'text-success',     bgColor: 'bg-success/10' },
  failed:    { icon: XCircle,      color: 'text-destructive', bgColor: 'bg-destructive/10' },
  error:     { icon: XCircle,      color: 'text-destructive', bgColor: 'bg-destructive/10' },
}

function formatDuration(ms: number): string {
  if (ms === 0) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export function TimelineNode({ nodeRun, durationMs, maxDurationMs, isSelected, onClick }: Props) {
  const style = statusStyles[nodeRun.status] ?? statusStyles.pending
  const Icon = style.icon
  const barWidth = maxDurationMs > 0 ? Math.max(8, (durationMs / maxDurationMs) * 100) : 8

  return (
    <button
      onClick={onClick}
      className={`relative flex flex-col items-center gap-1 px-3 py-2 rounded-lg border transition-all min-w-[90px] cursor-pointer ${
        isSelected
          ? 'border-primary bg-accent shadow-sm'
          : 'border-border hover:border-primary/40 hover:bg-accent/50'
      }`}
    >
      <div className="flex items-center gap-1.5">
        <Icon className={`h-3.5 w-3.5 ${style.color} ${nodeRun.status === 'running' ? 'animate-spin' : ''}`} />
        <span className="text-xs font-mono text-foreground truncate max-w-[80px]">
          {nodeRun.node_id}
        </span>
      </div>

      {/* Duration bar */}
      <div className="w-full h-1.5 bg-muted rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${style.bgColor.replace('/10', '/40')}`}
          style={{ width: `${barWidth}%` }}
        />
      </div>

      <span className="text-[10px] text-muted-foreground font-mono">
        {formatDuration(durationMs)}
      </span>
    </button>
  )
}
