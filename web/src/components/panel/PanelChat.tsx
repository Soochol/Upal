import { useWorkflowStore } from '@/stores/workflowStore'
import { useAutoScroll } from '@/hooks/useAutoScroll'
import { useExecutionStore } from '@/stores/executionStore'
import type { RunEvent, ToolCallEvent, ToolResultEvent, NodeCompletedEvent } from '@/lib/api'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Bot, Wrench, MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'

const chatEventTypes = new Set(['tool_call', 'tool_result', 'node_completed'])

type PanelChatProps = {
  selectedNodeId: string | null
}

/** Extract nodeId from events that carry one */
function getNodeId(event: RunEvent): string {
  if ('nodeId' in event) return event.nodeId
  return ''
}

/** Group consecutive events by nodeId into sections */
type NodeSection = {
  nodeId: string
  events: RunEvent[]
}

function groupByNode(events: RunEvent[]): NodeSection[] {
  const sections: NodeSection[] = []
  let current: NodeSection | null = null

  for (const event of events) {
    const nodeId = getNodeId(event)
    if (!nodeId) continue

    if (!current || current.nodeId !== nodeId) {
      current = { nodeId, events: [] }
      sections.push(current)
    }

    current.events.push(event)
  }

  return sections
}

function ToolCallCard({ tool, args }: { tool: string; args: string }) {
  const preview = args.length > 80 ? args.slice(0, 80) + '\u2026' : args

  return (
    <div className="flex items-start gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2 text-xs">
      <Wrench className="h-3.5 w-3.5 mt-0.5 shrink-0 text-amber-500" />
      <div className="min-w-0">
        <span className="font-medium">{tool}</span>
        {preview && (
          <span className="text-muted-foreground ml-1.5 break-all">{preview}</span>
        )}
      </div>
    </div>
  )
}

function ToolResultCard({ tool, result }: { tool: string; result: string }) {
  const preview = result.length > 200 ? result.slice(0, 200) + '\u2026' : result

  return (
    <div className="ml-5 flex items-start gap-2 rounded-lg border border-border bg-muted/20 px-3 py-2 text-xs">
      <Wrench className="h-3.5 w-3.5 mt-0.5 shrink-0 text-amber-400 opacity-60" />
      <div className="min-w-0">
        <span className="text-muted-foreground font-medium">{tool}</span>
        <pre className="mt-1 whitespace-pre-wrap break-all text-muted-foreground">{preview}</pre>
      </div>
    </div>
  )
}

function AssistantBubble({ content }: { content: string }) {
  return (
    <div className={cn('flex justify-start')}>
      <div className="flex items-start gap-2 max-w-[85%]">
        <div className="mt-1 shrink-0">
          <Bot className="h-4 w-4 text-muted-foreground" />
        </div>
        <div className="rounded-2xl px-3 py-2 text-sm bg-muted rounded-bl-sm">
          <div className="whitespace-pre-wrap break-words">{content}</div>
        </div>
      </div>
    </div>
  )
}

function renderEvent(event: RunEvent, index: number) {
  if (event.type === 'tool_call') {
    const tc = event as ToolCallEvent
    return (
      <div key={index} className="space-y-1">
        {tc.calls.map((c, ci) => (
          <ToolCallCard key={ci} tool={c.name} args={JSON.stringify(c.args ?? {})} />
        ))}
      </div>
    )
  }

  if (event.type === 'tool_result') {
    const tr = event as ToolResultEvent
    return (
      <div key={index} className="space-y-1">
        {tr.results.map((r, ri) => (
          <ToolResultCard key={ri} tool={r.name} result={JSON.stringify(r.response ?? {})} />
        ))}
      </div>
    )
  }

  if (event.type === 'node_completed') {
    const nc = event as NodeCompletedEvent
    if (!nc.output) return null
    return <AssistantBubble key={index} content={nc.output} />
  }

  return null
}

export function PanelChat({ selectedNodeId }: PanelChatProps) {
  const runEvents = useExecutionStore((s) => s.runEvents)
  const nodes = useWorkflowStore((s) => s.nodes)

  // Filter to chat-related events
  const chatEvents = runEvents.filter((e) => chatEventTypes.has(e.type))

  // Optionally filter by selected node
  const filteredEvents = selectedNodeId
    ? chatEvents.filter((e) => getNodeId(e) === selectedNodeId)
    : chatEvents

  // Group by node
  const sections = groupByNode(filteredEvents)

  const bottomRef = useAutoScroll(filteredEvents.length)

  // Look up the node label from the store nodes list
  const getNodeLabel = (nodeId: string): string => {
    const node = nodes.find((n) => n.id === nodeId)
    return node?.data?.label ?? nodeId
  }

  if (chatEvents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <MessageSquare className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm text-center">
          Run a workflow to see conversations here.
        </p>
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-3 space-y-4">
        {selectedNodeId && (
          <p className="text-muted-foreground text-[10px] uppercase tracking-wider font-sans">
            Filtered: {getNodeLabel(selectedNodeId)}
          </p>
        )}
        {sections.map((section, si) => (
          <div key={si} className="space-y-2">
            {/* Node section header */}
            {!selectedNodeId && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground border-b border-border pb-1.5 mb-1">
                <Bot className="h-3.5 w-3.5" />
                <span className="font-medium">{getNodeLabel(section.nodeId)}</span>
              </div>
            )}
            {section.events.map((event, ei) => renderEvent(event, si * 1000 + ei))}
          </div>
        ))}
        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  )
}
