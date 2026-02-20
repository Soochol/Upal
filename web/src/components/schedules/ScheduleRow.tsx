import { Pause, Play, Pencil, Trash2 } from 'lucide-react'
import type { Schedule } from '@/lib/api'

type Props = {
  schedule: Schedule
  onPause: (id: string) => void
  onResume: (id: string) => void
  onEdit: (schedule: Schedule) => void
  onDelete: (id: string) => void
}

function cronToHuman(expr: string): string {
  const presets: Record<string, string> = {
    '0 * * * *': 'Every hour',
    '0 */6 * * *': 'Every 6 hours',
    '0 0 * * *': 'Daily at midnight',
    '0 9 * * 1': 'Weekly Mon 9am',
  }
  return presets[expr] ?? expr
}

function formatDate(iso?: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function ScheduleRow({ schedule, onPause, onResume, onEdit, onDelete }: Props) {
  return (
    <div data-testid={`schedule-row-${schedule.id}`} className="flex items-center justify-between py-3 px-4 border-b border-border last:border-0 hover:bg-muted/30 transition-colors">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-sm font-medium text-foreground truncate">
            {schedule.workflow_name}
          </span>
          <code className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded font-mono">
            {schedule.cron_expr}
          </code>
          <span
            className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${
              schedule.enabled
                ? 'bg-green-500/10 text-green-500'
                : 'bg-muted text-muted-foreground'
            }`}
          >
            {schedule.enabled ? 'active' : 'paused'}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span>{cronToHuman(schedule.cron_expr)}</span>
          {schedule.next_run_at && (
            <span>next: {formatDate(schedule.next_run_at)}</span>
          )}
          {schedule.last_run_at && (
            <span>last: {formatDate(schedule.last_run_at)}</span>
          )}
          {schedule.timezone !== 'UTC' && <span>{schedule.timezone}</span>}
        </div>
      </div>

      <div className="flex items-center gap-1 ml-3">
        <button
          onClick={() => (schedule.enabled ? onPause(schedule.id) : onResume(schedule.id))}
          className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          title={schedule.enabled ? 'Pause' : 'Resume'}
        >
          {schedule.enabled ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
        </button>
        <button
          onClick={() => onEdit(schedule)}
          className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          title="Edit"
        >
          <Pencil className="h-3.5 w-3.5" />
        </button>
        <button
          onClick={() => onDelete(schedule.id)}
          className="p-1.5 rounded hover:bg-muted text-muted-foreground hover:text-destructive transition-colors"
          title="Delete"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  )
}
