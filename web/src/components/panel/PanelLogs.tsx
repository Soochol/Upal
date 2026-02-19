import { useEffect, useRef } from 'react'
import { useExecutionStore } from '@/stores/executionStore'
import type { RunEvent } from '@/stores/executionStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ScrollText } from 'lucide-react'
import { cn } from '@/lib/utils'

const eventColorMap: Record<string, string> = {
  agent: 'text-node-agent',
  event: 'text-muted-foreground',
  done: 'text-node-output font-semibold',
  error: 'text-destructive',
  info: 'text-muted-foreground',
}

function formatEvent(event: RunEvent): string {
  const data = event.data
  // UI-generated events (info, error with message)
  if (data.message && typeof data.message === 'string') return data.message

  const parts: string[] = []

  // ADK Event: show author (node ID) — Go struct uses PascalCase
  if (data.Author) parts.push(`[${data.Author}]`)

  // Show text content from genai.Content
  const content = data.Content as { parts?: { text?: string }[] } | undefined
  if (content?.parts) {
    for (const part of content.parts) {
      if (part.text) parts.push(part.text)
    }
  }

  // Show state delta keys from actions
  const actions = data.Actions as { StateDelta?: Record<string, unknown> } | undefined
  if (actions?.StateDelta) {
    const keys = Object.keys(actions.StateDelta)
    if (keys.length > 0) parts.push(`state: {${keys.join(', ')}}`)
  }

  // Done/error event fields
  if (data.status) parts.push(`status=${data.status}`)
  if (data.error) parts.push(`error: ${data.error}`)

  if (parts.length === 0) return JSON.stringify(data)
  return parts.join(' ')
}

function formatTimestamp(timestamp: unknown): string | null {
  if (!timestamp) return null
  try {
    const date = new Date(timestamp as string | number)
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return null
  }
}

type PanelLogsProps = {
  selectedNodeId: string | null
}

export function PanelLogs({ selectedNodeId }: PanelLogsProps) {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const bottomRef = useRef<HTMLDivElement>(null)

  const filteredEvents = selectedNodeId
    ? runEvents.filter(
        (e) =>
          e.data.Author === selectedNodeId ||
          e.type === 'done' ||
          e.type === 'error' ||
          e.type === 'info',
      )
    : runEvents

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [filteredEvents.length])

  if (runEvents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <ScrollText className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm text-center">Run a workflow to see logs here.</p>
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-3 space-y-0.5 font-mono text-xs">
        {selectedNodeId && (
          <p className="text-muted-foreground text-[10px] uppercase tracking-wider mb-2 font-sans">
            Filtered: {selectedNodeId}
          </p>
        )}
        {filteredEvents.map((event, i) => {
          const timestamp = formatTimestamp(event.data.Timestamp)
          return (
            <div
              key={i}
              className={cn(
                'flex gap-2 leading-relaxed',
                eventColorMap[event.type] ?? 'text-muted-foreground',
              )}
            >
              {timestamp && (
                <span className="text-muted-foreground/60 shrink-0">{timestamp}</span>
              )}
              <span className="text-muted-foreground shrink-0">{event.type}</span>
              <span className="break-all">{formatEvent(event)}</span>
            </div>
          )
        })}
        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  )
}
