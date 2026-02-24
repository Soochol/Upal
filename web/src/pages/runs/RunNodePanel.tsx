import type { Node } from '@xyflow/react'
import type { NodeData } from '@/entities/workflow'
import type { RunRecord, NodeRunRecord } from '@/shared/types'
import type { NodeRunStatus } from '@/entities/run'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/shared/ui/tabs'
import {
  FileText, AlertTriangle, Info,
  CheckCircle2, XCircle, Clock, Loader2, Timer, X,
} from 'lucide-react'

const nodeStatusConfig: Record<string, { icon: typeof Clock; color: string; label: string }> = {
  idle:      { icon: Clock,        color: 'text-muted-foreground', label: 'Idle' },
  running:   { icon: Loader2,      color: 'text-info',             label: 'Running' },
  completed: { icon: CheckCircle2, color: 'text-success',          label: 'Completed' },
  error:     { icon: XCircle,      color: 'text-destructive',      label: 'Error' },
  waiting:   { icon: Timer,        color: 'text-warning',          label: 'Waiting' },
  skipped:   { icon: XCircle,      color: 'text-muted-foreground', label: 'Skipped' },
}

type Props = {
  selectedNode: Node<NodeData> | null
  run: RunRecord
  onClose: () => void
  onCollapse: () => void
}

function formatDuration(startedAt?: string, completedAt?: string): string {
  if (!startedAt) return '-'
  const start = new Date(startedAt).getTime()
  const end = completedAt ? new Date(completedAt).getTime() : Date.now()
  const ms = end - start
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatTime(iso?: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

function formatOutput(value: unknown): string {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

export function RunNodePanel({ selectedNode, run, onClose, onCollapse }: Props) {
  if (!selectedNode) return null

  const nodeId = selectedNode.id
  const nodeRun: NodeRunRecord | undefined = run.node_runs?.find(
    (nr) => nr.node_id === nodeId,
  )
  const nodeStatus = (nodeRun?.status ?? 'idle') as NodeRunStatus
  const cfg = nodeStatusConfig[nodeStatus] ?? nodeStatusConfig.idle
  const StatusIcon = cfg.icon
  const hasError = !!nodeRun?.error
  const output = run.outputs?.[nodeId]
  const wasExecuted = !!nodeRun

  return (
    <div className="h-full flex flex-col bg-background border-l border-border">
      {/* Node header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <div className="flex items-center gap-2 min-w-0">
          <StatusIcon
            size={16}
            className={`shrink-0 ${cfg.color} ${nodeStatus === 'running' ? 'animate-spin' : ''}`}
          />
          <span className="text-sm font-medium text-foreground truncate">
            {selectedNode.data.label}
          </span>
          <span className={`text-xs ${cfg.color}`}>{cfg.label}</span>
        </div>
        <button
          onClick={onClose}
          className="p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
        >
          <X size={14} />
        </button>
      </div>

      {/* Tabs */}
      <Tabs
        defaultValue="output"
        className="flex-1 flex flex-col min-h-0"
        onValueChange={(value) => {
          // On Output tab, deselecting the node (via close) collapses the panel
          if (value === 'output') {
            // nothing — collapse is handled by onClose
          }
        }}
      >
        <TabsList variant="line" className="px-4 pt-2 shrink-0">
          <TabsTrigger value="output" className="gap-1 text-xs">
            <FileText size={14} /> Output
          </TabsTrigger>
          {hasError && (
            <TabsTrigger value="error" className="gap-1 text-xs">
              <AlertTriangle size={14} /> Error
            </TabsTrigger>
          )}
          <TabsTrigger value="info" className="gap-1 text-xs">
            <Info size={14} /> Info
          </TabsTrigger>
        </TabsList>

        <div className="flex-1 min-h-0 overflow-y-auto">
          {/* Output tab */}
          <TabsContent value="output" className="px-4 py-3">
            {!wasExecuted ? (
              <p className="text-sm text-muted-foreground">Node was not executed</p>
            ) : output ? (
              <pre className="text-xs text-foreground bg-muted p-3 rounded font-mono whitespace-pre-wrap break-words overflow-auto max-h-[60vh]">
                {formatOutput(output)}
              </pre>
            ) : (
              <p className="text-sm text-muted-foreground">No output</p>
            )}
          </TabsContent>

          {/* Error tab */}
          {hasError && (
            <TabsContent value="error" className="px-4 py-3">
              <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-3">
                <p className="text-sm text-destructive font-mono whitespace-pre-wrap break-words">
                  {nodeRun!.error}
                </p>
              </div>
            </TabsContent>
          )}

          {/* Info tab */}
          <TabsContent value="info" className="px-4 py-3">
            <div className="space-y-3">
              {[
                { label: 'Node ID', value: nodeId, mono: true },
                { label: 'Type', value: selectedNode.data.nodeType, capitalize: true },
                { label: 'Status', value: cfg.label },
                { label: 'Started', value: formatTime(nodeRun?.started_at) },
                { label: 'Completed', value: formatTime(nodeRun?.completed_at) },
                { label: 'Duration', value: formatDuration(nodeRun?.started_at, nodeRun?.completed_at), mono: true },
                { label: 'Retries', value: String(nodeRun?.retry_count ?? 0), mono: true },
              ].map((item) => (
                <div key={item.label} className="flex items-start justify-between gap-2">
                  <span className="text-xs text-muted-foreground shrink-0">{item.label}</span>
                  <span
                    className={`text-xs text-foreground text-right ${item.mono ? 'font-mono' : ''} ${item.capitalize ? 'capitalize' : ''}`}
                  >
                    {item.value}
                  </span>
                </div>
              ))}
            </div>
          </TabsContent>
        </div>
      </Tabs>
    </div>
  )
}
