import { useState, useRef, useEffect } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'
import { configureNode, listModels, type ModelInfo } from '@/lib/api'
import { useUpstreamNodes } from '@/hooks/useUpstreamNodes'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Sparkles, SendHorizontal, Loader2, ChevronDown, Check, AlertCircle, BrainCircuit } from 'lucide-react'
import { cn, groupModelsByProvider } from '@/lib/utils'

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
  const [models, setModels] = useState<ModelInfo[]>([])
  const [selectedModel, setSelectedModel] = useState(
    () => localStorage.getItem('upal:ai-model') ?? '',
  )
  const [thinking, setThinking] = useState(
    () => localStorage.getItem('upal:ai-thinking') === 'true',
  )

  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Persist model & thinking to localStorage
  useEffect(() => {
    localStorage.setItem('upal:ai-model', selectedModel)
  }, [selectedModel])
  useEffect(() => {
    localStorage.setItem('upal:ai-thinking', String(thinking))
  }, [thinking])

  // Fetch available models
  useEffect(() => {
    listModels().then(setModels).catch(() => {})
  }, [])

  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)
  const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)
  const upstreamNodes = useUpstreamNodes(nodeId)

  // Derive the last assistant message for inline status display
  const lastStatus = [...messages].reverse().find((m) => m.role === 'assistant') ?? null

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
        model: selectedModel || undefined,
        thinking,
        history,
        upstream_nodes: upstreamNodes,
      })

      // Apply returned config updates
      if (response.config && Object.keys(response.config).length > 0) {
        updateNodeConfig(nodeId, response.config)
      }
      if (response.label) {
        updateNodeLabel(nodeId, response.label)
      }
      const desc = response.description || (response.config?.description as string)
      if (desc) {
        updateNodeDescription(nodeId, desc)
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
          {/* Model selector + thinking toggle */}
          <div className="flex items-center gap-0.5">
            {models.length > 0 && (
              <Select value={selectedModel} onValueChange={setSelectedModel}>
                <SelectTrigger className="h-5 text-[10px] w-fit border-none shadow-none px-1.5 py-0 gap-1 text-muted-foreground hover:text-foreground">
                  <SelectValue placeholder="Auto" />
                </SelectTrigger>
                <SelectContent position="popper" side="top" sideOffset={4}>
                  {Object.entries(groupModelsByProvider(models)).map(([provider, items]) => (
                    <SelectGroup key={provider}>
                      <SelectLabel className="text-[10px]">{provider}</SelectLabel>
                      {items.map((m) => (
                        <SelectItem key={m.id} value={m.id} className="text-[11px]">
                          {m.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  ))}
                </SelectContent>
              </Select>
            )}
            <button
              type="button"
              onClick={() => setThinking((v) => !v)}
              className={cn(
                'flex items-center gap-0.5 rounded px-1 py-0.5 text-[10px] transition-colors',
                thinking
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:text-foreground',
              )}
              title={thinking ? 'Thinking enabled' : 'Thinking disabled'}
            >
              <BrainCircuit className="h-3 w-3" />
            </button>
          </div>

          {/* Status: last result or loading */}
          {isLoading && (
            <div className="flex items-center gap-1.5 text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span className="text-[10px]">Configuring...</span>
            </div>
          )}
          {!isLoading && lastStatus && (
            <div
              className={cn(
                'flex items-start gap-1.5 text-[10px]',
                lastStatus.isError
                  ? 'text-destructive'
                  : 'text-muted-foreground',
              )}
            >
              {lastStatus.isError ? (
                <AlertCircle className="h-3 w-3 mt-px shrink-0" />
              ) : (
                <Check className="h-3 w-3 mt-px shrink-0" />
              )}
              <span className="truncate">{lastStatus.content}</span>
            </div>
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
