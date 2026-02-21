import { useState } from 'react'
import type { NodeRunRecord, RunRecord } from '@/lib/api'

type Props = {
  nodeRun: NodeRunRecord
  run: RunRecord
}

type Tab = 'logs' | 'output' | 'error'

export function NodeDetail({ nodeRun, run }: Props) {
  const [activeTab, setActiveTab] = useState<Tab>(nodeRun.error ? 'error' : 'output')

  const hasError = !!nodeRun.error
  const output = run.outputs?.[nodeRun.node_id]
  const input = run.inputs?.[nodeRun.node_id]

  const tabs: { key: Tab; label: string; show: boolean }[] = [
    { key: 'output', label: 'Output', show: true },
    { key: 'error', label: 'Error', show: hasError },
    { key: 'logs', label: 'Info', show: true },
  ]

  function formatDuration(): string {
    if (!nodeRun.started_at || !nodeRun.completed_at) return '-'
    const ms = new Date(nodeRun.completed_at).getTime() - new Date(nodeRun.started_at).getTime()
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  return (
    <div className="border border-border rounded-lg bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-border bg-muted/30">
        <div className="flex items-center gap-2">
          <code className="text-sm font-mono font-medium text-foreground">{nodeRun.node_id}</code>
          <span className="text-xs text-muted-foreground capitalize">{nodeRun.status}</span>
          <span className="text-xs text-muted-foreground font-mono">{formatDuration()}</span>
        </div>
        {nodeRun.retry_count > 0 && (
          <span className="text-xs text-warning">retries: {nodeRun.retry_count}</span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-border">
        {tabs.filter(t => t.show).map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            className={`px-4 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              activeTab === tab.key
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="p-4 max-h-[300px] overflow-y-auto">
        {activeTab === 'output' && (
          <div className="space-y-3">
            {output != null ? (
              <pre className="text-xs font-mono text-foreground bg-muted p-3 rounded overflow-auto whitespace-pre-wrap">
                {typeof output === 'string' ? output : JSON.stringify(output, null, 2)}
              </pre>
            ) : (
              <p className="text-xs text-muted-foreground">No output recorded</p>
            )}
            {input != null && (
              <div>
                <p className="text-[10px] text-muted-foreground uppercase tracking-wider mb-1">Input</p>
                <pre className="text-xs font-mono text-muted-foreground bg-muted p-3 rounded overflow-auto whitespace-pre-wrap">
                  {typeof input === 'string' ? input : JSON.stringify(input, null, 2)}
                </pre>
              </div>
            )}
          </div>
        )}

        {activeTab === 'error' && nodeRun.error && (
          <div className="bg-destructive/10 border border-destructive/20 rounded p-3">
            <pre className="text-xs text-destructive font-mono whitespace-pre-wrap">{nodeRun.error}</pre>
          </div>
        )}

        {activeTab === 'logs' && (
          <div className="space-y-2 text-xs">
            <div className="flex gap-4">
              <span className="text-muted-foreground">Started</span>
              <span className="text-foreground">
                {nodeRun.started_at ? new Date(nodeRun.started_at).toLocaleString() : '-'}
              </span>
            </div>
            <div className="flex gap-4">
              <span className="text-muted-foreground">Completed</span>
              <span className="text-foreground">
                {nodeRun.completed_at ? new Date(nodeRun.completed_at).toLocaleString() : '-'}
              </span>
            </div>
            <div className="flex gap-4">
              <span className="text-muted-foreground">Duration</span>
              <span className="text-foreground font-mono">{formatDuration()}</span>
            </div>
            <div className="flex gap-4">
              <span className="text-muted-foreground">Status</span>
              <span className="text-foreground capitalize">{nodeRun.status}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
