import { useEffect, useRef, useState, useMemo } from 'react'
import { useExecutionStore } from '@/entities/run'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import {
  ChevronUp, ChevronDown, Trash2, Terminal, Copy, Check, Layers,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import {
  eventColorMap,
  formatEvent,
  formatEventVerbose,
  formatRelativeTime,
} from '@/shared/lib/eventFormatting'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import { useCopyToClipboard } from '@/shared/lib/useCopyToClipboard'
import type { RunEvent, NodeCompletedEvent, NodeStartedEvent } from '@/shared/types'

const DEFAULT_HEIGHT = 160
const MIN_HEIGHT = 80
const MAX_HEIGHT = 500

type LogLevel = 'info' | 'debug' | 'verbose'

const INFO_TYPES = new Set([
  'node_started', 'node_completed', 'node_skipped', 'node_waiting', 'node_resumed', 'done', 'error',
])

function matchesLevel(event: RunEvent, level: LogLevel): boolean {
  if (level === 'info') return INFO_TYPES.has(event.type)
  return true
}

function getEventTimestamp(event: RunEvent): number | undefined {
  if (event.type === 'node_started') return (event as NodeStartedEvent).startedAt
  if (event.type === 'node_completed') return (event as NodeCompletedEvent).completedAt
  return undefined
}

type ConsoleRowProps = {
  event: RunEvent
  originalIndex: number
  isExpanded: boolean
  onToggle: (i: number) => void
  runStartTime: number | null
  verbose: boolean
}

function ConsoleRow({ event, originalIndex, isExpanded, onToggle, runStartTime, verbose }: ConsoleRowProps) {
  const ts = getEventTimestamp(event)
  const relTime = formatRelativeTime(ts, runStartTime)
  const color = eventColorMap[event.type] ?? 'text-muted-foreground'
  const text = verbose ? formatEventVerbose(event) : formatEvent(event)
  const completed = event.type === 'node_completed' ? (event as NodeCompletedEvent) : null

  return (
    <div>
      <div
        className={cn('flex items-center gap-1.5 py-0.5 cursor-pointer hover:bg-muted/20 rounded px-1 group', color)}
        onClick={() => onToggle(originalIndex)}
      >
        <span className="text-muted-foreground/50 w-10 text-right shrink-0 text-[10px] tabular-nums">
          {relTime}
        </span>
        <span className="text-muted-foreground/80 shrink-0 w-[110px] truncate text-[10px]">
          {event.type}
        </span>
        <span className="flex-1 truncate">{text}</span>
        {completed?.tokens && (
          <span className="text-muted-foreground/50 shrink-0 text-[10px] tabular-nums">
            {completed.tokens.input}↑{completed.tokens.output}↓
          </span>
        )}
        <ChevronDown
          className={cn(
            'h-3 w-3 shrink-0 text-muted-foreground/30 transition-transform opacity-0 group-hover:opacity-100',
            isExpanded && 'rotate-180 opacity-100',
          )}
        />
      </div>
      {isExpanded && (
        <pre className="ml-[88px] my-0.5 text-[11px] text-muted-foreground/80 bg-muted/20 rounded p-2 overflow-x-auto whitespace-pre-wrap break-all">
          {JSON.stringify(event, null, 2)}
        </pre>
      )}
    </div>
  )
}

type GroupEntry = { event: RunEvent; originalIndex: number }

function NodeGroup({
  nodeId,
  entries,
  expandedRows,
  onToggle,
  runStartTime,
  verbose,
}: {
  nodeId: string
  entries: GroupEntry[]
  expandedRows: Set<number>
  onToggle: (i: number) => void
  runStartTime: number | null
  verbose: boolean
}) {
  const [collapsed, setCollapsed] = useState(false)

  const { totalIn, totalOut, durationStr } = useMemo(() => {
    let totalIn = 0, totalOut = 0
    let firstStart: number | undefined
    let lastComplete: number | undefined
    for (const { event: e } of entries) {
      if (e.type === 'node_completed') {
        const c = e as NodeCompletedEvent
        if (c.tokens) { totalIn += c.tokens.input; totalOut += c.tokens.output }
        if (c.completedAt) lastComplete = Math.max(lastComplete ?? 0, c.completedAt)
      }
      if (e.type === 'node_started') {
        const s = e as NodeStartedEvent
        if (s.startedAt) firstStart = Math.min(firstStart ?? Infinity, s.startedAt)
      }
    }
    let durationStr = ''
    if (firstStart && lastComplete && lastComplete > firstStart) {
      durationStr = `${((lastComplete - firstStart) / 1000).toFixed(1)}s`
    }
    return { totalIn, totalOut, durationStr }
  }, [entries])

  return (
    <div className="mb-1">
      <div
        className="flex items-center gap-1.5 px-1 py-0.5 cursor-pointer hover:bg-muted/20 rounded text-muted-foreground"
        onClick={() => setCollapsed(!collapsed)}
      >
        <ChevronDown className={cn('h-3 w-3 transition-transform', collapsed && '-rotate-90')} />
        <span className="font-medium text-foreground text-[11px]">{nodeId}</span>
        <span className="text-muted-foreground/50 text-[10px]">({entries.length})</span>
        {(totalIn > 0 || totalOut > 0) && (
          <span className="text-muted-foreground/50 text-[10px] tabular-nums ml-1">
            {totalIn}↑{totalOut}↓
          </span>
        )}
        {durationStr && (
          <span className="text-muted-foreground/50 text-[10px] ml-1">{durationStr}</span>
        )}
      </div>
      {!collapsed && (
        <div className="ml-4">
          {entries.map(({ event, originalIndex }) => (
            <ConsoleRow
              key={originalIndex}
              event={event}
              originalIndex={originalIndex}
              isExpanded={expandedRows.has(originalIndex)}
              onToggle={onToggle}
              runStartTime={runStartTime}
              verbose={verbose}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function Console() {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const runStartTime = useExecutionStore((s) => s.runStartTime)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [isExpanded, setIsExpanded] = useState(false)
  const { size: height, handleMouseDown } = useResizeDrag({
    direction: 'vertical',
    min: MIN_HEIGHT,
    max: MAX_HEIGHT,
    initial: DEFAULT_HEIGHT,
  })
  const { copied, copyToClipboard } = useCopyToClipboard()

  const [logLevel, setLogLevel] = useState<LogLevel>('info')
  const [searchQuery, setSearchQuery] = useState('')
  const [nodeFilter, setNodeFilter] = useState('')
  const [groupByNode, setGroupByNode] = useState(false)
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())

  const nodeIds = useMemo(
    () => [...new Set(runEvents.flatMap((e) => ('nodeId' in e && e.nodeId ? [e.nodeId] : [])))],
    [runEvents],
  )

  const filtered = useMemo<GroupEntry[]>(() => {
    const verbose = logLevel === 'verbose'
    return runEvents
      .map((event, originalIndex) => ({ event, originalIndex }))
      .filter(({ event }) => {
        if (!matchesLevel(event, logLevel)) return false
        if (nodeFilter && 'nodeId' in event && event.nodeId !== nodeFilter) return false
        if (searchQuery) {
          const text = (verbose ? formatEventVerbose(event) : formatEvent(event)).toLowerCase()
          const typeStr = event.type.toLowerCase()
          if (!text.includes(searchQuery.toLowerCase()) && !typeStr.includes(searchQuery.toLowerCase())) return false
        }
        return true
      })
  }, [runEvents, logLevel, nodeFilter, searchQuery])

  const handleToggleRow = (i: number) => {
    setExpandedRows((prev) => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const handleCopy = () => {
    const verbose = logLevel === 'verbose'
    const text = filtered
      .map(({ event }) => `[${event.type}] ${verbose ? formatEventVerbose(event) : formatEvent(event)}`)
      .join('\n')
    copyToClipboard(text)
  }

  useEffect(() => {
    if (isRunning) setIsExpanded(true)
  }, [isRunning])

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [filtered])

  const nodeGroups = useMemo(() => {
    if (!groupByNode) return null
    const map = new Map<string, GroupEntry[]>()
    const ungrouped: GroupEntry[] = []
    for (const entry of filtered) {
      const nodeId = 'nodeId' in entry.event ? entry.event.nodeId : null
      if (nodeId) {
        if (!map.has(nodeId)) map.set(nodeId, [])
        map.get(nodeId)!.push(entry)
      } else {
        ungrouped.push(entry)
      }
    }
    return { groups: [...map.entries()], ungrouped }
  }, [filtered, groupByNode])

  return (
    <footer className="border-t border-border bg-background">
      {isExpanded && (
        <div
          onMouseDown={handleMouseDown}
          className="h-1 cursor-row-resize hover:bg-primary/20 active:bg-primary/40 transition-colors"
        />
      )}

      <div className="flex items-center justify-between px-3 h-9 gap-2">
        <button
          className="flex items-center gap-1.5 text-xs hover:text-foreground transition-colors shrink-0"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <Terminal className="h-3.5 w-3.5" />
          <span className="font-medium">Console</span>
          {isRunning ? (
            <Badge variant="default" className="animate-pulse text-[10px] px-1.5 py-0">running</Badge>
          ) : (
            runEvents.length > 0 && (
              <Badge variant="secondary" className="text-[10px] px-1.5 py-0">{filtered.length}/{runEvents.length}</Badge>
            )
          )}
          {isExpanded ? <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" /> : <ChevronUp className="h-3.5 w-3.5 text-muted-foreground" />}
        </button>

        {isExpanded && (
          <div className="flex items-center gap-1 min-w-0">
            <div className="flex rounded border border-border overflow-hidden shrink-0">
              {(['info', 'debug', 'verbose'] as LogLevel[]).map((l) => (
                <button
                  key={l}
                  onClick={() => setLogLevel(l)}
                  className={cn(
                    'px-2 py-0.5 text-[10px] font-medium transition-colors',
                    logLevel === l
                      ? 'bg-primary text-primary-foreground'
                      : 'text-muted-foreground hover:text-foreground hover:bg-muted/30',
                  )}
                >
                  {l}
                </button>
              ))}
            </div>

            <input
              type="text"
              placeholder="search…"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-6 w-28 rounded border border-border bg-background px-2 text-[11px] text-foreground placeholder:text-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-primary/40 min-w-0"
            />

            {nodeIds.length > 0 && (
              <select
                value={nodeFilter}
                onChange={(e) => setNodeFilter(e.target.value)}
                className="h-6 rounded border border-border bg-background px-1 text-[11px] text-foreground focus:outline-none focus:ring-1 focus:ring-primary/40 max-w-[100px]"
              >
                <option value="">all nodes</option>
                {nodeIds.map((id) => (
                  <option key={id} value={id}>{id}</option>
                ))}
              </select>
            )}

            <Button
              variant="ghost"
              size="icon"
              className={cn('h-6 w-6', groupByNode && 'text-primary bg-primary/10')}
              title="Group by node"
              onClick={() => setGroupByNode(!groupByNode)}
            >
              <Layers className="h-3 w-3" />
            </Button>

            {filtered.length > 0 && (
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleCopy}>
                {copied ? <Check className="h-3 w-3 text-success" /> : <Copy className="h-3 w-3" />}
              </Button>
            )}

            {runEvents.length > 0 && (
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
                <Trash2 className="h-3 w-3" />
              </Button>
            )}
          </div>
        )}
      </div>

      {isExpanded && (
        <div
          ref={scrollRef}
          style={{ height }}
          className="overflow-y-auto px-2 pb-3 font-mono text-xs border-t border-border"
        >
          {filtered.length === 0 ? (
            <p className="text-muted-foreground px-2 pt-2">
              {runEvents.length === 0 ? 'Ready. Run a workflow to see events…' : 'No events match current filters.'}
            </p>
          ) : groupByNode && nodeGroups ? (
            <div className="pt-1">
              {nodeGroups.ungrouped.map(({ event, originalIndex }) => (
                <ConsoleRow
                  key={originalIndex}
                  event={event}
                  originalIndex={originalIndex}
                  isExpanded={expandedRows.has(originalIndex)}
                  onToggle={handleToggleRow}
                  runStartTime={runStartTime}
                  verbose={logLevel === 'verbose'}
                />
              ))}
              {nodeGroups.groups.map(([nodeId, entries]) => (
                <NodeGroup
                  key={nodeId}
                  nodeId={nodeId}
                  entries={entries}
                  expandedRows={expandedRows}
                  onToggle={handleToggleRow}
                  runStartTime={runStartTime}
                  verbose={logLevel === 'verbose'}
                />
              ))}
            </div>
          ) : (
            <div className="pt-1">
              {filtered.map(({ event, originalIndex }) => (
                <ConsoleRow
                  key={originalIndex}
                  event={event}
                  originalIndex={originalIndex}
                  isExpanded={expandedRows.has(originalIndex)}
                  onToggle={handleToggleRow}
                  runStartTime={runStartTime}
                  verbose={logLevel === 'verbose'}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </footer>
  )
}
