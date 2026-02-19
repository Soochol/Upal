import { type ComponentType } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { Inbox, Bot, Wrench, ArrowRightFromLine, Loader2, Check, X } from 'lucide-react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData, NodeRunStatus } from '@/stores/workflowStore'
import { cn } from '@/lib/utils'

const iconMap: Record<string, ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
}

const colorMap: Record<string, string> = {
  input: 'border-node-input/50 bg-node-input/10',
  agent: 'border-node-agent/50 bg-node-agent/10',
  tool: 'border-node-tool/50 bg-node-tool/10',
  output: 'border-node-output/50 bg-node-output/10',
}

const accentMap: Record<string, string> = {
  input: 'bg-node-input text-node-input-foreground',
  agent: 'bg-node-agent text-node-agent-foreground',
  tool: 'bg-node-tool text-node-tool-foreground',
  output: 'bg-node-output text-node-output-foreground',
}

const glowMap: Record<string, string> = {
  input: 'shadow-[0_0_16px_oklch(0.795_0.184_86.047/0.4)]',
  agent: 'shadow-[0_0_16px_oklch(0.588_0.158_241.966/0.4)]',
  tool: 'shadow-[0_0_16px_oklch(0.637_0.237_25.331/0.4)]',
  output: 'shadow-[0_0_16px_oklch(0.648_0.2_142.495/0.4)]',
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
  const selectedNodeId = useWorkflowStore((s) => s.selectedNodeId)
  const selectNode = useWorkflowStore((s) => s.selectNode)
  const runStatus = useWorkflowStore((s) => s.nodeStatuses[id] ?? 'idle')
  const isSelected = selectedNodeId === id

  const handleClick = () => {
    selectNode(isSelected ? null : id)
  }

  const Icon = iconMap[data.nodeType]
  const status = statusConfig[runStatus]
  const StatusIcon = status.icon

  return (
    <div
      className={cn(
        'rounded-xl border bg-card shadow-sm min-w-[280px] cursor-pointer transition-all duration-200',
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

      <div className="flex items-center gap-3 px-4 py-3">
        <div
          className={cn(
            'h-9 w-9 rounded-lg flex items-center justify-center shrink-0',
            accentMap[data.nodeType],
          )}
        >
          {Icon && <Icon className="h-4.5 w-4.5" />}
        </div>

        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-card-foreground truncate">
            {data.label}
          </p>
          {data.description && (
            <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">
              {data.description}
            </p>
          )}
        </div>

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

      <Handle
        type="source"
        position={Position.Right}
        className="!w-3 !h-3 !bg-border !border-2 !border-background !-right-1.5"
      />
    </div>
  )
}
