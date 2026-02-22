import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { fetchRuns } from '@/entities/run'
import type { RunRecord } from '@/shared/types'
import { Header } from '@/shared/ui/Header'
import { Clock, CheckCircle2, XCircle, Loader2, Timer } from 'lucide-react'

const statusConfig: Record<string, { icon: typeof Clock; color: string; label: string }> = {
  pending:   { icon: Clock,        color: 'text-warning',          label: 'Pending' },
  running:   { icon: Loader2,      color: 'text-info',             label: 'Running' },
  success:   { icon: CheckCircle2, color: 'text-success',          label: 'Success' },
  failed:    { icon: XCircle,      color: 'text-destructive',      label: 'Failed' },
  cancelled: { icon: XCircle,      color: 'text-muted-foreground', label: 'Cancelled' },
  retrying:  { icon: Timer,        color: 'text-warning',          label: 'Retrying' },
}

export default function Runs() {
  const navigate = useNavigate()
  const [runs, setRuns] = useState<RunRecord[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<string>('all')
  const limit = 20
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const loadRuns = (showSpinner = true) => {
    if (showSpinner) setLoading(true)
    fetchRuns(limit, offset, filter === 'all' ? '' : filter)
      .then(({ runs, total }) => {
        setRuns(runs ?? [])
        setTotal(total)
      })
      .catch(() => setRuns([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadRuns()
  }, [offset, filter]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-refetch when there are running runs
  useEffect(() => {
    const hasRunning = runs.some(r => r.status === 'running' || r.status === 'pending')
    if (hasRunning && !pollRef.current) {
      pollRef.current = setInterval(() => loadRuns(false), 10000)
    } else if (!hasRunning && pollRef.current) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
    return () => {
      if (pollRef.current) {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
    }
  }, [runs]) // eslint-disable-line react-hooks/exhaustive-deps

  // Runs are already filtered server-side; use directly.
  const filtered = runs

  function formatDuration(r: RunRecord): string {
    if (!r.started_at || !r.completed_at) return '-'
    const ms = new Date(r.completed_at).getTime() - new Date(r.started_at).getTime()
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  function formatTime(iso?: string): string {
    if (!iso) return '-'
    return new Date(iso).toLocaleString()
  }

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header />

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-6xl mx-auto px-6 py-6">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-semibold">Run History</h1>
            <span className="text-sm text-muted-foreground">{total} total runs</span>
          </div>

          {/* Filters */}
          <div className="flex gap-2 mb-4">
            {['all', 'success', 'failed', 'running', 'pending'].map(f => (
              <button
                key={f}
                onClick={() => { setFilter(f); setOffset(0) }}
                className={`px-3 py-1 rounded-full text-sm capitalize ${
                  filter === f
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-muted-foreground hover:bg-muted/80'
                }`}
              >
                {f}
              </button>
            ))}
          </div>

          {/* Table */}
          {loading ? (
            <div className="flex items-center justify-center py-20">
              <Loader2 className="animate-spin text-muted-foreground" size={32} />
            </div>
          ) : filtered.length === 0 ? (
            <div className="text-center py-20 text-muted-foreground">
              No runs found
            </div>
          ) : (
            <div className="border border-border rounded-lg overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="bg-muted/50 text-left text-sm text-muted-foreground">
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Workflow</th>
                    <th className="px-4 py-3 font-medium">Trigger</th>
                    <th className="px-4 py-3 font-medium">Duration</th>
                    <th className="px-4 py-3 font-medium">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map(run => {
                    const cfg = statusConfig[run.status] ?? statusConfig.pending
                    const Icon = cfg.icon
                    return (
                      <tr
                        key={run.id}
                        className="border-t border-border hover:bg-muted/30 cursor-pointer transition-colors"
                        onClick={() => navigate(`/runs/${run.id}`)}
                      >
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Icon size={16} className={`${cfg.color} ${run.status === 'running' ? 'animate-spin' : ''}`} />
                            <span className={`text-sm font-medium ${cfg.color}`}>{cfg.label}</span>
                            {run.status === 'running' && (
                              <span className="text-[10px] px-1 py-0.5 rounded-full bg-info/10 text-info animate-pulse">
                                live
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3 text-sm text-foreground font-medium">{run.workflow_name}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground capitalize">{run.trigger_type}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground font-mono">{formatDuration(run)}</td>
                        <td className="px-4 py-3 text-sm text-muted-foreground">{formatTime(run.created_at)}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}

          {/* Pagination */}
          {total > limit && (
            <div className="flex items-center justify-between mt-4">
              <button
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
                className="px-4 py-2 text-sm rounded bg-muted hover:bg-muted/80 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="text-sm text-muted-foreground">
                {offset + 1}-{Math.min(offset + limit, total)} of {total}
              </span>
              <button
                disabled={offset + limit >= total}
                onClick={() => setOffset(offset + limit)}
                className="px-4 py-2 text-sm rounded bg-muted hover:bg-muted/80 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
