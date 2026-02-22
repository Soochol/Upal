import { useRef } from 'react'
import { Upload } from 'lucide-react'
import { Separator } from '@/components/ui/separator'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { getAllNodeDefinitions } from '@/entities/node'
import type { NodeType } from '@/entities/node'
import { uploadFile } from '@/lib/api/upload'
import { useWorkflowStore } from '@/entities/workflow'

interface NodePaletteProps {
  onAddNode: (type: NodeType) => void
}

export function NodePalette({ onAddNode }: NodePaletteProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const addNode = useWorkflowStore((s) => s.addNode)

  const paletteItems = getAllNodeDefinitions()

  function handleFileUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    files.forEach((file, i) => {
      uploadFile(file)
        .then((result) => {
          addNode('asset', { x: 100 + i * 20, y: 100 + i * 20 }, {
            file_id: result.id,
            filename: result.filename,
            content_type: result.content_type,
            preview_text: result.preview_text ?? '',
          })
        })
        .catch((err) => {
          console.error('Asset upload failed:', err)
        })
    })
    e.target.value = ''
  }

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
                    item.paletteBg
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
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={handleFileUpload}
        />
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={() => fileInputRef.current?.click()}
              className="flex items-center gap-3 rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors bg-node-asset/15 text-node-asset border-node-asset/30 hover:bg-node-asset/25"
            >
              <Upload className="h-4 w-4 shrink-0" />
              <span>Upload File</span>
            </button>
          </TooltipTrigger>
          <TooltipContent side="right">Upload a file and create an asset node</TooltipContent>
        </Tooltip>
        <Separator className="my-4" />
        <p className="text-xs text-muted-foreground">
          Click to add a step, then connect nodes on the canvas.
        </p>
      </aside>
    </TooltipProvider>
  )
}
