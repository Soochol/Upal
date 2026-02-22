import { Separator } from '@/shared/ui/separator'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/shared/ui/tooltip'
import { cn } from '@/shared/lib/utils'
import { getAllNodeDefinitions } from '@/entities/node'
import type { NodeType } from '@/entities/node'

interface NodePaletteProps {
  onAddNode: (type: NodeType) => void
}

export function NodePalette({ onAddNode }: NodePaletteProps) {
  const paletteItems = getAllNodeDefinitions()

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
                    'flex items-center gap-3 rounded-lg border border-border/50 px-3 py-2.5 text-sm font-medium transition-all duration-200 cursor-grab active:cursor-grabbing hover:shadow-md hover:-translate-y-0.5',
                    item.paletteBg
                  )}
                >
                  <item.icon className="h-4 w-4 shrink-0 drop-shadow-sm" />
                  <span className="tracking-tight">{item.label}</span>
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
