import { useEffect, useRef } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { RunEvent } from '@/stores/workflowStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Bot, User, Wrench, MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'

const chatEventTypes = new Set([
  'model.request',
  'model.response',
  'tool.call',
  'tool.result',
])

type ChatMessage = {
  role: 'system' | 'user' | 'assistant' | 'tool'
  content: string
  toolCalls?: unknown[]
}

type PanelChatProps = {
  selectedNodeId: string | null
}

/** Extract message objects from a model.request event's payload.messages */
function extractMessages(event: RunEvent): ChatMessage[] {
  const payload = event.data.payload as Record<string, unknown> | undefined
  if (!payload) return []
  const msgs = payload.messages as ChatMessage[] | undefined
  return msgs ?? []
}

/** Get the node_id from an event */
function getNodeId(event: RunEvent): string {
  return (event.data.node_id as string) ?? ''
}

/** Get model from the closest model.request event payload */
function getModel(event: RunEvent): string {
  const payload = event.data.payload as Record<string, unknown> | undefined
  return (payload?.model as string) ?? ''
}

/** Group consecutive events by node_id into sections */
type NodeSection = {
  nodeId: string
  model: string
  events: RunEvent[]
}

function groupByNode(events: RunEvent[]): NodeSection[] {
  const sections: NodeSection[] = []
  let current: NodeSection | null = null

  for (const event of events) {
    const nodeId = getNodeId(event)
    if (!nodeId) continue

    if (!current || current.nodeId !== nodeId) {
      const model =
        event.type === 'model.request' ? getModel(event) : ''
      current = { nodeId, model, events: [] }
      sections.push(current)
    }

    // Capture model from the first model.request in this section
    if (!current.model && event.type === 'model.request') {
      current.model = getModel(event)
    }

    current.events.push(event)
  }

  return sections
}

function ChatBubble({ role, content }: { role: string; content: string }) {
  const isUser = role === 'user'

  return (
    <div className={cn('flex', isUser ? 'justify-end' : 'justify-start')}>
      <div className="flex items-start gap-2 max-w-[85%]">
        {!isUser && (
          <div className="mt-1 shrink-0">
            <Bot className="h-4 w-4 text-muted-foreground" />
          </div>
        )}
        <div
          className={cn(
            'rounded-2xl px-3 py-2 text-sm',
            isUser
              ? 'bg-primary text-primary-foreground rounded-br-sm'
              : 'bg-muted rounded-bl-sm',
          )}
        >
          <div className="whitespace-pre-wrap break-words">{content}</div>
        </div>
        {isUser && (
          <div className="mt-1 shrink-0">
            <User className="h-4 w-4 text-muted-foreground" />
          </div>
        )}
      </div>
    </div>
  )
}

function ToolCallCard({
  tool,
  args,
}: {
  tool: string
  args: string
}) {
  const preview =
    args.length > 80 ? args.slice(0, 80) + '\u2026' : args

  return (
    <div className="flex items-start gap-2 rounded-lg border border-border bg-muted/30 px-3 py-2 text-xs">
      <Wrench className="h-3.5 w-3.5 mt-0.5 shrink-0 text-amber-500" />
      <div className="min-w-0">
        <span className="font-medium">{tool}</span>
        {preview && (
          <span className="text-muted-foreground ml-1.5 break-all">
            {preview}
          </span>
        )}
      </div>
    </div>
  )
}

function ToolResultCard({
  tool,
  result,
}: {
  tool: string
  result: string
}) {
  const preview =
    result.length > 200 ? result.slice(0, 200) + '\u2026' : result

  return (
    <div className="ml-5 flex items-start gap-2 rounded-lg border border-border bg-muted/20 px-3 py-2 text-xs">
      <Wrench className="h-3.5 w-3.5 mt-0.5 shrink-0 text-amber-400 opacity-60" />
      <div className="min-w-0">
        <span className="text-muted-foreground font-medium">{tool}</span>
        <pre className="mt-1 whitespace-pre-wrap break-all text-muted-foreground">
          {preview}
        </pre>
      </div>
    </div>
  )
}

function renderEvent(event: RunEvent, index: number) {
  const payload = event.data.payload as Record<string, unknown> | undefined

  if (event.type === 'model.request') {
    const messages = extractMessages(event)
    return (
      <div key={index} className="space-y-2">
        {messages.map((msg, mi) => {
          if (msg.role === 'system') {
            return (
              <div
                key={mi}
                className="text-[10px] uppercase tracking-wider text-muted-foreground/60 px-1"
              >
                System prompt
              </div>
            )
          }
          if (msg.role === 'tool') {
            return null // Tool messages in the request are already shown as tool.result events
          }
          return (
            <ChatBubble
              key={mi}
              role={msg.role}
              content={msg.content}
            />
          )
        })}
      </div>
    )
  }

  if (event.type === 'model.response') {
    const content = (payload?.content as string) ?? ''
    if (!content) return null
    return (
      <ChatBubble key={index} role="assistant" content={content} />
    )
  }

  if (event.type === 'tool.call') {
    const tool = (payload?.tool as string) ?? 'unknown'
    const args = typeof payload?.args === 'string'
      ? payload.args
      : JSON.stringify(payload?.args ?? '')
    return <ToolCallCard key={index} tool={tool} args={args} />
  }

  if (event.type === 'tool.result') {
    const tool = (payload?.tool as string) ?? 'unknown'
    const result = (payload?.result as string) ?? ''
    return <ToolResultCard key={index} tool={tool} result={result} />
  }

  return null
}

export function PanelChat({ selectedNodeId }: PanelChatProps) {
  const runEvents = useWorkflowStore((s) => s.runEvents)
  const nodes = useWorkflowStore((s) => s.nodes)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Filter to chat-related events
  const chatEvents = runEvents.filter((e) => chatEventTypes.has(e.type))

  // Optionally filter by selected node
  const filteredEvents = selectedNodeId
    ? chatEvents.filter((e) => getNodeId(e) === selectedNodeId)
    : chatEvents

  // Group by node
  const sections = groupByNode(filteredEvents)

  // Auto-scroll on new events
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [filteredEvents.length])

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
                <span className="font-medium">
                  {getNodeLabel(section.nodeId)}
                </span>
                {section.model && (
                  <span className="text-muted-foreground/60">
                    {section.model}
                  </span>
                )}
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
