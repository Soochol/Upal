import { useEffect, useRef } from 'react'
import { useWorkflowStore } from '../../stores/workflowStore'
import type { RunEvent } from '../../stores/workflowStore'

const eventColorMap: Record<string, string> = {
  'node.started': 'text-blue-400',
  'node.completed': 'text-green-400',
  'node.error': 'text-red-400',
  'model.request': 'text-purple-400',
  'model.response': 'text-purple-300',
  'tool.call': 'text-orange-400',
  'tool.result': 'text-orange-300',
  done: 'text-green-300 font-bold',
  error: 'text-red-400',
  info: 'text-zinc-400',
}

function formatEvent(event: RunEvent): string {
  const data = event.data
  if (data.message && typeof data.message === 'string') {
    return data.message
  }
  // For node events, show the node id and any relevant details
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

export function Console() {
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
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between px-1 mb-1">
        <p className="text-xs text-zinc-500 uppercase font-medium">
          Console
          {isRunning && (
            <span className="ml-2 text-green-400 normal-case animate-pulse">running</span>
          )}
        </p>
        {runEvents.length > 0 && (
          <button
            onClick={clearRunEvents}
            className="text-xs text-zinc-500 hover:text-zinc-300"
          >
            Clear
          </button>
        )}
      </div>
      <div ref={scrollRef} className="flex-1 overflow-y-auto font-mono text-xs space-y-0.5">
        {runEvents.length === 0 ? (
          <p className="text-zinc-600">
            Ready. Add nodes and connect them to build a workflow.
          </p>
        ) : (
          runEvents.map((event, i) => (
            <div key={i} className={eventColorMap[event.type] ?? 'text-zinc-300'}>
              <span className="text-zinc-600 mr-2">{event.type}</span>
              {formatEvent(event)}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
