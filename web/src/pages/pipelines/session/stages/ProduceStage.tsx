import { Loader2, ExternalLink, CheckCircle2, XCircle } from 'lucide-react'
import type { ContentSession, WorkflowResult } from '@/entities/content-session'

const STATUS_COLOR: Record<string, string> = {
  success: 'text-success bg-success/10 border-success/20',
  failed: 'text-destructive bg-destructive/10 border-destructive/20',
  running: 'text-info bg-info/10 border-info/20',
  pending: 'text-muted-foreground bg-muted/20 border-border',
}

/**
 * Build a collapsed summary string for the produce stage.
 * Example: "2/3 completed, 1 failed"
 */
export function produceSummary(session: ContentSession): string {
  const results = session.workflow_results ?? []
  if (results.length === 0) return 'No workflows'
  const completed = results.filter((r) => r.status === 'success').length
  const failed = results.filter((r) => r.status === 'failed').length
  const parts: string[] = [`${completed}/${results.length} completed`]
  if (failed > 0) parts.push(`${failed} failed`)
  return parts.join(', ')
}

function StatusPill({ status }: { status: WorkflowResult['status'] }) {
  return (
    <span
      className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${STATUS_COLOR[status] ?? STATUS_COLOR.pending}`}
    >
      {status}
    </span>
  )
}

function LogArea({ status }: { status: WorkflowResult['status'] }) {
  return (
    <div className="h-[120px] rounded-lg bg-muted/30 border border-border p-3 font-mono text-xs text-muted-foreground overflow-auto">
      {status === 'pending' && (
        <p className="text-muted-foreground/60">Waiting for agent to initialize...</p>
      )}
      {status === 'running' && (
        <div className="space-y-1.5">
          <p>[system] Initializing workflow runner...</p>
          <p>[system] Building DAG from definition...</p>
          <p>[system] Executing nodes in topological order...</p>
          <div className="flex items-center gap-1.5 text-info">
            <Loader2 className="h-3 w-3 animate-spin" />
            <span>Processing...</span>
          </div>
        </div>
      )}
      {status === 'success' && (
        <div className="flex items-center gap-1.5 text-success">
          <CheckCircle2 className="h-3.5 w-3.5 shrink-0" />
          <span>Completed successfully</span>
        </div>
      )}
      {status === 'failed' && (
        <div className="flex items-center gap-1.5 text-destructive">
          <XCircle className="h-3.5 w-3.5 shrink-0" />
          <span>Execution failed</span>
        </div>
      )}
    </div>
  )
}

function WorkflowCard({ result, index }: { result: WorkflowResult; index: number }) {
  const isRunning = result.status === 'running'

  return (
    <div
      className={`rounded-xl border transition-all ${
        isRunning
          ? 'border-info/50 shadow-[0_0_12px_rgba(59,130,246,0.1)]'
          : 'border-border'
      }`}
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border">
        <span className="flex items-center justify-center w-5 h-5 rounded-full bg-muted text-[10px] font-bold text-muted-foreground shrink-0">
          {index + 1}
        </span>
        <span className="text-sm font-medium truncate flex-1">
          {result.workflow_name}
        </span>
        <StatusPill status={result.status} />
      </div>

      {/* Body — mock log area */}
      <div className="px-4 py-3">
        <LogArea status={result.status} />
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between px-4 py-2.5 border-t border-border">
        <span className="text-[11px] text-muted-foreground font-mono">
          {result.run_id.slice(0, 8)}
        </span>
        {result.status === 'success' && result.output_url ? (
          <a
            href={result.output_url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1 text-xs font-medium text-foreground hover:text-foreground/80 transition-colors"
          >
            Preview
            <ExternalLink className="h-3 w-3" />
          </a>
        ) : (
          <span className="inline-flex items-center gap-1 text-xs text-muted-foreground/50 cursor-not-allowed">
            Preview
            <ExternalLink className="h-3 w-3" />
          </span>
        )}
      </div>
    </div>
  )
}

interface ProduceStageProps {
  session: ContentSession
}

export function ProduceStage({ session }: ProduceStageProps) {
  const results = session.workflow_results ?? []

  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-12 rounded-xl border-2 border-dashed border-border">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
        <div className="text-center">
          <p className="text-sm font-medium text-muted-foreground">
            Waiting for Approval
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            Approve the analysis to trigger workflow execution.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {results.map((result, i) => (
        <WorkflowCard key={result.run_id} result={result} index={i} />
      ))}
    </div>
  )
}
