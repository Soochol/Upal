import { useEffect, useRef, useState } from 'react'
import { useExecutionStore } from '@/stores/executionStore'
import type { RunEvent } from '@/stores/executionStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ChevronUp, ChevronDown, Trash2, Terminal, Copy, Check } from 'lucide-react'
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

export function BottomConsole() {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isExpanded, setIsExpanded] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    const text = runEvents.map((e) => `[${e.type}] ${formatEvent(e)}`).join('\n')
    navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
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
                {copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
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
