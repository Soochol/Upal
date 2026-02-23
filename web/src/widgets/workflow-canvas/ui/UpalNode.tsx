import { type ComponentType, useState, useEffect } from 'react'
import { Handle, Position, useConnection, type NodeProps, type Node } from '@xyflow/react'
import { Loader2, Check, X, Hourglass, SkipForward } from 'lucide-react'
import { useUIStore } from '@/entities/ui'
import { useExecutionStore } from '@/entities/run'
import type { NodeRunStatus } from '@/entities/run'
import type { NodeData } from '@/entities/workflow'
import { cn } from '@/shared/lib/utils'
import { getAllNodeDefinitions, getNodeDefinition } from '@/entities/node'
import type { NodeType } from '@/entities/node'

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

// Silence unused import warning — getAllNodeDefinitions is imported to ensure
// the registry is populated when this module loads.
void getAllNodeDefinitions

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

  let ntCfg: ReturnType<typeof getNodeDefinition> | undefined
  try {
    ntCfg = getNodeDefinition(data.nodeType as NodeType)
  } catch {
    ntCfg = undefined
  }
  const Icon = ntCfg?.icon
  const status = statusConfig[runStatus]
  const StatusIcon = status.icon

  return (
    <div
      className={cn(
        'group relative rounded-[20px] transition-all duration-300 w-[300px]',
        'bg-card/70 backdrop-blur-xl border border-white/5 shadow-[0_8px_30px_rgb(0,0,0,0.12)]',
        'hover:-translate-y-1 hover:shadow-[0_20px_40px_rgba(0,0,0,0.2)] hover:border-white/10',
        selected && 'ring-2 ring-primary/80 border-primary/50 shadow-[0_0_30px_var(--primary)/20]',
        isSelected && 'ring-2 ring-primary border-primary shadow-[0_0_40px_var(--primary)/30]',
        status.ring,
        isTarget && 'ring-2 ring-primary/50 scale-105',
        isSource && 'opacity-50 scale-[0.97]'
      )}
      onClick={handleClick}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!w-4 !h-4 !bg-background !border-2 !border-muted-foreground !-left-2 transition-transform duration-200 hover:scale-125 hover:!border-primary"
      />

      {/* Header bar — accent-colored with icon + label */}
      <div
        className={cn(
          'flex items-center gap-3 px-4 py-3 rounded-t-[20px] relative overflow-hidden',
          ntCfg?.headerBg,
          !data.description && 'rounded-b-[20px]'
        )}
      >
        {/* Subtle gradient overlay */}
        <div className="absolute inset-0 bg-gradient-to-r from-transparent to-black/5 dark:to-white/5 pointer-events-none" />

        <div
          className={cn(
            'size-8 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm border border-black/10 dark:border-white/10 relative z-10',
            ntCfg?.accent,
          )}
        >
          {Icon && <Icon className="size-4.5 drop-shadow-sm" />}
        </div>

        <div className="flex-1 min-w-0 relative z-10">
          <p className="text-[15px] font-bold text-foreground truncate tracking-tight">
            {data.label}
          </p>
        </div>

        <div className="flex items-center gap-2 shrink-0 relative z-10">
          {displayTime != null && (
            <span className="text-[11px] font-mono text-muted-foreground bg-black/5 dark:bg-white/5 px-2 py-0.5 rounded-md">
              {displayTime.toFixed(2)}s
            </span>
          )}
          {StatusIcon && runStatus === 'running' && (
            <Loader2 className="size-4 animate-spin text-warning drop-shadow-sm" />
          )}
          {StatusIcon && runStatus === 'completed' && (
            <Check className="size-4 text-success drop-shadow-sm" />
          )}
          {StatusIcon && runStatus === 'error' && (
            <X className="size-4 text-destructive drop-shadow-sm" />
          )}
          {StatusIcon && runStatus === 'waiting' && (
            <Hourglass className="size-4 animate-pulse text-warning drop-shadow-sm" />
          )}
          {StatusIcon && runStatus === 'skipped' && (
            <SkipForward className="size-4 text-muted-foreground" />
          )}
        </div>
      </div>

      {/* Description body */}
      {data.description && (
        <div className="px-4 py-3.5 bg-black/5 dark:bg-white/[0.02] border-t border-white/5 rounded-b-[20px]">
          <p className="text-[13px] text-muted-foreground/90 leading-relaxed max-h-[80px] overflow-hidden font-medium">
            {data.description}
          </p>
        </div>
      )}

      {/* Asset preview */}
      {data.nodeType === 'asset' && (
        <div className="px-4 py-3 bg-black/10 dark:bg-black/20 border-t border-white/5 rounded-b-[20px] space-y-1.5">
          {(data.config.filename as string | undefined) ? (
            <div className="flex items-center gap-2">
              <div className="size-2 rounded-full bg-primary animate-pulse" />
              <p className="text-[13px] font-semibold text-foreground truncate">
                {data.config.filename as string}
              </p>
            </div>
          ) : (data.config.file_id as string | undefined) ? (
            <p className="text-xs text-muted-foreground truncate">
              {data.config.file_id as string}
            </p>
          ) : null}
        </div>
      )}

      <Handle
        type="source"
        position={Position.Right}
        className="!w-4 !h-4 !bg-border !border-2 !border-background !-right-2 transition-transform duration-200 hover:scale-125 hover:!border-primary hover:!bg-background"
      />
    </div>
  )
}
