import { useState, useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  X, ExternalLink, CheckCircle, XCircle, CheckSquare, Square, Loader2,
} from 'lucide-react'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ScoreIndicator } from '@/shared/ui/ScoreIndicator'
import { SourceTypeBadge } from '@/shared/ui/SourceTypeBadge'
import { fetchContentSession } from '@/entities/content-session/api'
import { useContentSessionStore } from '@/entities/content-session/store'
import type { ContentSession, ContentAngle } from '@/entities/content-session'

// ─── Source panel ─────────────────────────────────────────────────────────────

function SourcePanel({ session }: { session: ContentSession }) {
  const isCollecting = session.status === 'collecting'
  const sources = session.sources ?? []
  return (
    <div className="space-y-6">
      {isCollecting && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Collecting sources...
        </div>
      )}
      {sources.map((src) => (
        <div key={src.id}>
          <div className="flex items-center gap-2 mb-2">
            <SourceTypeBadge type={src.source_type} />
            <span className="text-sm font-medium">{src.label}</span>
            <span className="text-xs text-muted-foreground">· {src.count} items</span>
          </div>
          <div className="rounded-xl border border-border overflow-hidden">
            {src.items.map((item, i) => (
              <div
                key={i}
                className="flex items-center gap-3 px-4 py-2.5 border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors"
              >
                {item.score != null && (
                  <span className="text-xs text-muted-foreground tabular-nums w-16 shrink-0">
                    {item.extra ?? item.score}
                  </span>
                )}
                <span className="text-sm flex-1 truncate">{item.title}</span>
                {item.url && (
                  <a href={item.url} target="_blank" rel="noopener noreferrer"
                    className="text-muted-foreground hover:text-foreground transition-colors shrink-0">
                    <ExternalLink className="h-3.5 w-3.5" />
                  </a>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
      {sources.length === 0 && !isCollecting && (
        <p className="text-sm text-muted-foreground">No sources collected.</p>
      )}
    </div>
  )
}

// ─── Analysis panel ───────────────────────────────────────────────────────────

function AnalysisPanel({
  session, angles, onToggleAngle,
}: {
  session: ContentSession
  angles: ContentAngle[]
  onToggleAngle: (id: string) => void
}) {
  if (!session.analysis) return <p className="text-sm text-muted-foreground">No analysis results.</p>
  const { analysis } = session
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Total Collected: <span className="text-foreground">{analysis.total_collected}</span>
          {' → '}Selected: <span className="text-foreground">{analysis.total_selected}</span>
        </p>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 h-[500px]">
        <div className="flex flex-col rounded-xl border border-border bg-card overflow-hidden shadow-sm">
          <div className="bg-muted/50 border-b border-border px-4 py-2 flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">Original Sources (Raw)</span>
            <span className="text-[10px] bg-background px-2 py-0.5 rounded-md border border-border text-muted-foreground">Read-only</span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 text-sm text-muted-foreground leading-relaxed font-mono whitespace-pre-wrap">
            {(session.sources ?? []).flatMap(s => s.items).slice(0, 5).map((item, idx) => (
              `<article id="${idx}">\n  <title>${item.title}</title>\n  <source>${item.url || 'unknown'}</source>\n</article>\n\n`
            ))}
          </div>
        </div>
        <div className="flex flex-col rounded-xl border border-success/30 bg-success/5 overflow-hidden shadow-sm">
          <div className="bg-success/10 border-b border-success/20 px-4 py-2 flex items-center justify-between">
            <span className="text-xs font-medium text-success-foreground">AI Generated Summary &amp; Insights</span>
            <span className="flex h-2 w-2">
              <span className="animate-ping absolute inline-flex h-2 w-2 rounded-full bg-success opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-success" />
            </span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-5">
            <div>
              <h3 className="text-xs font-semibold text-success-foreground mb-1.5 uppercase tracking-wide">Summary</h3>
              <div className="rounded-lg border border-success/20 bg-background/50 px-3 py-2 text-sm leading-relaxed text-foreground focus-within:ring-2 focus-within:ring-success/50 outline-none" contentEditable suppressContentEditableWarning>
                {analysis.summary}
              </div>
            </div>
            <div>
              <h3 className="text-xs font-semibold text-success-foreground mb-1.5 uppercase tracking-wide">Key Insights</h3>
              <ul className="space-y-1.5">
                {analysis.insights.map((insight, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm">
                    <span className="text-success mt-0.5 shrink-0">•</span>
                    <div className="flex-1 outline-none text-foreground/90 focus-within:text-foreground" contentEditable suppressContentEditableWarning>{insight}</div>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      </div>
      <div>
        <h3 className="text-sm font-semibold mb-2 mt-4 ml-1">Recommended Workflows (Content Formats)</h3>
        <div className="rounded-xl border border-border overflow-hidden bg-card">
          {angles.map((angle) => (
            <button key={angle.id} onClick={() => onToggleAngle(angle.id)}
              className={`w-full flex items-center gap-3 px-4 py-3 border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors text-left cursor-pointer ${angle.selected ? 'bg-primary/5' : ''}`}
            >
              {angle.selected ? <CheckSquare className="h-4 w-4 text-primary shrink-0" /> : <Square className="h-4 w-4 text-muted-foreground shrink-0" />}
              <span className="text-xs font-mono text-muted-foreground w-12 shrink-0">{angle.format}</span>
              <span className={`text-sm ${angle.selected ? 'text-foreground font-medium' : 'text-foreground/80'}`}>{angle.title}</span>
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ─── Workflow execution panel ─────────────────────────────────────────────────

function WorkflowExecutionPanel({ session }: { session: ContentSession }) {
  const results = session.workflow_results ?? []
  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center border-2 border-dashed border-border rounded-2xl">
        <Loader2 className="h-8 w-8 text-muted-foreground/50 mb-4 animate-spin" />
        <h3 className="text-sm font-semibold text-foreground">Waiting for Workflows</h3>
        <p className="text-xs text-muted-foreground mt-1">Accept the analysis to trigger parallel workflows.</p>
      </div>
    )
  }
  const statusLabel: Record<string, string> = { pending: 'Pending', running: 'Running...', success: 'Completed', failed: 'Failed' }
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-2">
        <h3 className="text-sm font-semibold">Workflow Execution Monitor <span className="text-muted-foreground font-normal ml-2">({results.length} active)</span></h3>
        <button className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors flex items-center gap-1">
          <ExternalLink className="w-3.5 h-3.5" /> Open Global Logs
        </button>
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {results.map((result, idx) => {
          const isDone = result.status === 'success' || result.status === 'failed'
          const statusColor = result.status === 'success' ? 'text-success bg-success/10 border-success/20'
            : result.status === 'failed' ? 'text-destructive bg-destructive/10 border-destructive/20'
            : result.status === 'running' ? 'text-info bg-info/10 border-info/20'
            : 'text-muted-foreground bg-muted/20 border-border'
          const borderHighlight = result.status === 'running' ? 'border-info/50 shadow-[0_0_15px_rgba(59,130,246,0.15)]' : 'border-border/50'
          return (
            <div key={result.run_id} className={`flex flex-col rounded-xl border bg-card overflow-hidden shadow-sm transition-all ${borderHighlight}`}>
              <div className="bg-muted/30 border-b border-border px-4 py-2.5 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="size-5 rounded-md bg-background flex items-center justify-center shadow-sm border border-black/5">
                    <span className="text-[10px] font-bold text-foreground">{idx + 1}</span>
                  </div>
                  <span className="text-sm font-bold tracking-tight text-foreground truncate max-w-[150px]">{result.workflow_name}</span>
                </div>
                <div className={`px-2 py-0.5 rounded-md border text-[10px] font-bold uppercase tracking-wider flex items-center gap-1.5 ${statusColor}`}>
                  {result.status === 'running' && <div className="w-1.5 h-1.5 rounded-full bg-info animate-pulse" />}
                  {statusLabel[result.status] ?? result.status}
                </div>
              </div>
              <div className="flex-1 flex flex-col p-3 bg-background/50 space-y-3">
                <div className="flex-1 rounded-lg border border-black/5 bg-background shadow-inner p-3 font-mono text-xs overflow-y-auto h-[120px]">
                  {result.status === 'pending' && <span className="text-muted-foreground">Waiting for agent to initialize...</span>}
                  {result.status === 'running' && (
                    <div className="space-y-1.5">
                      <div className="text-muted-foreground">{"[System]"} Injecting context parameters... <span className="text-success">OK</span></div>
                      <div className="text-muted-foreground">{"[Agent]"} Planning execution steps... <span className="text-success">OK</span></div>
                      <div className="text-foreground flex items-center gap-2"><Loader2 className="w-3 h-3 animate-spin text-info" /> Generating content formatting...</div>
                    </div>
                  )}
                  {isDone && (
                    <div className="space-y-1">
                      <div className="text-success">Execution finished successfully in 4.2s.</div>
                      <div className="text-muted-foreground mt-2">Tokens used: 1,240 input / 450 output</div>
                    </div>
                  )}
                </div>
                <div className="flex items-center justify-between pt-1">
                  <span className="text-[10px] text-muted-foreground font-mono">Run ID: {result.run_id.slice(0, 8)}</span>
                  {result.output_url && result.status === 'success' ? (
                    <button className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-foreground text-background hover:opacity-90 transition-opacity">Preview Output</button>
                  ) : (
                    <button disabled className="text-xs font-semibold px-3 py-1.5 rounded-lg border border-border text-muted-foreground opacity-50 cursor-not-allowed">Preview Output</button>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ─── Publish panel ────────────────────────────────────────────────────────────

function PublishPanel({ session }: { session: ContentSession }) {
  const results = session.workflow_results ?? []
  const hasSuccess = results.some(r => r.status === 'success')
  if (!hasSuccess) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center border border-border bg-card rounded-2xl">
        <CheckSquare className="w-12 h-12 text-muted-foreground/30 mb-4" />
        <h3 className="text-lg font-semibold mb-2 text-muted-foreground">No content ready for publishing</h3>
        <p className="text-sm text-muted-foreground/80">Wait for the workflows to complete successfully.</p>
      </div>
    )
  }
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-sm font-semibold">Dry Run / Preview</h3>
          <p className="text-xs text-muted-foreground mt-0.5">Review the generated content before final dispatch.</p>
        </div>
        <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-success/15 text-success hover:bg-success/20 transition-colors cursor-pointer text-xs font-semibold">
          <CheckCircle className="h-3.5 w-3.5" /> Publish All Approvals
        </button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {results.filter(r => r.status === 'success').map((result) => (
          <div key={result.run_id} className="rounded-xl border border-border bg-card overflow-hidden shadow-sm flex flex-col">
            <div className="bg-muted/30 border-b border-border px-4 py-3 flex items-center justify-between">
              <span className="text-sm font-bold text-foreground">{result.workflow_name}</span>
              <span className="text-[10px] bg-background px-2 py-0.5 rounded-md border text-muted-foreground">Draft Mode</span>
            </div>
            <div className="p-4 flex-1">
              <div className="flex items-start gap-3 mb-3">
                <div className="w-10 h-10 rounded-full bg-muted animate-pulse shrink-0" />
                <div className="space-y-1.5 flex-1">
                  <div className="h-3.5 bg-muted rounded animate-pulse w-1/3" />
                  <div className="h-2.5 bg-muted rounded animate-pulse w-1/4" />
                </div>
              </div>
              <div className="space-y-2 mb-4">
                <div className="h-3 bg-muted rounded animate-pulse w-full" />
                <div className="h-3 bg-muted rounded animate-pulse w-full" />
                <div className="h-3 bg-muted rounded animate-pulse w-5/6" />
                <div className="h-3 bg-muted rounded animate-pulse w-4/6" />
              </div>
              <div className="rounded-lg aspect-video bg-muted/50 border border-border flex items-center justify-center">
                <span className="text-xs font-mono text-muted-foreground/50">Generated Media / Link Card Preview</span>
              </div>
            </div>
            <div className="border-t border-border bg-muted/10 p-3 flex gap-2">
              <button className="flex-1 text-xs font-medium py-1.5 rounded-lg border border-border hover:bg-muted transition-colors">Edit Content</button>
              <button className="flex-[2] text-xs font-medium py-1.5 rounded-lg bg-foreground text-background hover:opacity-90 transition-opacity">Schedule Post</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

// ─── Reject dialog ────────────────────────────────────────────────────────────

function RejectDialog({ onConfirm, onCancel, isLoading }: { onConfirm: (reason: string) => void; onCancel: () => void; isLoading: boolean }) {
  const [reason, setReason] = useState('')
  return (
    <div className="fixed inset-0 z-[60] flex items-end sm:items-center justify-center">
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative bg-card border border-border rounded-2xl p-5 w-full max-w-sm mx-4 shadow-xl">
        <h3 className="text-sm font-semibold mb-1">Reject Session</h3>
        <p className="text-xs text-muted-foreground mb-3">Enter a reason (optional)</p>
        <textarea value={reason} onChange={(e) => setReason(e.target.value)}
          placeholder="e.g. Low relevance, reschedule for next week..." rows={3}
          className="w-full rounded-xl border border-border bg-muted/20 px-3 py-2 text-sm placeholder:text-muted-foreground/60 focus:outline-none focus:ring-1 focus:ring-border resize-none mb-3"
        />
        <div className="flex items-center justify-end gap-2">
          <button onClick={onCancel} className="px-3 py-1.5 rounded-lg text-sm text-muted-foreground hover:text-foreground transition-colors cursor-pointer">Cancel</button>
          <button onClick={() => onConfirm(reason)} disabled={isLoading}
            className="flex items-center gap-1.5 px-4 py-1.5 rounded-xl text-sm font-medium bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed">
            {isLoading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
            Confirm Reject
          </button>
        </div>
      </div>
    </div>
  )
}

// ─── Modal ────────────────────────────────────────────────────────────────────

type Stage = 'collect' | 'analyze' | 'workflow' | 'publish'

interface SessionDetailModalProps {
  sessionId: string
  pipelineId: string
  onClose: () => void
}

export function SessionDetailModal({ sessionId, pipelineId, onClose }: SessionDetailModalProps) {
  const queryClient = useQueryClient()
  const [activeStage, setActiveStage] = useState<Stage>('collect')
  const [showRejectDialog, setShowRejectDialog] = useState(false)
  const [angles, setAngles] = useState<ContentAngle[]>([])
  const [isApproving, setIsApproving] = useState(false)
  const [isRejecting, setIsRejecting] = useState(false)

  const { data: session, isLoading } = useQuery({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId),
    enabled: !!sessionId,
    refetchInterval: (query) =>
      query.state.data?.status === 'collecting' ? 3000 : false,
  })

  useEffect(() => {
    if (session?.analysis?.angles) {
      setAngles(session.analysis.angles)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.id])

  // Close on Escape key
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const { approveSession, rejectSession } = useContentSessionStore()

  const handleApprove = async () => {
    setIsApproving(true)
    try {
      await approveSession(sessionId, angles.filter((a) => a.selected).map((a) => a.id))
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      onClose()
    } finally {
      setIsApproving(false)
    }
  }

  const handleReject = async (reason: string) => {
    setIsRejecting(true)
    try {
      await rejectSession(sessionId, reason || undefined)
      queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
      onClose()
    } finally {
      setIsRejecting(false)
    }
  }

  const selectedCount = angles.filter((a) => a.selected).length
  const isPending = session?.status === 'pending_review'

  const stages: { value: Stage; label: string }[] = [
    { value: 'collect', label: '1. Collect' },
    { value: 'analyze', label: '2. AI Analyze' },
    { value: 'workflow', label: '3. Workflow' },
    { value: 'publish', label: '4. Publish' },
  ]

  const createdAt = session
    ? new Date(session.created_at).toLocaleString('en-US', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
      })
    : ''

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      {/* Modal panel */}
      <div className="relative z-10 bg-background border border-border rounded-2xl shadow-2xl
        w-full max-w-5xl mx-4 max-h-[90vh] overflow-hidden flex flex-col
        animate-in fade-in slide-in-from-bottom-4 duration-300">

        {/* Header */}
        <div className="flex items-start justify-between gap-4 px-6 py-4 border-b border-border shrink-0">
          {isLoading || !session ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="text-sm">Loading session...</span>
            </div>
          ) : (
            <div>
              <div className="flex items-center gap-2 flex-wrap mb-0.5">
                <h2 className="text-base font-semibold">
                  {session.session_number != null ? `Session ${session.session_number}` : session.id}
                </h2>
                <StatusBadge status={session.status} />
                {session.analysis && <ScoreIndicator score={session.analysis.score} />}
              </div>
              <p className="text-xs text-muted-foreground">
                {createdAt} · trigger: {session.trigger_type}
              </p>
            </div>
          )}
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer shrink-0"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {session && (
          <>
            {/* Stage stepper */}
            <div className="px-6 pt-4 pb-2 border-b border-border shrink-0">
              <div className="flex items-center gap-2 relative">
                <div className="absolute left-[10%] right-[10%] top-1/2 -translate-y-1/2 h-0.5 bg-border/40 z-0" />
                {stages.map((stage, idx) => {
                  const isActive = activeStage === stage.value
                  const isPast = stages.findIndex(s => s.value === activeStage) > idx
                  return (
                    <button
                      key={stage.value}
                      onClick={() => setActiveStage(stage.value)}
                      className={`relative z-10 flex-1 flex flex-col items-center justify-center p-2.5 rounded-xl border transition-all duration-300 cursor-pointer text-sm font-bold tracking-tight
                        ${isActive
                          ? 'bg-foreground border-foreground text-background shadow-lg shadow-black/20 scale-105'
                          : isPast
                          ? 'glass-panel border-success/30 text-success'
                          : 'glass-panel border-white/5 text-muted-foreground hover:bg-white/5'
                        }`}
                    >
                      {stage.label}
                      {isActive && <div className="absolute -bottom-1.5 w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />}
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Stage content */}
            <div className="flex-1 overflow-y-auto px-6 py-5 animate-in fade-in slide-in-from-bottom-2 duration-300">
              {activeStage === 'collect' && <SourcePanel session={session} />}
              {activeStage === 'analyze' && (
                <AnalysisPanel session={session} angles={angles} onToggleAngle={(id) => setAngles(prev => prev.map(a => a.id === id ? { ...a, selected: !a.selected } : a))} />
              )}
              {activeStage === 'workflow' && <WorkflowExecutionPanel session={session} />}
              {activeStage === 'publish' && <PublishPanel session={session} />}
            </div>

            {/* Bottom action bar */}
            {isPending && (
              <div className="border-t border-border bg-background/95 backdrop-blur-sm px-6 py-3 shrink-0 flex items-center justify-between gap-4">
                <p className="text-sm text-muted-foreground">{selectedCount} format(s) selected</p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setShowRejectDialog(true)}
                    disabled={isRejecting}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <XCircle className="h-4 w-4" /> Reject
                  </button>
                  <button
                    onClick={() => void handleApprove()}
                    disabled={selectedCount === 0 || isApproving}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-xl text-sm font-medium bg-success/15 text-success hover:bg-success/25 transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {isApproving ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}
                    Approve &amp; Run Selected Workflows
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {showRejectDialog && (
        <RejectDialog
          onConfirm={(reason) => void handleReject(reason)}
          onCancel={() => setShowRejectDialog(false)}
          isLoading={isRejecting}
        />
      )}
    </div>
  )
}
