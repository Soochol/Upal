import { useEffect, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { fetchRun, type RunRecord } from '@/lib/api'
import { usePolling } from '@/hooks/usePolling'
import { NodeTimeline } from '@/components/runs/NodeTimeline'
import { NodeDetail } from '@/components/runs/NodeDetail'
import { Header } from '@/components/Header'
import {
  ArrowLeft, CheckCircle2, XCircle, Clock, Loader2, Timer,
  ExternalLink, RotateCcw,
} from 'lucide-react'

const statusConfig: Record<string, { icon: typeof Clock; color: string; label: string }> = {
  pending:   { icon: Clock,        color: 'text-warning',          label: 'Pending' },
  running:   { icon: Loader2,      color: 'text-info',             label: 'Running' },
  success:   { icon: CheckCircle2, color: 'text-success',          label: 'Success' },
  failed:    { icon: XCircle,      color: 'text-destructive',      label: 'Failed' },
  cancelled: { icon: XCircle,      color: 'text-muted-foreground', label: 'Cancelled' },
  retrying:  { icon: Timer,        color: 'text-warning',          label: 'Retrying' },
  completed: { icon: CheckCircle2, color: 'text-success',          label: 'Completed' },
  error:     { icon: XCircle,      color: 'text-destructive',      label: 'Error' },
}

export function RunDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [run, setRun] = useState<RunRecord | null>(null)
  const [loading, setLoading] = useState(true)
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null)

  // Initial fetch
  useEffect(() => {
    if (!id) return
    fetchRun(id)
      .then((r) => {
        setRun(r)
        if (r.node_runs && r.node_runs.length > 0) {
          setSelectedNodeId(r.node_runs[0].node_id)
        }
      })
      .catch(() => setRun(null))
      .finally(() => setLoading(false))
  }, [id])

  // Poll while running
  const isRunning = run?.status === 'running' || run?.status === 'pending'
  const { data: polledRun } = usePolling(
    () => fetchRun(id!),
    3000,
    !!id && isRunning,
  )

  useEffect(() => {
    if (polledRun) setRun(polledRun)
  }, [polledRun])

  if (loading) {
    return (
      <div className="h-screen flex flex-col bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="animate-spin text-muted-foreground" size={32} />
        </div>
      </div>
    )
  }

  if (!run) {
    return (
      <div className="h-screen flex flex-col bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <p className="text-muted-foreground">Run not found</p>
        </div>
      </div>
    )
  }

  const cfg = statusConfig[run.status] ?? statusConfig.pending
  const Icon = cfg.icon

  function formatTime(iso?: string): string {
    if (!iso) return '-'
    return new Date(iso).toLocaleString()
  }

  function formatDuration(): string {
    if (!run?.started_at) return '-'
    const end = run.completed_at ? new Date(run.completed_at) : new Date()
    const ms = end.getTime() - new Date(run.started_at).getTime()
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  const selectedNodeRun = selectedNodeId
    ? run.node_runs?.find((nr) => nr.node_id === selectedNodeId)
    : null

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header />

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-4xl mx-auto px-6 py-6 space-y-5">
          {/* Header row */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate('/runs')}
                className="text-muted-foreground hover:text-foreground transition-colors"
              >
                <ArrowLeft size={18} />
              </button>
              <code className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded font-mono">
                {run.id.length > 12 ? `${run.id.slice(0, 12)}...` : run.id}
              </code>
              <div className="flex items-center gap-1.5">
                <Icon
                  size={18}
                  className={`${cfg.color} ${run.status === 'running' ? 'animate-spin' : ''}`}
                />
                <span className={`text-sm font-semibold ${cfg.color}`}>{cfg.label}</span>
              </div>
              {isRunning && (
                <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-info/10 text-info animate-pulse">
                  live
                </span>
              )}
            </div>
            <div className="flex items-center gap-2">
              {run.status === 'failed' && (
                <button className="flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors">
                  <RotateCcw className="h-3 w-3" />
                  Retry
                </button>
              )}
              <Link
                to={`/editor?name=${encodeURIComponent(run.workflow_name)}`}
                className="flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
              >
                <ExternalLink className="h-3 w-3" />
                Open Workflow
              </Link>
            </div>
          </div>

          {/* Summary cards */}
          <div className="grid grid-cols-4 gap-3">
            {[
              { label: 'Workflow', value: run.workflow_name },
              { label: 'Trigger', value: run.trigger_type, capitalize: true },
              { label: 'Duration', value: formatDuration(), mono: true },
              { label: 'Retries', value: String(run.retry_count), mono: true },
            ].map((card) => (
              <div key={card.label} className="border border-border rounded-lg p-3 bg-card">
                <p className="text-[10px] text-muted-foreground uppercase tracking-wider mb-0.5">{card.label}</p>
                <p className={`text-sm font-medium text-foreground ${card.mono ? 'font-mono' : ''} ${card.capitalize ? 'capitalize' : ''}`}>
                  {card.value}
                </p>
              </div>
            ))}
          </div>

          {/* Error banner */}
          {run.error && (
            <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-3">
              <p className="text-sm text-destructive font-mono">{run.error}</p>
            </div>
          )}

          {/* Time info */}
          <div className="flex gap-6 text-xs text-muted-foreground">
            <span>Started: {formatTime(run.started_at)}</span>
            <span>Completed: {formatTime(run.completed_at)}</span>
          </div>

          {/* Node Timeline */}
          {run.node_runs && run.node_runs.length > 0 && (
            <NodeTimeline
              nodeRuns={run.node_runs}
              selectedNodeId={selectedNodeId}
              onSelectNode={setSelectedNodeId}
            />
          )}

          {/* Node Detail */}
          {selectedNodeRun && (
            <NodeDetail nodeRun={selectedNodeRun} run={run} />
          )}

          {/* Inputs / Outputs (collapsible) */}
          {run.inputs && Object.keys(run.inputs).length > 0 && (
            <details className="border border-border rounded-lg bg-card">
              <summary className="px-4 py-3 text-sm font-medium text-foreground cursor-pointer hover:bg-muted/30">
                Inputs
              </summary>
              <div className="px-4 pb-4">
                <pre className="text-xs text-foreground bg-muted p-3 rounded overflow-auto max-h-48 font-mono">
                  {JSON.stringify(run.inputs, null, 2)}
                </pre>
              </div>
            </details>
          )}

          {run.outputs && Object.keys(run.outputs).length > 0 && (
            <details className="border border-border rounded-lg bg-card">
              <summary className="px-4 py-3 text-sm font-medium text-foreground cursor-pointer hover:bg-muted/30">
                Outputs
              </summary>
              <div className="px-4 pb-4">
                <pre className="text-xs text-foreground bg-muted p-3 rounded overflow-auto max-h-48 font-mono">
                  {JSON.stringify(run.outputs, null, 2)}
                </pre>
              </div>
            </details>
          )}
        </div>
      </div>
    </div>
  )
}
