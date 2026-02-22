import { useEffect, useRef, useState } from 'react'
import { useExecutionStore } from '@/entities/run'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import { ChevronUp, ChevronDown, Trash2, Terminal, Copy, Check } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { eventColorMap, formatEvent } from '@/shared/lib/eventFormatting'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import { useCopyToClipboard } from '@/shared/lib/useCopyToClipboard'

const DEFAULT_HEIGHT = 160
const MIN_HEIGHT = 80
const MAX_HEIGHT = 500

export function Console() {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isExpanded, setIsExpanded] = useState(false)
  const { size: height, handleMouseDown } = useResizeDrag({
    direction: 'vertical',
    min: MIN_HEIGHT,
    max: MAX_HEIGHT,
    initial: DEFAULT_HEIGHT,
  })
  const { copied, copyToClipboard } = useCopyToClipboard()

  const handleCopy = () => {
    const text = runEvents.map((e) => `[${e.type}] ${formatEvent(e)}`).join('\n')
    copyToClipboard(text)
  }

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
      {/* Resize handle — only when expanded */}
      {isExpanded && (
        <div
          onMouseDown={handleMouseDown}
          className="h-1 cursor-row-resize hover:bg-primary/20 active:bg-primary/40 transition-colors"
        />
      )}

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

      {/* Expandable log area */}
      {isExpanded && (
        <div
          ref={scrollRef}
          style={{ height }}
          className="overflow-y-auto px-4 pb-3 font-mono text-xs space-y-0.5 border-t border-border"
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
