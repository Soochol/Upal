import { useState } from 'react'
import { Play, Square, FileText, Trash2 } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { RUN_STATUS_DOT, runDisplayName } from '@/entities/session-run/constants'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { EditableName } from '@/shared/ui/EditableName'
import { ConfirmDialog } from '@/shared/ui/ConfirmDialog'
import type { Run } from '@/entities/session-run'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { cancelRun, collectRun, deleteRun, toggleRunSchedule, updateRunConfig } from '@/entities/session-run/api'

interface RunCardProps {
  run: Run
  isSelected?: boolean
  onSelect: (run: Run) => void
  onDeleted?: () => void
}

export function RunCard({ run, isSelected, onSelect, onDeleted }: RunCardProps) {
  const qc = useQueryClient()
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)

  const toggleMutation = useMutation({
    mutationFn: () => toggleRunSchedule(run.id, !run.schedule_active),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['session-runs'] }),
  })

  const renameMutation = useMutation({
    mutationFn: (name: string) => updateRunConfig(run.id, { name }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['session-runs'] }),
  })

  const collectMutation = useMutation({
    mutationFn: () => collectRun(run.id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['session-runs'] }),
  })

  const cancelMutation = useMutation({
    mutationFn: () => cancelRun(run.id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['session-runs'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: () => deleteRun(run.id),
    onSuccess: () => {
      setShowDeleteDialog(false)
      qc.invalidateQueries({ queryKey: ['session-runs'] })
      onDeleted?.()
    },
  })

  const displayName = run.name || runDisplayName(run)
  const isDraft = run.status === 'draft'
  const isRunning = run.status === 'collecting' || run.status === 'analyzing'
  const hasContent = (run.run_sources?.length ?? 0) > 0 || (run.context?.prompt?.trim() ?? '') !== ''
  const hasSchedule = !!run.schedule

  return (
    <>
      <div
        onClick={() => onSelect(run)}
        className={cn(
          'group flex items-center gap-3 p-4 rounded-xl border bg-card cursor-pointer transition-all',
          isSelected ? 'border-primary/30 bg-primary/5' : 'border-border/50 hover:bg-muted/30',
          run.schedule_active && !isRunning && 'run-card-active',
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
          {isDraft && (
            <button
              onClick={(e) => { e.stopPropagation(); collectMutation.mutate() }}
              disabled={collectMutation.isPending || !hasContent}
              className="p-1.5 rounded-lg text-primary hover:bg-primary/10 transition-colors cursor-pointer disabled:opacity-30 disabled:cursor-not-allowed"
              title={hasContent ? 'Start collection' : 'Add sources or a task prompt first'}
            >
              <Play className="h-3.5 w-3.5" />
            </button>
          )}
          {isRunning && (
            <button
              onClick={(e) => { e.stopPropagation(); cancelMutation.mutate() }}
              disabled={cancelMutation.isPending}
              className="p-1.5 rounded-lg text-destructive hover:bg-destructive/10 transition-colors cursor-pointer disabled:opacity-50"
              title="Stop collection"
            >
              <Square className="h-3.5 w-3.5" />
            </button>
          )}
          {hasSchedule && !isDraft && !isRunning && (
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
            onClick={(e) => { e.stopPropagation(); setShowDeleteDialog(true) }}
            className="p-1.5 rounded-lg text-muted-foreground/0 group-hover:text-muted-foreground/50 hover:!text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
            title="Delete run"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
      <ConfirmDialog
        open={showDeleteDialog}
        onOpenChange={setShowDeleteDialog}
        title="Delete run"
        description={`"${displayName}" and all associated data (sources, analysis, workflow runs) will be permanently deleted.`}
        onConfirm={() => deleteMutation.mutate()}
        isPending={deleteMutation.isPending}
      />
    </>
  )
}
