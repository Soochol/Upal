// web/src/components/pipelines/PipelineRunDetail.tsx
import { useState } from 'react'
import {
  ArrowLeft, CheckCircle2, XCircle, Loader2, PauseCircle,
  Clock, Play, Zap, RefreshCw, GitBranch,
} from 'lucide-react'
import type { Pipeline, PipelineRun, StageResult } from '@/lib/api/types'
import { ContentViewer } from '@/components/ui/ContentViewer'

// ─── Status/type config (mirrors StageCard) ────────────────────────────────

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

const stageTypeConfig: Record<string, { label: string; icon: typeof GitBranch; color: string }> = {
  workflow:  { label: 'Workflow',  icon: Play,        color: 'var(--info)' },
  approval:  { label: 'Approval',  icon: PauseCircle, color: 'var(--warning)' },
  schedule:  { label: 'Schedule',  icon: Clock,       color: 'var(--success)' },
  trigger:   { label: 'Trigger',   icon: Zap,         color: 'var(--node-agent)' },
  transform: { label: 'Transform', icon: RefreshCw,   color: 'var(--muted-foreground)' },
}

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatDuration(startedAt: string, completedAt?: string): string {
  if (!completedAt) return ''
  const ms = new Date(completedAt).getTime() - new Date(startedAt).getTime()
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

/** "output-1" → "Output · 1",  "agent-abc" → "Agent · abc",  else → as-is */
function prettifyNodeId(id: string): string {
  const match = id.match(/^([a-z][a-z0-9]*)[-_](.+)$/)
  if (match) {
    const type = match[1].charAt(0).toUpperCase() + match[1].slice(1)
    return `${type} · ${match[2]}`
  }
  return id
}

// ─── StageOutputPanel ───────────────────────────────────────────────────────

function StageOutputPanel({ result }: { result: StageResult }) {
  const entries = Object.entries(result.output ?? {}).filter(([, v]) => {
    if (v === null || v === undefined) return false
    if (typeof v === 'string' && !v.trim()) return false
    return true
  })

  if (entries.length === 0) return null

  return (
    <div className="space-y-4">
      {entries.map(([key, value]) => (
        <div key={key}>
          <div className="flex items-center gap-1.5 mb-1.5">
            <span className="text-[10px] font-mono text-muted-foreground bg-muted/50 px-1.5 py-0.5 rounded">
              {prettifyNodeId(key)}
            </span>
          </div>
          <ContentViewer value={value} />
        </div>
      ))}
    </div>
  )
}

// ─── PipelineRunDetail ──────────────────────────────────────────────────────

type Props = {
  pipeline: Pipeline
  run: PipelineRun
  onBack: () => void
  onApprove: () => Promise<void>
  onReject: () => Promise<void>
}

export function PipelineRunDetail({ pipeline, run, onBack, onApprove, onReject }: Props) {
  const [acting, setActing] = useState(false)

  const act = (fn: () => Promise<void>) => async () => {
    setActing(true)
    try { await fn() } finally { setActing(false) }
  }

  const RunIcon = statusIcons[run.status] ?? Clock

  return (
    <div className="space-y-5">

      {/* ── Header ── */}
      <div className="flex items-start gap-3">
        <button
          onClick={onBack}
          className="mt-0.5 p-1.5 rounded-md hover:bg-muted transition-colors cursor-pointer shrink-0"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <RunIcon className={`h-4 w-4 ${statusColors[run.status]} ${run.status === 'running' ? 'animate-spin' : ''}`} />
            <span className="text-sm font-semibold">
              Run <span className="font-mono text-muted-foreground">{run.id.slice(0, 16)}</span>
            </span>
            <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium bg-muted ${statusColors[run.status]}`}>
              {run.status}
            </span>
          </div>
          <p className="text-[11px] text-muted-foreground mt-0.5">
            {new Date(run.started_at).toLocaleString()}
            {run.completed_at && ` · ${formatDuration(run.started_at, run.completed_at)}`}
          </p>
        </div>
      </div>

      {/* ── Stage timeline ── */}
      <div className="relative">
        {/* Vertical connector line */}
        {pipeline.stages.length > 1 && (
          <div className="absolute left-[15px] top-5 bottom-5 w-px bg-border/60 pointer-events-none" />
        )}

        <div className="space-y-3">
          {pipeline.stages.map((stage) => {
            const result = run.stage_results?.[stage.id]
            const stageStatus = result?.status ?? 'pending'
            const cfg = stageTypeConfig[stage.type] ?? { label: stage.type, icon: GitBranch, color: 'var(--muted-foreground)' }
            const TypeIcon = cfg.icon
            const StatusIcon = statusIcons[stageStatus] ?? Clock
            const isWaiting = run.status === 'waiting' && run.current_stage === stage.id
            const hasOutput = result?.output && Object.keys(result.output).length > 0

            return (
              <div key={stage.id} className="relative pl-9">

                {/* Timeline dot */}
                <div
                  className="absolute left-2 top-3 w-5 h-5 rounded-full border-2 bg-background flex items-center justify-center z-10"
                  style={{ borderColor: stageStatus === 'pending' ? 'var(--border)' : cfg.color }}
                >
                  <TypeIcon className="h-2.5 w-2.5" style={{ color: stageStatus === 'pending' ? 'var(--muted-foreground)' : cfg.color }} />
                </div>

                {/* Stage card */}
                <div className={[
                  'rounded-xl border bg-card overflow-hidden',
                  isWaiting ? 'ring-2 ring-warning/40 border-warning/30' : 'border-border',
                ].join(' ')}>

                  {/* Card header */}
                  <div className="flex items-center justify-between px-3 py-2.5 border-b border-border/60 bg-muted/10">
                    <div className="flex items-center gap-2 min-w-0">
                      <span
                        className="shrink-0 text-[10px] font-semibold px-1.5 py-0.5 rounded"
                        style={{
                          background: `color-mix(in oklch, ${cfg.color}, transparent 88%)`,
                          color: cfg.color,
                        }}
                      >
                        {cfg.label}
                      </span>
                      <span className="text-sm font-medium truncate">{stage.name || stage.type}</span>
                    </div>

                    <div className="flex items-center gap-2 shrink-0 ml-2">
                      {result?.started_at && (
                        <span className="text-[10px] text-muted-foreground font-mono">
                          {formatDuration(result.started_at, result.completed_at)}
                        </span>
                      )}
                      <div className="flex items-center gap-1">
                        <StatusIcon className={`h-3.5 w-3.5 ${statusColors[stageStatus]} ${stageStatus === 'running' ? 'animate-spin' : ''}`} />
                        <span className={`text-[10px] font-medium ${statusColors[stageStatus]}`}>{stageStatus}</span>
                      </div>
                    </div>
                  </div>

                  {/* Card body */}
                  <div className="px-3 py-3">

                    {/* Error message */}
                    {result?.error && (
                      <div className="flex items-start gap-1.5 text-xs text-destructive mb-2">
                        <XCircle className="h-3.5 w-3.5 mt-px shrink-0" />
                        <span>{result.error}</span>
                      </div>
                    )}

                    {/* Output entries */}
                    {hasOutput && <StageOutputPanel result={result!} />}

                    {/* Empty state for completed stage */}
                    {!hasOutput && !result?.error && result && !['pending', 'running'].includes(stageStatus) && (
                      <p className="text-[11px] text-muted-foreground/40 italic">출력 없음</p>
                    )}

                    {/* Pending placeholder */}
                    {stageStatus === 'pending' && (
                      <p className="text-[11px] text-muted-foreground/40">대기 중</p>
                    )}

                    {/* Approve / Reject */}
                    {isWaiting && (
                      <div className={`flex items-center gap-2 ${(hasOutput || result?.error) ? 'mt-3 pt-3 border-t border-border' : ''}`}>
                        <span className="text-xs text-muted-foreground flex-1">승인 대기 중</span>
                        <button
                          onClick={act(onApprove)}
                          disabled={acting}
                          className="px-3 py-1 text-xs font-medium rounded-lg bg-success/10 text-success
                            hover:bg-success/20 transition-colors disabled:opacity-50 cursor-pointer"
                        >
                          Approve
                        </button>
                        <button
                          onClick={act(onReject)}
                          disabled={acting}
                          className="px-3 py-1 text-xs font-medium rounded-lg bg-destructive/10 text-destructive
                            hover:bg-destructive/20 transition-colors disabled:opacity-50 cursor-pointer"
                        >
                          Reject
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
