import { useState, useRef, useEffect, useCallback } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { configureNode } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Sparkles, SendHorizontal, Loader2, ChevronDown, Bot, User } from 'lucide-react'
import { cn } from '@/lib/utils'

type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
}

type AIChatEditorProps = {
  nodeId: string
  data: NodeData
}

export function AIChatEditor({ nodeId, data }: AIChatEditorProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(true)

  const scrollRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)
  const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)
  const edges = useWorkflowStore((s) => s.edges)
  const nodes = useWorkflowStore((s) => s.nodes)

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  // Compute upstream nodes by traversing edges backwards from this nodeId
  const getUpstreamNodes = useCallback(() => {
    const upstreamIds = edges
      .filter((e) => e.target === nodeId)
      .map((e) => e.source)

    const result: { id: string; type: string; label: string }[] = []
    for (const id of upstreamIds) {
      const node = nodes.find((n) => n.id === id)
      if (node) {
        result.push({
          id: node.id,
          type: node.data.nodeType as string,
          label: node.data.label,
        })
      }
    }
    return result
  }, [edges, nodes, nodeId])

  const handleSubmit = async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)

    try {
      const history = messages.map((m) => ({
        role: m.role,
        content: m.content,
      }))

      const response = await configureNode({
        node_type: data.nodeType,
        node_id: nodeId,
        current_config: data.config,
        label: data.label,
        description: data.description ?? '',
        message: trimmed,
        history,
        upstream_nodes: getUpstreamNodes(),
      })

      // Apply returned config updates
      if (response.config && Object.keys(response.config).length > 0) {
        updateNodeConfig(nodeId, response.config)
      }
      if (response.label) {
        updateNodeLabel(nodeId, response.label)
      }
      if (response.description) {
        updateNodeDescription(nodeId, response.description)
      }

      // Add AI explanation as assistant message
      const explanation = response.explanation || 'Configuration updated.'
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: explanation },
      ])
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : 'An unknown error occurred'
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: errorMsg, isError: true },
      ])
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger asChild>
        <button className="flex w-full items-center justify-between px-2 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">
          <span className="flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5" />
            AI Assistant
          </span>
          <ChevronDown
            className={cn(
              'h-3.5 w-3.5 transition-transform',
              isOpen && 'rotate-180',
            )}
          />
        </button>
      </CollapsibleTrigger>

      <CollapsibleContent>
        <div className="px-2 pb-2 space-y-2">
          {/* Chat history */}
          {messages.length > 0 && (
            <ScrollArea className="max-h-40">
              <div className="space-y-1.5 pr-1">
                {messages.map((msg, i) => (
                  <div
                    key={i}
                    className={cn(
                      'flex',
                      msg.role === 'user' ? 'justify-end' : 'justify-start',
                    )}
                  >
                    <div className="flex items-start gap-1 max-w-[95%]">
                      {msg.role === 'assistant' && (
                        <Bot className="h-3 w-3 mt-0.5 shrink-0 text-muted-foreground" />
                      )}
                      <div
                        className={cn(
                          'rounded-md px-2 py-1 text-[11px]',
                          msg.role === 'user'
                            ? 'bg-primary text-primary-foreground'
                            : msg.isError
                              ? 'bg-destructive/10 text-destructive border border-destructive/20'
                              : 'bg-muted',
                        )}
                      >
                        <p className="whitespace-pre-wrap break-words">
                          {msg.content}
                        </p>
                      </div>
                      {msg.role === 'user' && (
                        <User className="h-3 w-3 mt-0.5 shrink-0 text-muted-foreground" />
                      )}
                    </div>
                  </div>
                ))}
                {isLoading && (
                  <div className="flex items-center gap-1 text-muted-foreground">
                    <Bot className="h-3 w-3" />
                    <Loader2 className="h-2.5 w-2.5 animate-spin" />
                    <span className="text-[11px]">Thinking...</span>
                  </div>
                )}
                <div ref={scrollRef} />
              </div>
            </ScrollArea>
          )}

          {/* Input bar */}
          <div className="flex items-end gap-1">
            <Textarea
              ref={textareaRef}
              className="min-h-[32px] max-h-[60px] resize-none text-[11px]"
              rows={1}
              placeholder="Describe this node..."
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={isLoading}
            />
            <Button
              size="icon"
              variant="ghost"
              className="h-7 w-7 shrink-0"
              onClick={handleSubmit}
              disabled={!input.trim() || isLoading}
            >
              {isLoading ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <SendHorizontal className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
