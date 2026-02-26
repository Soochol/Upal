import { useQuery } from '@tanstack/react-query'
import { Loader2, FileText, Play } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { fetchSessionRuns } from '@/entities/session-run'
import { RUN_STATUS_DOT, runDisplayName } from '@/entities/session-run/constants'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import type { Run } from '@/entities/session-run'

interface RunHistoryPanelProps {
  sessionId: string
}

export function RunHistoryPanel({ sessionId }: RunHistoryPanelProps) {
  const { data: runs = [], isLoading } = useQuery({
    queryKey: ['session-runs', sessionId],
    queryFn: () => fetchSessionRuns(sessionId),
    enabled: !!sessionId,
    refetchInterval: 10000,
  })

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground gap-3">
        <Loader2 className="w-5 h-5 animate-spin text-primary/50" />
        <p className="text-sm">Loading runs...</p>
      </div>
    )
  }

  if (runs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3 text-center">
        <Play className="w-8 h-8 opacity-20" />
        <div>
          <p className="font-medium text-foreground">No runs yet</p>
          <p className="text-xs mt-1">Runs will appear here when this session is triggered.</p>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-6 space-y-2">
      {runs.map((run: Run) => (
        <div
          key={run.id}
          className="flex items-center gap-3 p-4 rounded-xl border border-border/50 bg-card hover:bg-muted/30 transition-colors"
        >
          <span className={cn('w-2 h-2 rounded-full shrink-0', RUN_STATUS_DOT[run.status] ?? 'bg-muted-foreground/30')} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <FileText className="h-3.5 w-3.5 shrink-0 text-muted-foreground/60" />
              <span className="text-sm font-semibold text-foreground">{runDisplayName(run)}</span>
              <StatusBadge status={run.status} />
            </div>
            <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
              <span>{new Date(run.created_at).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })}</span>
              {run.source_count != null && run.source_count > 0 && <span>{run.source_count} sources</span>}
              <span className="capitalize">{run.trigger_type}</span>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
