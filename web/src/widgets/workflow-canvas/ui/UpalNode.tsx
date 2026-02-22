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
        'rounded-xl glass-panel w-[280px] cursor-pointer transition-all duration-300',
        selected ? `border-2 ${ntCfg?.borderSelected} shadow-md` : `border ${ntCfg?.border}`,
        isSelected && `ring-2 ring-ring/80 ${ntCfg?.glow} shadow-glow`,
        status.ring,
        isTarget && `ring-2 ring-ring/50 ${ntCfg?.glow}`,
        isSource && 'opacity-60 scale-[0.98]',
        'hover:-translate-y-0.5 hover:shadow-lg'
      )}
      onClick={handleClick}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="!w-3.5 !h-3.5 !bg-background !border-2 !border-primary !-left-2 transition-transform duration-200 hover:scale-125"
      />

      {/* Header bar — accent-colored with icon + label */}
      <div
        className={cn(
          'flex items-center gap-2.5 px-3 py-2.5 backdrop-blur-md bg-opacity-80 rounded-t-xl',
          ntCfg?.headerBg,
          !data.description && 'rounded-b-xl border-b-0',
          'border-b border-border/40' // Add subtle border separation
        )}
      >
        <div
          className={cn(
            'h-7 w-7 rounded-md flex items-center justify-center shrink-0 shadow-sm border border-black/5 dark:border-white/5',
            ntCfg?.accent,
          )}
        >
          {Icon && <Icon className="h-4 w-4 drop-shadow-sm" />}
        </div>

        <p className="text-sm font-semibold text-foreground truncate flex-1 min-w-0 tracking-tight">
          {data.label}
        </p>

        {displayTime != null && (
          <span className="text-[10px] font-mono text-muted-foreground shrink-0 tabular-nums bg-background/50 px-1.5 py-0.5 rounded-md">
            {displayTime.toFixed(2)}s
          </span>
        )}
        {StatusIcon && runStatus === 'running' && (
          <Loader2 className="h-4 w-4 shrink-0 animate-spin text-warning drop-shadow-sm" />
        )}
        {StatusIcon && runStatus === 'completed' && (
          <Check className="h-4 w-4 shrink-0 text-success drop-shadow-sm" />
        )}
        {StatusIcon && runStatus === 'error' && (
          <X className="h-4 w-4 shrink-0 text-destructive drop-shadow-sm" />
        )}
        {StatusIcon && runStatus === 'waiting' && (
          <Hourglass className="h-4 w-4 shrink-0 animate-pulse text-warning drop-shadow-sm" />
        )}
        {StatusIcon && runStatus === 'skipped' && (
          <SkipForward className="h-4 w-4 shrink-0 text-muted-foreground" />
        )}
      </div>

      {/* Description body — flows downward */}
      {data.description && (
        <div className="px-3 py-3 bg-card/40">
          <p className="text-xs text-muted-foreground/90 leading-relaxed line-clamp-4 font-medium">
            {data.description}
          </p>
        </div>
      )}

      {/* Asset preview — shown only for asset nodes */}
      {data.nodeType === 'asset' && (
        <div className="px-3 py-2 border-t border-border/50 space-y-1">
          {(data.config.filename as string | undefined) ? (
            <p className="text-xs font-medium text-card-foreground truncate">
              {data.config.filename as string}
            </p>
          ) : (data.config.file_id as string | undefined) ? (
            <p className="text-xs text-muted-foreground truncate">
              {data.config.file_id as string}
            </p>
          ) : null}
          {(data.config.content_type as string | undefined) && (
            <span className="inline-block text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded font-mono leading-none">
              {data.config.content_type as string}
            </span>
          )}
          {(data.config.preview_text as string | undefined) && (
            <p className="text-[10px] text-muted-foreground leading-relaxed line-clamp-2 break-words">
              {data.config.preview_text as string}
            </p>
          )}
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
