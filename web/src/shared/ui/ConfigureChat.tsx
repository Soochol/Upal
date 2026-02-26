import { useState, useRef, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { listModels } from '@/shared/api'
import { Button } from '@/shared/ui/button'
import { Textarea } from '@/shared/ui/textarea'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/shared/ui/select'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/shared/ui/collapsible'
import { Sparkles, SendHorizontal, Loader2, ChevronDown, BrainCircuit } from 'lucide-react'
import { cn, groupModelsByProvider } from '@/shared/lib/utils'

export type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
}

type ConfigureChatProps = {
  onSubmit: (params: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => Promise<{ explanation: string }>
  placeholder?: string
  loadingText?: string
}

export function ConfigureChat({
  onSubmit,
  placeholder = 'Describe this node...',
  loadingText = 'Configuring...',
}: ConfigureChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isOpen, setIsOpen] = useState(true)
  const [selectedModel, setSelectedModel] = useState(
    () => localStorage.getItem('upal:ai-model') ?? '',
  )
  const [thinking, setThinking] = useState(
    () => localStorage.getItem('upal:ai-thinking') === 'true',
  )

  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Persist model & thinking to localStorage
  useEffect(() => {
    localStorage.setItem('upal:ai-model', selectedModel)
  }, [selectedModel])
  useEffect(() => {
    localStorage.setItem('upal:ai-thinking', String(thinking))
  }, [thinking])

  // Auto-scroll to latest message
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isLoading])

  const { data: models = [] } = useQuery({
    queryKey: ['models'],
    queryFn: listModels,
    staleTime: 5 * 60 * 1000,
  })

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

      const response = await onSubmit({
        message: trimmed,
        model: selectedModel,
        thinking,
        history,
      })

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
    <Collapsible open={isOpen} onOpenChange={setIsOpen} className="flex flex-col min-h-0">
      <CollapsibleTrigger asChild>
        <button className="flex w-full items-center justify-between px-2 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors shrink-0">
          <span className="flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5" />
            AI Assistant
            {messages.length > 0 && (
              <span className="text-[10px] text-muted-foreground/50">{messages.length}</span>
            )}
          </span>
          <ChevronDown
            className={cn(
              'h-3.5 w-3.5 transition-transform',
              isOpen && 'rotate-180',
            )}
          />
        </button>
      </CollapsibleTrigger>

      <CollapsibleContent className="flex flex-col min-h-0 flex-1">
        <div className="flex flex-col min-h-0 flex-1">
          {/* Chat messages */}
          <div ref={scrollRef} className="flex-1 overflow-y-auto px-2 min-h-0 max-h-48">
            {messages.length > 0 && (
              <div className="flex flex-col gap-1.5 py-1">
                {messages.map((msg, i) => (
                  <div
                    key={i}
                    className={msg.role === 'user' ? 'self-end max-w-[90%]' : 'self-start max-w-[90%]'}
                  >
                    <div
                      className={
                        msg.role === 'user'
                          ? 'rounded-lg rounded-br-sm px-2.5 py-1.5 text-[11px] bg-primary text-primary-foreground'
                          : msg.isError
                            ? 'rounded-lg rounded-bl-sm px-2.5 py-1.5 text-[11px] bg-destructive/10 text-destructive'
                            : 'rounded-lg rounded-bl-sm px-2.5 py-1.5 text-[11px] bg-muted/50 text-foreground'
                      }
                    >
                      {msg.content}
                    </div>
                  </div>
                ))}
                {isLoading && (
                  <div className="self-start">
                    <div className="rounded-lg rounded-bl-sm px-2.5 py-1.5 text-[11px] bg-muted/50 text-muted-foreground flex items-center gap-1.5">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      {loadingText}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Model selector + thinking toggle + input */}
          <div className="px-2 pb-2 pt-1 space-y-1.5 shrink-0">
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

            <div className="flex items-end gap-1">
              <Textarea
                ref={textareaRef}
                className="min-h-[32px] max-h-[60px] resize-none text-[11px]"
                rows={1}
                placeholder={placeholder}
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
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
