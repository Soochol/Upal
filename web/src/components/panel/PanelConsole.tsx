import { useEffect, useRef } from 'react'
import { useExecutionStore } from '@/stores/executionStore'
import { Button } from '@/components/ui/button'
import { Trash2, Copy, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { eventColorMap, formatEvent } from '@/lib/eventFormatting'
import { useCopyToClipboard } from '@/hooks/useCopyToClipboard'
import type { RunEvent } from '@/lib/api'

type PanelConsoleProps = {
  selectedNodeId: string | null
}

function eventBelongsToNode(event: RunEvent, nodeId: string): boolean {
  if ('nodeId' in event && event.nodeId === nodeId) return true
  return false
}

export function PanelConsole({ selectedNodeId }: PanelConsoleProps) {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const scrollRef = useRef<HTMLDivElement>(null)
  const { copied, copyToClipboard } = useCopyToClipboard()

  const filtered = selectedNodeId
    ? runEvents.filter((e) => eventBelongsToNode(e, selectedNodeId))
    : runEvents

  const handleCopy = () => {
    const text = filtered.map((e) => `[${e.type}] ${formatEvent(e)}`).join('\n')
    copyToClipboard(text)
  }

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
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-3 py-2 font-mono text-xs space-y-0.5">
        {filtered.length === 0 ? (
          <p className="text-muted-foreground">
            {runEvents.length === 0
              ? 'Ready. Run a workflow to see events...'
              : 'No events for this node yet.'}
          </p>
        ) : (
          filtered.map((event, i) => (
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
