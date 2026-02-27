import { motion } from 'framer-motion'
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

const iconColor: Record<string, string> = {
  'var(--node-input)': 'text-node-input',
  'var(--node-run-input)': 'text-node-run-input',
  'var(--node-agent)': 'text-node-agent',
  'var(--node-output)': 'text-node-output',
  'var(--node-tool)': 'text-node-tool',
  'var(--node-asset)': 'text-node-asset',
}

const hoverAccent: Record<string, string> = {
  'var(--node-input)': 'hover:bg-node-input/10',
  'var(--node-run-input)': 'hover:bg-node-run-input/10',
  'var(--node-agent)': 'hover:bg-node-agent/10',
  'var(--node-output)': 'hover:bg-node-output/10',
  'var(--node-tool)': 'hover:bg-node-tool/10',
  'var(--node-asset)': 'hover:bg-node-asset/10',
}

export function NodePalette({ onAddNode }: NodePaletteProps) {
  const paletteItems = getAllNodeDefinitions()

  return (
    <TooltipProvider delayDuration={400}>
      <motion.div
        initial={{ opacity: 0, y: -12, scale: 0.96 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        transition={{ duration: 0.35, ease: [0.16, 1, 0.3, 1] }}
        className="absolute top-3 left-1/2 -translate-x-1/2 z-10
                    flex items-center gap-0.5 p-1
                    rounded-2xl
                    bg-card/80 backdrop-blur-2xl
                    border border-border/40
                    shadow-[0_8px_32px_-8px_rgba(0,0,0,0.12),0_2px_4px_rgba(0,0,0,0.04),inset_0_1px_0_rgba(255,255,255,0.06)]
                    dark:shadow-[0_8px_32px_-8px_rgba(0,0,0,0.5),0_2px_4px_rgba(0,0,0,0.3),inset_0_1px_0_rgba(255,255,255,0.04)]"
      >
        {paletteItems.map((item, i) => (
          <div key={item.type} className="flex items-center">
            {i > 0 && <div className="w-px h-4 bg-border/60 mx-0.5 shrink-0" />}
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                draggable
                onDragStart={(e) => {
                  e.dataTransfer.setData('application/upal-node-type', item.type)
                  e.dataTransfer.effectAllowed = 'move'
                }}
                onClick={() => onAddNode(item.type)}
                className={cn(
                  'flex items-center gap-2 rounded-xl px-3 py-2',
                  'text-xs font-medium',
                  'cursor-grab active:cursor-grabbing',
                  'transition-all duration-150 ease-out',
                  'text-muted-foreground hover:text-foreground',
                  'active:scale-[0.97]',
                  hoverAccent[item.cssVar],
                )}
              >
                <item.icon className={cn(
                  'h-3.5 w-3.5 shrink-0 transition-colors duration-150',
                  iconColor[item.cssVar],
                )} />
                <span className="tracking-tight">{item.label}</span>
              </button>
            </TooltipTrigger>
            <TooltipContent side="bottom" sideOffset={8}>
              <p>{item.description}</p>
              <p className="text-muted-foreground text-[10px] mt-0.5">Click or drag to canvas</p>
            </TooltipContent>
          </Tooltip>
          </div>
        ))}
      </motion.div>
    </TooltipProvider>
  )
}
