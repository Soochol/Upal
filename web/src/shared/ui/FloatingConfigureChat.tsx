import { useState, useRef, useEffect, useCallback } from 'react'
import { Sparkles, ArrowRight, Loader2 } from 'lucide-react'
import type { ChatMessage } from './ConfigureChat'

type Props = {
  onSubmit: (params: {
    message: string
    model: string
    thinking: boolean
    history: ChatMessage[]
  }) => Promise<{ explanation: string }>
  placeholder?: string
}

export function FloatingConfigureChat({
  onSubmit,
  placeholder = 'Describe your session settings...',
}: Props) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to latest message
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isLoading])

  const handleSubmit = useCallback(async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    const history = messages.map((m) => ({ role: m.role, content: m.content }))

    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)

    try {
      const response = await onSubmit({
        message: trimmed,
        model: '',
        thinking: false,
        history,
      })
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: response.explanation || 'Settings updated.' },
      ])
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: err instanceof Error ? err.message : 'An unknown error occurred',
          isError: true,
        },
      ])
    } finally {
      setIsLoading(false)
    }
  }, [input, isLoading, messages, onSubmit])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSubmit()
      }
    },
    [handleSubmit],
  )

  return (
    <div className="shrink-0 flex flex-col border-t border-border/50 h-64">
      {/* Chat messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 min-h-0">
        {messages.length === 0 ? (
          <div className="flex items-center justify-center h-full">
            <span className="text-xs text-muted-foreground/40">AI에게 세션 설정을 요청하세요</span>
          </div>
        ) : (
          <div className="flex flex-col gap-2 max-w-xl mx-auto">
            {messages.map((msg, i) => (
              <div
                key={i}
                className={msg.role === 'user' ? 'self-end max-w-[85%]' : 'self-start max-w-[85%]'}
              >
                <div
                  className={
                    msg.role === 'user'
                      ? 'rounded-2xl rounded-br-sm px-3 py-2 text-sm bg-primary text-primary-foreground'
                      : msg.isError
                        ? 'rounded-2xl rounded-bl-sm px-3 py-2 text-sm bg-destructive/10 text-destructive border border-destructive/20'
                        : 'rounded-2xl rounded-bl-sm px-3 py-2 text-sm bg-muted/50 text-foreground'
                  }
                >
                  {msg.content}
                </div>
              </div>
            ))}
            {isLoading && (
              <div className="self-start">
                <div className="rounded-2xl rounded-bl-sm px-3 py-2 text-sm bg-muted/50 text-muted-foreground flex items-center gap-1.5">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Configuring...
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Input bar */}
      <div className="shrink-0 px-4 py-3 border-t border-border/30">
        <div className="flex items-center gap-2 max-w-xl mx-auto rounded-xl border border-border/60 bg-card/80 px-3 py-2 focus-within:border-ring/50 transition-colors">
          <Sparkles className="h-4 w-4 text-muted-foreground/60 shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={isLoading}
            className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground/50 outline-none disabled:opacity-50"
          />
          <button
            onClick={handleSubmit}
            disabled={!input.trim() || isLoading}
            className="shrink-0 h-7 w-7 rounded-lg flex items-center justify-center bg-primary text-primary-foreground transition-all duration-150 hover:bg-primary/90 disabled:opacity-30 disabled:pointer-events-none"
          >
            {isLoading ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <ArrowRight className="h-3.5 w-3.5" />
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
