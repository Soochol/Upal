import { useCallback, useEffect, useRef, useState } from 'react'
import { useExecutionStore } from '@/entities/run'
import { Button } from '@/shared/ui/button'
import { Trash2, Copy, Check, ChevronDown } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { eventColorMap, formatEvent, formatRelativeTime } from '@/shared/lib/eventFormatting'
import { useCopyToClipboard } from '@/shared/lib/useCopyToClipboard'
import type { RunEvent, NodeCompletedEvent, NodeStartedEvent } from '@/shared/types'

type PanelConsoleProps = {
  selectedNodeId: string | null
}

function eventBelongsToNode(event: RunEvent, nodeId: string): boolean {
  return 'nodeId' in event && event.nodeId === nodeId
}

function getEventTimestamp(event: RunEvent): number | undefined {
  if (event.type === 'node_started') return (event as NodeStartedEvent).startedAt
  if (event.type === 'node_completed') return (event as NodeCompletedEvent).completedAt
  return undefined
}

export function PanelConsole({ selectedNodeId }: PanelConsoleProps) {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const runStartTime = useExecutionStore((s) => s.runStartTime)
  const scrollRef = useRef<HTMLDivElement>(null)
  const { copied, copyToClipboard } = useCopyToClipboard()
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())

  const filtered = selectedNodeId
    ? runEvents.map((e, i) => ({ event: e, index: i })).filter(({ event: e }) => eventBelongsToNode(e, selectedNodeId))
    : runEvents.map((e, i) => ({ event: e, index: i }))

  const handleToggle = useCallback((i: number) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }, [])

  const handleCopy = useCallback(() => {
    const text = filtered.map(({ event: e }) => `[${e.type}] ${formatEvent(e)}`).join('\n')
    copyToClipboard(text)
  }, [filtered, copyToClipboard])

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [filtered])

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-border shrink-0">
        <span className="text-xs text-muted-foreground">
          {selectedNodeId ? `Node: ${selectedNodeId}` : 'All nodes'}
        </span>
        <div className="flex items-center gap-1">
          {filtered.length > 0 && (
            <>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleCopy}>
                {copied ? <Check className="h-3 w-3 text-success" /> : <Copy className="h-3 w-3" />}
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
                <Trash2 className="h-3 w-3" />
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Log area */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-2 py-2 font-mono text-xs">
        {filtered.length === 0 ? (
          <p className="text-muted-foreground px-1">
            {runEvents.length === 0 ? 'Ready. Run a workflow to see events...' : 'No events for this node yet.'}
          </p>
        ) : (
          filtered.map(({ event, index }) => {
            const ts = getEventTimestamp(event)
            const relTime = formatRelativeTime(ts, runStartTime)
            const color = eventColorMap[event.type] ?? 'text-muted-foreground'
            const completed = event.type === 'node_completed' ? (event as NodeCompletedEvent) : null
            const isExpanded = expandedRows.has(index)

            return (
              <div key={index}>
                <div
                  className={cn('flex items-center gap-1.5 py-0.5 cursor-pointer hover:bg-muted/20 rounded px-1 group', color)}
                  onClick={() => handleToggle(index)}
                >
                  {relTime && (
                    <span className="text-muted-foreground/50 text-[10px] shrink-0 tabular-nums">{relTime}</span>
                  )}
                  <span className="text-muted-foreground/80 shrink-0 text-[10px]">{event.type}</span>
                  <span className="flex-1 truncate">{formatEvent(event)}</span>
                  {completed?.tokens && (
                    <span className="text-muted-foreground/50 shrink-0 text-[10px] tabular-nums">
                      {completed.tokens.input}↑{completed.tokens.output}↓
                    </span>
                  )}
                  <ChevronDown
                    className={cn(
                      'h-3 w-3 shrink-0 text-muted-foreground/30 opacity-0 group-hover:opacity-100 transition-transform',
                      isExpanded && 'rotate-180 opacity-100',
                    )}
                  />
                </div>
                {isExpanded && (
                  <pre className="ml-2 my-0.5 text-[11px] text-muted-foreground/80 bg-muted/20 rounded p-2 overflow-x-auto whitespace-pre-wrap break-all">
                    {JSON.stringify(event, null, 2)}
                  </pre>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
