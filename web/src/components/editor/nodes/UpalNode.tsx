import { type ComponentType, useState, useEffect } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { Loader2, Check, X } from 'lucide-react'
import { useUIStore } from '@/stores/uiStore'
import { useExecutionStore } from '@/stores/executionStore'
import type { NodeRunStatus } from '@/stores/executionStore'
import type { NodeData } from '@/stores/workflowStore'
import { cn } from '@/lib/utils'
import { nodeIconMap } from '@/lib/nodeTypes'

const colorMap: Record<string, string> = {
  input: 'border-node-input/30',
  agent: 'border-node-agent/30',
  tool: 'border-node-tool/30',
  output: 'border-node-output/30',
  external: 'border-purple-500/30',
}

const headerMap: Record<string, string> = {
  input: 'bg-node-input/15',
  agent: 'bg-node-agent/15',
  tool: 'bg-node-tool/15',
  output: 'bg-node-output/15',
  external: 'bg-purple-500/15',
}

const accentMap: Record<string, string> = {
  input: 'bg-node-input text-node-input-foreground',
  agent: 'bg-node-agent text-node-agent-foreground',
  tool: 'bg-node-tool text-node-tool-foreground',
  output: 'bg-node-output text-node-output-foreground',
  external: 'bg-purple-500 text-white',
}

const glowMap: Record<string, string> = {
  input: 'shadow-[0_0_16px_oklch(0.795_0.184_86.047/0.4)]',
  agent: 'shadow-[0_0_16px_oklch(0.588_0.158_241.966/0.4)]',
  tool: 'shadow-[0_0_16px_oklch(0.637_0.237_25.331/0.4)]',
  output: 'shadow-[0_0_16px_oklch(0.648_0.2_142.495/0.4)]',
  external: 'shadow-[0_0_16px_oklch(0.553_0.235_303.4/0.4)]',
}

const statusConfig: Record<
  NodeRunStatus,
  { icon: ComponentType<{ className?: string }> | null; ring: string }
> = {
  idle: { icon: null, ring: '' },
  running: { icon: Loader2, ring: 'ring-2 ring-amber-400' },
  completed: { icon: Check, ring: 'ring-2 ring-green-400' },
  error: { icon: X, ring: 'ring-2 ring-destructive' },
}

export function UpalNode({ id, data }: NodeProps<Node<NodeData>>) {
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

  const Icon = nodeIconMap[data.nodeType]
  const status = statusConfig[runStatus]
  const StatusIcon = status.icon

  return (
    <div
      className={cn(
        'rounded-xl border bg-card shadow-sm w-[280px] cursor-pointer transition-all duration-200',
        colorMap[data.nodeType],
        isSelected && `ring-2 ring-ring ${glowMap[data.nodeType]}`,
        status.ring,
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
          headerMap[data.nodeType],
          data.description ? 'rounded-t-xl' : 'rounded-xl',
        )}
      >
        <div
          className={cn(
            'h-7 w-7 rounded-md flex items-center justify-center shrink-0',
            accentMap[data.nodeType],
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
          <Loader2 className="h-4 w-4 shrink-0 animate-spin text-amber-400" />
        )}
        {StatusIcon && runStatus === 'completed' && (
          <Check className="h-4 w-4 shrink-0 text-green-400" />
        )}
        {StatusIcon && runStatus === 'error' && (
          <X className="h-4 w-4 shrink-0 text-destructive" />
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
