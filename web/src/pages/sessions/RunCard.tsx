import { Play, Square, Settings, FileText } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { RUN_STATUS_DOT, runDisplayName } from '@/entities/session-run/constants'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { EditableName } from '@/shared/ui/EditableName'
import type { Run } from '@/entities/session-run'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toggleRunSchedule, updateRunConfig } from '@/entities/session-run/api'

interface RunCardProps {
  run: Run
  isSelected?: boolean
  onSelect: (run: Run) => void
  onOpenConfig: (run: Run) => void
}

export function RunCard({ run, isSelected, onSelect, onOpenConfig }: RunCardProps) {
  const qc = useQueryClient()

  const toggleMutation = useMutation({
    mutationFn: () => toggleRunSchedule(run.id, !run.schedule_active),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['session-runs'] }),
  })

  const renameMutation = useMutation({
    mutationFn: (name: string) => updateRunConfig(run.id, { name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['session-runs'] }),
  })

  const displayName = run.name || runDisplayName(run)
  const hasSchedule = !!run.schedule

  return (
    <div
      onClick={() => onSelect(run)}
      className={cn(
        'group flex items-center gap-3 p-4 rounded-xl border bg-card cursor-pointer transition-all',
        isSelected ? 'border-primary/30 bg-primary/5' : 'border-border/50 hover:bg-muted/30',
        run.schedule_active && 'run-card-active',
      )}
    >
      <span className={cn('w-2 h-2 rounded-full shrink-0', RUN_STATUS_DOT[run.status] ?? 'bg-muted-foreground/30')} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
          <EditableName
            value={displayName}
            onSave={(name) => renameMutation.mutate(name)}
            className="text-sm font-semibold"
          />
          <StatusBadge status={run.status} />
        </div>
        <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
          <span>{new Date(run.created_at).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
          {run.source_count != null && run.source_count > 0 && <span>{run.source_count} sources</span>}
          <span className="capitalize">{run.trigger_type}</span>
        </div>
      </div>
      <div className="flex items-center gap-1 shrink-0">
        {hasSchedule && (
          <button
            onClick={(e) => { e.stopPropagation(); toggleMutation.mutate() }}
            className={cn(
              'p-1.5 rounded-lg transition-colors cursor-pointer',
              run.schedule_active
                ? 'text-success hover:bg-success/10'
                : 'text-muted-foreground hover:bg-muted/50',
            )}
            title={run.schedule_active ? 'Stop schedule' : 'Start schedule'}
          >
            {run.schedule_active ? <Square className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
          </button>
        )}
        <button
          onClick={(e) => { e.stopPropagation(); onOpenConfig(run) }}
          className="p-1.5 rounded-lg text-muted-foreground hover:bg-muted/50 transition-colors cursor-pointer opacity-0 group-hover:opacity-100"
          title="Edit config"
        >
          <Settings className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  )
}
