import { useEffect, useRef } from 'react'
import { useExecutionStore } from '@/stores/executionStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ScrollText } from 'lucide-react'
import { cn } from '@/lib/utils'
import { eventColorMap, formatEvent } from '@/lib/eventFormatting'

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
