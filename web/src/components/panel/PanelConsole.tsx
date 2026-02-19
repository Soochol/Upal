import { useEffect, useRef } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { RunEvent } from '@/stores/workflowStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'

const eventColorMap: Record<string, string> = {
  'node.started': 'text-node-agent',
  'node.completed': 'text-node-output',
  'node.error': 'text-destructive',
  'model.request': 'text-purple-500 dark:text-purple-400',
  'model.response': 'text-purple-400 dark:text-purple-300',
  'tool.call': 'text-node-tool',
  'tool.result': 'text-amber-600 dark:text-amber-300',
  done: 'text-node-output font-semibold',
  error: 'text-destructive',
  info: 'text-muted-foreground',
}

function formatEvent(event: RunEvent): string {
  const data = event.data
  if (data.message && typeof data.message === 'string') return data.message
  const parts: string[] = []
  if (data.node_id) parts.push(`[${data.node_id}]`)
  if (data.node_type) parts.push(`(${data.node_type})`)
  if (data.model) parts.push(`model=${data.model}`)
  if (data.tool) parts.push(`tool=${data.tool}`)
  if (data.error) parts.push(`error: ${data.error}`)
  if (data.result !== undefined) parts.push(`result: ${typeof data.result === 'string' ? data.result : JSON.stringify(data.result)}`)
  if (data.output !== undefined) parts.push(`output: ${typeof data.output === 'string' ? data.output : JSON.stringify(data.output)}`)
  if (parts.length === 0) return JSON.stringify(data)
  return parts.join(' ')
}

export function PanelConsole() {
  const runEvents = useWorkflowStore((s) => s.runEvents)
  const isRunning = useWorkflowStore((s) => s.isRunning)
  const clearRunEvents = useWorkflowStore((s) => s.clearRunEvents)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [runEvents])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border">
        <div className="flex items-center gap-2">
          {isRunning && (
            <Badge variant="default" className="animate-pulse text-xs">running</Badge>
          )}
          {!isRunning && runEvents.length > 0 && (
            <Badge variant="secondary" className="text-xs">{runEvents.length} events</Badge>
          )}
        </div>
        {runEvents.length > 0 && (
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
            <Trash2 className="h-3 w-3" />
          </Button>
        )}
      </div>
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-3 py-2 font-mono text-xs space-y-0.5">
        {runEvents.length === 0 ? (
          <p className="text-muted-foreground">Ready. Run a workflow to see events...</p>
        ) : (
          runEvents.map((event, i) => (
            <div key={i} className={cn(eventColorMap[event.type] ?? 'text-muted-foreground')}>
              <span className="text-muted-foreground mr-2">{event.type}</span>
              {formatEvent(event)}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
