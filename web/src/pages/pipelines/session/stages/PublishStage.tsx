import { useState, useMemo } from 'react'
import { CheckCircle, CheckSquare, Square, Loader2 } from 'lucide-react'
import type { ContentSession, WorkflowResult } from '@/entities/content-session'

interface PublishStageProps {
  session: ContentSession
  onPublish: (approvedRunIds: string[]) => void
  onRejectAll: () => void
  isPublishing: boolean
}

export function PublishStage({ session, onPublish, onRejectAll, isPublishing }: PublishStageProps) {
  const successResults = useMemo<WorkflowResult[]>(
    () => (session.workflow_results ?? []).filter((r) => r.status === 'success'),
    [session.workflow_results],
  )

  const [approved, setApproved] = useState<Set<string>>(
    () => new Set(successResults.map((r) => r.run_id)),
  )

  const toggleApproval = (runId: string) => {
    setApproved((prev) => {
      const next = new Set(prev)
      if (next.has(runId)) {
        next.delete(runId)
      } else {
        next.add(runId)
      }
      return next
    })
  }

  const approvedCount = approved.size
  const totalCount = successResults.length

  // Empty state
  if (successResults.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed border-border py-12">
        <CheckSquare className="h-8 w-8 text-muted-foreground/50" />
        <p className="text-sm font-medium text-muted-foreground">No content ready</p>
        <p className="text-xs text-muted-foreground/70">Wait for workflows to complete.</p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Content cards grid */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {successResults.map((result) => {
          const isApproved = approved.has(result.run_id)

          return (
            <div
              key={result.run_id}
              className={`rounded-xl border transition-colors ${
                isApproved
                  ? 'border-success/30 bg-success/5'
                  : 'border-border bg-card'
              }`}
            >
              {/* Card header */}
              <div className="flex items-center gap-2 px-4 py-3 border-b border-border/50">
                <CheckCircle className={`h-4 w-4 shrink-0 ${isApproved ? 'text-success' : 'text-muted-foreground'}`} />
                <span className="text-sm font-medium truncate">{result.workflow_name}</span>
                <span className="ml-auto shrink-0 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium text-muted-foreground">
                  Draft
                </span>
              </div>

              {/* Card body */}
              <div className="px-4 py-6">
                <p className="text-xs text-muted-foreground">
                  Content preview will appear here when workflow output is available.
                </p>
              </div>

              {/* Card footer */}
              <div className="flex items-center justify-end px-4 py-2 border-t border-border/50">
                <button
                  type="button"
                  onClick={() => toggleApproval(result.run_id)}
                  className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors cursor-pointer ${
                    isApproved
                      ? 'text-success hover:bg-success/10'
                      : 'text-muted-foreground hover:bg-muted/50'
                  }`}
                >
                  {isApproved
                    ? <CheckSquare className="h-3.5 w-3.5" />
                    : <Square className="h-3.5 w-3.5" />}
                  {isApproved ? 'Approved' : 'Approve'}
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {/* Bottom actions bar */}
      <div className="flex items-center justify-between rounded-xl border border-border px-4 py-3">
        <span className="text-sm text-muted-foreground">
          {approvedCount} of {totalCount} approved
        </span>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onRejectAll}
            disabled={isPublishing}
            className="rounded-lg px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted/50 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Reject All
          </button>
          <button
            type="button"
            onClick={() => onPublish(Array.from(approved))}
            disabled={approvedCount === 0 || isPublishing}
            className="flex items-center gap-1.5 rounded-lg bg-success px-4 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-success/90 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isPublishing && <Loader2 className="h-3 w-3 animate-spin" />}
            Publish {approvedCount} {approvedCount === 1 ? 'Item' : 'Items'}
          </button>
        </div>
      </div>
    </div>
  )
}
