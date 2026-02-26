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
  const [expanded, setExpanded] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Collapse on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setExpanded(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Auto-scroll to latest message
  useEffect(() => {
    if (scrollRef.current && expanded) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isLoading, expanded])

  const handleSubmit = useCallback(async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    const history = messages.map((m) => ({ role: m.role, content: m.content }))

    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setIsLoading(true)
    setExpanded(true)

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

  const showMessages = expanded && messages.length > 0

  return (
    <div
      ref={containerRef}
      className="absolute bottom-6 left-1/2 -translate-x-1/2 z-20 flex flex-col items-center w-full max-w-xl px-4"
    >
      {/* Chat messages — visible when expanded */}
      {showMessages && (
        <div
          ref={scrollRef}
          className="w-full max-h-48 overflow-y-auto mb-2 flex flex-col gap-1.5 scrollbar-thin animate-in fade-in slide-in-from-bottom-2 duration-200"
        >
          {messages.map((msg, i) => (
            <div
              key={i}
              className={
                msg.role === 'user'
                  ? 'self-end max-w-[85%]'
                  : 'self-start max-w-[85%]'
              }
            >
              <div
                className={
                  msg.role === 'user'
                    ? 'rounded-lg rounded-br-sm px-3 py-1.5 text-xs bg-primary text-primary-foreground'
                    : msg.isError
                      ? 'rounded-lg rounded-bl-sm px-3 py-1.5 text-xs bg-destructive/10 text-destructive border border-destructive/20'
                      : 'rounded-lg rounded-bl-sm px-3 py-1.5 text-xs bg-card/90 text-muted-foreground border border-border/40'
                }
              >
                {msg.content}
              </div>
            </div>
          ))}
          {isLoading && (
            <div className="self-start max-w-[85%]">
              <div className="rounded-lg rounded-bl-sm px-3 py-1.5 text-xs bg-card/90 text-muted-foreground border border-border/40 flex items-center gap-1.5">
                <Loader2 className="h-3 w-3 animate-spin" />
                Configuring...
              </div>
            </div>
          )}
        </div>
      )}

      {/* Input bar */}
      <div
        className="prompt-bar w-full flex items-center gap-2 rounded-xl border border-border/60 bg-card/80 backdrop-blur-xl shadow-lg px-4 py-2.5 transition-all duration-200 focus-within:border-ring/50 focus-within:shadow-xl cursor-text"
        onClick={() => { setExpanded(true); inputRef.current?.focus() }}
      >
        <Sparkles className="h-4 w-4 text-muted-foreground/60 shrink-0" />
        <input
          ref={inputRef}
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={() => setExpanded(true)}
          placeholder={placeholder}
          disabled={isLoading}
          className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground/50 outline-none disabled:opacity-50"
        />
        {/* Message count badge when collapsed with history */}
        {!expanded && messages.length > 0 && (
          <span className="text-[10px] text-muted-foreground/50 shrink-0">
            {messages.length}
          </span>
        )}
        <button
          onClick={(e) => { e.stopPropagation(); handleSubmit() }}
          disabled={!input.trim() || isLoading}
          className="shrink-0 h-8 w-8 rounded-lg flex items-center justify-center bg-primary text-primary-foreground transition-all duration-150 hover:bg-primary/90 disabled:opacity-30 disabled:pointer-events-none"
        >
          {isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ArrowRight className="h-4 w-4" />
          )}
        </button>
      </div>
    </div>
  )
}
