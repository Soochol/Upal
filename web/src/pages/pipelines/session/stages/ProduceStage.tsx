import { useState } from 'react'
import { Loader2, ExternalLink, CheckCircle2, XCircle, RotateCcw } from 'lucide-react'
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

function StatusArea({ result }: { result: WorkflowResult }) {
  return (
    <div className="h-[80px] rounded-lg bg-muted/30 border border-border p-3 text-xs text-muted-foreground flex items-center justify-center">
      {result.status === 'pending' && (
        <p className="text-muted-foreground/60">Waiting to start...</p>
      )}
      {result.status === 'running' && (
        <div className="flex items-center gap-2 text-info">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span className="font-medium">Executing workflow...</span>
        </div>
      )}
      {result.status === 'success' && (
        <div className="flex flex-col items-center gap-1">
          <div className="flex items-center gap-1.5 text-success">
            <CheckCircle2 className="h-4 w-4 shrink-0" />
            <span className="font-medium">Completed</span>
          </div>
          {result.completed_at && (
            <span className="text-[10px] text-muted-foreground/60">
              {new Date(result.completed_at).toLocaleString()}
            </span>
          )}
        </div>
      )}
      {result.status === 'failed' && (
        <div className="flex items-center gap-1.5 text-destructive">
          <XCircle className="h-4 w-4 shrink-0" />
          <span className="font-medium">Failed</span>
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

      {/* Body */}
      <div className="px-4 py-3">
        <StatusArea result={result} />
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
  onRetry?: () => void
}

export function ProduceStage({ session, onRetry }: ProduceStageProps) {
  const results = session.workflow_results ?? []
  const [retrying, setRetrying] = useState(false)

  const handleRetry = async () => {
    if (!onRetry) return
    setRetrying(true)
    try {
      onRetry()
    } finally {
      setRetrying(false)
    }
  }

  if (results.length === 0) {
    const isError = session.status === 'error'
    const isStuck = (session.status === 'approved' || session.status === 'producing') && !results.length
    const isApproved = session.status === 'approved' || session.status === 'producing'
    const showRetry = (isError || isStuck) && onRetry
    return (
      <div className={`flex flex-col items-center justify-center gap-3 py-12 rounded-xl border-2 border-dashed ${isError ? 'border-destructive/30' : 'border-border'}`}>
        {isError
          ? <XCircle className="h-5 w-5 text-destructive" />
          : <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />}
        <div className="text-center">
          <p className={`text-sm font-medium ${isError ? 'text-destructive' : 'text-muted-foreground'}`}>
            {isError ? 'Production Failed' : isApproved ? 'Starting Workflows...' : 'Waiting for Approval'}
          </p>
          <p className="text-xs text-muted-foreground/70 mt-1">
            {isError
              ? 'All workflows failed to start. Check logs for details.'
              : isApproved
                ? 'Workflows are being prepared. This may take a moment.'
                : 'Approve the analysis to trigger workflow execution.'}
          </p>
        </div>
        {showRetry && (
          <button
            onClick={handleRetry}
            disabled={retrying}
            className="mt-2 inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-medium
              bg-foreground text-background hover:opacity-90 transition-opacity cursor-pointer
              disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {retrying ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RotateCcw className="h-3.5 w-3.5" />}
            Retry Production
          </button>
        )}
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
