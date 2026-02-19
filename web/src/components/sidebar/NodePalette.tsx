import { Inbox, Bot, Wrench, ArrowRightFromLine, Globe } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

type NodeType = 'input' | 'agent' | 'tool' | 'output' | 'external'

const paletteItems = [
  {
    type: 'input' as NodeType,
    label: 'Input',
    description: 'User-provided data entry point',
    icon: Inbox,
    colorClass:
      'bg-node-input/15 text-node-input border-node-input/30 hover:bg-node-input/25',
  },
  {
    type: 'agent' as NodeType,
    label: 'Agent',
    description: 'AI model processing step',
    icon: Bot,
    colorClass:
      'bg-node-agent/15 text-node-agent border-node-agent/30 hover:bg-node-agent/25',
  },
  {
    type: 'tool' as NodeType,
    label: 'Tool',
    description: 'External tool or function call',
    icon: Wrench,
    colorClass:
      'bg-node-tool/15 text-node-tool border-node-tool/30 hover:bg-node-tool/25',
  },
  {
    type: 'output' as NodeType,
    label: 'Output',
    description: 'Workflow result endpoint',
    icon: ArrowRightFromLine,
    colorClass:
      'bg-node-output/15 text-node-output border-node-output/30 hover:bg-node-output/25',
  },
  {
    type: 'external' as NodeType,
    label: 'External',
    description: 'External A2A-compatible agent',
    icon: Globe,
    colorClass:
      'bg-purple-500/15 text-purple-500 border-purple-500/30 hover:bg-purple-500/25',
  },
]

interface NodePaletteProps {
  onAddNode: (type: NodeType) => void
}

export function NodePalette({ onAddNode }: NodePaletteProps) {
  return (
    <TooltipProvider delayDuration={300}>
      <aside className="w-56 border-r border-border bg-sidebar p-4 flex flex-col">
        <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-3">
          Components
        </p>
        <div className="flex flex-col gap-2">
          {paletteItems.map((item) => (
            <Tooltip key={item.type}>
              <TooltipTrigger asChild>
                <button
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData('application/upal-node-type', item.type)
                    e.dataTransfer.effectAllowed = 'move'
                  }}
                  onClick={() => onAddNode(item.type)}
                  className={cn(
                    'flex items-center gap-3 rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors cursor-grab active:cursor-grabbing',
                    item.colorClass
                  )}
                >
                  <item.icon className="h-4 w-4 shrink-0" />
                  <span>{item.label}</span>
                </button>
              </TooltipTrigger>
              <TooltipContent side="right">{item.description}</TooltipContent>
            </Tooltip>
          ))}
        </div>
        <Separator className="my-4" />
        <p className="text-xs text-muted-foreground">
          Click to add a step, then connect nodes on the canvas.
        </p>
      </aside>
    </TooltipProvider>
  )
}
