import { type ComponentType, useState, useEffect } from 'react'
import { Handle, Position, useConnection, type NodeProps, type Node } from '@xyflow/react'
import { Loader2, Check, X, Hourglass, SkipForward } from 'lucide-react'
import { useUIStore } from '@/stores/uiStore'
import { useExecutionStore } from '@/stores/executionStore'
import type { NodeRunStatus } from '@/stores/executionStore'
import type { NodeData } from '@/stores/workflowStore'
import { cn } from '@/lib/utils'
import { NODE_TYPES, type NodeType } from '@/lib/nodeTypes'

const statusConfig: Record<
  NodeRunStatus,
  { icon: ComponentType<{ className?: string }> | null; ring: string }
> = {
  idle: { icon: null, ring: '' },
  running: { icon: Loader2, ring: 'ring-2 ring-amber-400' },
  completed: { icon: Check, ring: 'ring-2 ring-green-400' },
  error: { icon: X, ring: 'ring-2 ring-destructive' },
  waiting: { icon: Hourglass, ring: 'ring-2 ring-amber-400 animate-pulse' },
  skipped: { icon: SkipForward, ring: 'ring-2 ring-muted-foreground/40 opacity-60' },
}

export function UpalNode({ id, data, selected }: NodeProps<Node<NodeData>>) {
  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)
  const runStatus = useExecutionStore((s) => s.nodeStatuses[id] ?? 'idle')
  const startTime = useExecutionStore((s) => s.nodeStartTimes[id])
  const finalDuration = useExecutionStore((s) => s.nodeDurations[id])
  const isSelected = selectedNodeId === id

  // Live elapsed timer while node is running
  const [elapsed, setElapsed] = useState(0)
  useEffect(() => {
    if (runStatus !== 'running' || !startTime) {
      setElapsed(0)
      return
    }
    setElapsed((Date.now() - startTime) / 1000)
    const interval = setInterval(() => {
      setElapsed((Date.now() - startTime) / 1000)
    }, 100)
    return () => clearInterval(interval)
  }, [runStatus, startTime])

  const displayTime =
    runStatus === 'running' ? elapsed
    : finalDuration != null ? finalDuration
    : null

  const handleClick = (e: React.MouseEvent) => {
    // Don't override React Flow's multi-selection (Ctrl/Meta+click or Shift+click)
    if (e.shiftKey || e.ctrlKey || e.metaKey) return
    selectNode(id)
  }

  // Connection drag visual feedback
  const connection = useConnection()
  const isTarget = connection.inProgress && connection.fromNode?.id !== id
  const isSource = connection.inProgress && connection.fromNode?.id === id

  const ntCfg = NODE_TYPES[data.nodeType as NodeType]
  const Icon = ntCfg?.icon
  const status = statusConfig[runStatus]
  const StatusIcon = status.icon

  return (
    <div
      className={cn(
        'rounded-xl bg-card shadow-sm w-[280px] cursor-pointer transition-all duration-200',
        selected ? `border-2 ${ntCfg?.borderSelected}` : `border ${ntCfg?.border}`,
        isSelected && `ring-2 ring-ring ${ntCfg?.glow}`,
        status.ring,
        isTarget && `ring-2 ring-ring/50 ${ntCfg?.glow}`,
        isSource && 'opacity-50',
      )}
      onClick={handleClick}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!w-3 !h-3 !bg-border !border-2 !border-background !-left-1.5"
      />

      {/* Header bar — accent-colored with icon + label */}
      <div
        className={cn(
          'flex items-center gap-2.5 px-3 py-2.5',
          ntCfg?.headerBg,
          data.description ? 'rounded-t-xl' : 'rounded-xl',
        )}
      >
        <div
          className={cn(
            'h-7 w-7 rounded-md flex items-center justify-center shrink-0',
            ntCfg?.accent,
          )}
        >
          {Icon && <Icon className="h-3.5 w-3.5" />}
        </div>

        <p className="text-sm font-semibold text-card-foreground truncate flex-1 min-w-0">
          {data.label}
        </p>

        {displayTime != null && (
          <span className="text-[10px] font-mono text-muted-foreground shrink-0 tabular-nums">
            {displayTime.toFixed(2)}s
          </span>
        )}
        {StatusIcon && runStatus === 'running' && (
          <Loader2 className="h-4 w-4 shrink-0 animate-spin text-warning" />
        )}
        {StatusIcon && runStatus === 'completed' && (
          <Check className="h-4 w-4 shrink-0 text-success" />
        )}
        {StatusIcon && runStatus === 'error' && (
          <X className="h-4 w-4 shrink-0 text-destructive" />
        )}
        {StatusIcon && runStatus === 'waiting' && (
          <Hourglass className="h-4 w-4 shrink-0 animate-pulse text-warning" />
        )}
        {StatusIcon && runStatus === 'skipped' && (
          <SkipForward className="h-4 w-4 shrink-0 text-muted-foreground" />
        )}
      </div>

      {/* Description body — flows downward */}
      {data.description && (
        <div className="px-3 py-2.5">
          <p className="text-xs text-muted-foreground leading-relaxed line-clamp-4">
            {data.description}
          </p>
        </div>
      )}

      <Handle
        type="source"
        position={Position.Right}
        className="!w-3 !h-3 !bg-border !border-2 !border-background !-right-1.5"
      />
    </div>
  )
}
