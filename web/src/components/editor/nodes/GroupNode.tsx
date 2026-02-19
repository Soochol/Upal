import { memo } from 'react'
import { NodeResizer, type NodeProps, type Node } from '@xyflow/react'
import { useUIStore } from '@/stores/uiStore'
import type { NodeData } from '@/stores/workflowStore'
import { cn } from '@/lib/utils'

const colorStyles: Record<string, { border: string; bg: string }> = {
  purple: { border: 'border-purple-400/50', bg: 'bg-purple-400/5' },
  blue: { border: 'border-blue-400/50', bg: 'bg-blue-400/5' },
  green: { border: 'border-green-400/50', bg: 'bg-green-400/5' },
  orange: { border: 'border-orange-400/50', bg: 'bg-orange-400/5' },
  pink: { border: 'border-pink-400/50', bg: 'bg-pink-400/5' },
}

function GroupNodeComponent({ id, data, selected }: NodeProps<Node<NodeData>>) {
  const selectNode = useUIStore((s) => s.selectNode)
  const color = colorStyles[(data.config.color as string) ?? 'purple'] ?? colorStyles.purple

  return (
    <div
      className={cn(
        'rounded-2xl border-2 border-dashed min-w-[320px] min-h-[200px] w-full h-full',
        color.border,
        color.bg,
        selected && 'ring-2 ring-ring',
      )}
      onClick={(e) => {
        e.stopPropagation()
        selectNode(id)
      }}
    >
      <NodeResizer
        isVisible={selected}
        minWidth={320}
        minHeight={200}
        lineClassName="!border-purple-400/50"
        handleClassName="!w-2.5 !h-2.5 !bg-purple-400 !border-2 !border-background !rounded-sm"
      />
      <div className="px-3 py-1.5">
        <span className="text-xs font-medium text-muted-foreground">
          {data.label || 'Group'}
        </span>
      </div>
    </div>
  )
}

export const GroupNode = memo(GroupNodeComponent)
