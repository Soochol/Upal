// web/src/components/pipelines/PipelineRunHistory.tsx
import { useState, useEffect } from 'react'
import { CheckCircle2, XCircle, Loader2, PauseCircle, Clock } from 'lucide-react'
import { fetchPipelineRuns, approvePipelineStage, rejectPipelineStage } from '@/lib/api'
import type { Pipeline, PipelineRun } from '@/lib/api/types'

type Props = {
  pipeline: Pipeline
}

const statusIcons: Record<string, typeof CheckCircle2> = {
  pending:   Clock,
  running:   Loader2,
  waiting:   PauseCircle,
  completed: CheckCircle2,
  failed:    XCircle,
  skipped:   Clock,
}

const statusColors: Record<string, string> = {
  pending:   'text-muted-foreground',
  running:   'text-info',
  waiting:   'text-warning',
  completed: 'text-success',
  failed:    'text-destructive',
  skipped:   'text-muted-foreground/50',
}

export function PipelineRunHistory({ pipeline }: Props) {
  const [runs, setRuns] = useState<PipelineRun[]>([])
  const [loading, setLoading] = useState(true)

  const reload = async () => {
    try {
      const data = await fetchPipelineRuns(pipeline.id)
      setRuns(data)
    } catch {
      // silent
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { reload() }, [pipeline.id])

  const handleApprove = async (_run: PipelineRun, stageId: string) => {
    await approvePipelineStage(pipeline.id, stageId)
    reload()
  }

  const handleReject = async (_run: PipelineRun, stageId: string) => {
    await rejectPipelineStage(pipeline.id, stageId)
    reload()
  }

  if (loading) {
    return <Loader2 className="h-4 w-4 animate-spin text-muted-foreground mx-auto" />
  }

  if (runs.length === 0) {
    return (
      <p className="text-xs text-muted-foreground text-center py-4">
        No runs yet. Start the pipeline to see history.
      </p>
    )
  }

  return (
    <div className="space-y-2">
      <h3 className="text-sm font-medium">Run History</h3>
      {runs.map((run) => {
        const Icon = statusIcons[run.status] || Clock
        return (
          <div key={run.id} className="border rounded-lg p-3">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <Icon className={`h-3.5 w-3.5 ${statusColors[run.status]} ${run.status === 'running' ? 'animate-spin' : ''}`} />
                <span className="text-xs font-medium">{run.id.slice(0, 12)}</span>
                <span className="text-[10px] text-muted-foreground">
                  {new Date(run.started_at).toLocaleString()}
                </span>
              </div>
              <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${statusColors[run.status]}`}>
                {run.status}
              </span>
            </div>

            <div className="flex items-center gap-1">
              {pipeline.stages.map((stage, i) => {
                const result = run.stage_results?.[stage.id]
                const stageStatus = result?.status || 'pending'
                const StageIcon = statusIcons[stageStatus] || Clock

                return (
                  <div key={stage.id} className="flex items-center gap-1">
                    {i > 0 && <span className="text-border text-xs">→</span>}
                    <div className={`flex items-center gap-1 px-1.5 py-0.5 rounded text-xs ${
                      stageStatus === 'waiting' ? 'bg-warning/10' : 'bg-muted'
                    }`}>
                      <StageIcon className={`h-3 w-3 ${statusColors[stageStatus]} ${stageStatus === 'running' ? 'animate-spin' : ''}`} />
                      <span className="truncate max-w-[80px]">{stage.name || stage.type}</span>
                    </div>
                  </div>
                )
              })}
            </div>

            {run.status === 'waiting' && run.current_stage && (
              <div className="flex items-center gap-2 mt-2 pt-2 border-t">
                <span className="text-xs text-muted-foreground">Awaiting approval:</span>
                <button
                  onClick={() => handleApprove(run, run.current_stage!)}
                  className="px-2 py-0.5 text-xs font-medium rounded bg-success/10 text-success hover:bg-success/20 transition-colors"
                >
                  Approve
                </button>
                <button
                  onClick={() => handleReject(run, run.current_stage!)}
                  className="px-2 py-0.5 text-xs font-medium rounded bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors"
                >
                  Reject
                </button>
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
