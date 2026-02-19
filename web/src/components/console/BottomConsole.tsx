import { useEffect, useRef, useState } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { RunEvent } from '@/stores/workflowStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ChevronUp, ChevronDown, Trash2, Terminal } from 'lucide-react'
import { cn } from '@/lib/utils'

const eventColorMap: Record<string, string> = {
  'node.started': 'text-node-agent',
  'node.completed': 'text-node-output',
  'node.error': 'text-destructive',
  'model.request': 'text-purple-500 dark:text-purple-400',
  'model.response': 'text-purple-400 dark:text-purple-300',
  'tool.call': 'text-node-tool',
  'tool.result': 'text-amber-600 dark:text-amber-300',
  'a2a.sending': 'text-blue-500 dark:text-blue-400',
  'a2a.received': 'text-blue-400 dark:text-blue-300',
  'workflow.failed': 'text-destructive font-semibold',
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
  if (data.url) parts.push(`→ ${data.url}`)
  if (data.status) parts.push(`status=${data.status}`)
  if (data.error) parts.push(`error: ${data.error}`)
  if (data.result !== undefined)
    parts.push(
      `result: ${typeof data.result === 'string' ? data.result : JSON.stringify(data.result)}`,
    )
  if (data.output !== undefined)
    parts.push(
      `output: ${typeof data.output === 'string' ? data.output : JSON.stringify(data.output)}`,
    )
  if (parts.length === 0) return JSON.stringify(data)
  return parts.join(' ')
}

export function BottomConsole() {
  const runEvents = useWorkflowStore((s) => s.runEvents)
  const isRunning = useWorkflowStore((s) => s.isRunning)
  const clearRunEvents = useWorkflowStore((s) => s.clearRunEvents)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isExpanded, setIsExpanded] = useState(false)

  // Auto-expand when a run starts
  useEffect(() => {
    if (isRunning) setIsExpanded(true)
  }, [isRunning])

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [runEvents])

  return (
    <footer className="border-t border-border bg-background">
      {/* Toggle bar */}
      <div className="flex items-center justify-between px-4 h-9">
        <button
          className="flex items-center gap-2 text-xs hover:text-foreground transition-colors"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <Terminal className="h-3.5 w-3.5" />
          <span className="font-medium">Console</span>
          {isRunning ? (
            <Badge variant="default" className="animate-pulse text-[10px] px-1.5 py-0">
              running
            </Badge>
          ) : (
            runEvents.length > 0 && (
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                {runEvents.length}
              </Badge>
            )
          )}
          {isExpanded ? (
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
          ) : (
            <ChevronUp className="h-3.5 w-3.5 text-muted-foreground" />
          )}
        </button>
        <div className="flex items-center gap-1">
          {runEvents.length > 0 && (
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
              <Trash2 className="h-3 w-3" />
            </Button>
          )}
        </div>
      </div>

      {/* Expandable log area */}
      {isExpanded && (
        <div
          ref={scrollRef}
          className="h-40 overflow-y-auto px-4 pb-3 font-mono text-xs space-y-0.5 border-t border-border"
        >
          {runEvents.length === 0 ? (
            <p className="text-muted-foreground pt-2">
              Ready. Run a workflow to see events...
            </p>
          ) : (
            runEvents.map((event, i) => (
              <div
                key={i}
                className={cn(eventColorMap[event.type] ?? 'text-muted-foreground')}
              >
                <span className="text-muted-foreground mr-2">{event.type}</span>
                {formatEvent(event)}
              </div>
            ))
          )}
        </div>
      )}
    </footer>
  )
}
