import { useState, useRef, useEffect, useCallback } from 'react'
import { Sparkles, ArrowRight, Loader2, GripVertical, Wrench } from 'lucide-react'
import { useChatBarStore } from '@/entities/ui/model/chatStore'
import type { ChatMessage } from '@/entities/ui/model/chatStore'

function messageBubbleClass(msg: ChatMessage): string {
  const base = 'rounded-2xl px-3 py-2 text-sm'
  if (msg.role === 'user') {
    return `${base} rounded-br-sm bg-primary text-primary-foreground`
  }
  if (msg.isError) {
    return `${base} rounded-bl-sm bg-destructive/10 text-destructive border border-destructive/20`
  }
  return `${base} rounded-bl-sm bg-muted/60 text-foreground`
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

export function GlobalChatBar() {
  const {
    isOpen, isLoading, messages,
    chatContext, position,
    open, close, submit, setPosition,
  } = useChatBarStore()

  const [input, setInput] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  // Drag state
  const isDraggingRef = useRef(false)
  const dragOffsetRef = useRef({ x: 0, y: 0 })

  // Auto-scroll to latest message
  useEffect(() => {
    if (scrollRef.current && isOpen) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isLoading, isOpen])

  // Focus input when opened
  useEffect(() => {
    if (isOpen) inputRef.current?.focus()
  }, [isOpen])

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (isOpen && containerRef.current && !containerRef.current.contains(e.target as Node)) {
        close()
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen, close])

  // Drag handlers
  const handleDragMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    isDraggingRef.current = true

    const container = containerRef.current
    if (!container) return

    const rect = container.getBoundingClientRect()
    dragOffsetRef.current = {
      x: e.clientX - rect.left,
      y: e.clientY - rect.top,
    }
  }, [])

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDraggingRef.current) return
      e.preventDefault()

      const maxX = window.innerWidth - 200
      const maxY = window.innerHeight - 60
      const x = Math.max(0, Math.min(maxX, e.clientX - dragOffsetRef.current.x))
      const y = Math.max(0, Math.min(maxY, e.clientY - dragOffsetRef.current.y))

      const container = containerRef.current
      if (container) {
        container.style.left = `${x}px`
        container.style.top = `${y}px`
        container.style.transform = 'none'
      }
    }

    const handleMouseUp = (e: MouseEvent) => {
      if (!isDraggingRef.current) return
      isDraggingRef.current = false

      const maxX = window.innerWidth - 200
      const maxY = window.innerHeight - 60
      const x = Math.max(0, Math.min(maxX, e.clientX - dragOffsetRef.current.x))
      const y = Math.max(0, Math.min(maxY, e.clientY - dragOffsetRef.current.y))

      setPosition({ x, y })
    }

    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [setPosition])

  const handleSubmit = useCallback(async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading) return
    setInput('')
    open()
    await submit(trimmed)
  }, [input, isLoading, open, submit])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        handleSubmit()
      }
      if (e.key === 'Escape') {
        close()
      }
    },
    [handleSubmit, close],
  )

  // Don't render if no context registered
  if (!chatContext) return null

  const showMessages = isOpen && messages.length > 0
  const pageLabel = capitalize(chatContext.page)

  const containerStyle = position
    ? { left: position.x, top: position.y, transform: 'none' }
    : undefined

  return (
    <div
      ref={containerRef}
      className={[
        'fixed z-50 flex flex-col items-center w-full max-w-xl px-4 pointer-events-none',
        !position ? 'bottom-6 left-1/2 -translate-x-1/2' : '',
      ].join(' ')}
      style={containerStyle}
    >
      {/* Chat messages — visible when expanded */}
      {showMessages && (
        <div className="w-full pointer-events-auto mb-2 animate-in fade-in slide-in-from-bottom-2 duration-200">
          <div
            ref={scrollRef}
            className="w-full max-h-80 overflow-y-auto flex flex-col gap-2 bg-card/95 backdrop-blur-xl
              border border-border/60 rounded-2xl shadow-2xl p-4 scrollbar-thin"
          >
            {messages.map((msg, i) => (
              <div
                key={i}
                className={msg.role === 'user' ? 'self-end max-w-[85%]' : 'self-start max-w-[85%]'}
              >
                <div className={messageBubbleClass(msg)}>
                  {msg.content}
                  {msg.toolCalls?.map((tc) => (
                    <div key={tc.id} className="text-xs text-muted-foreground/70 flex items-center gap-1 mt-1">
                      <Wrench className="h-3 w-3" />
                      <span>{tc.name}</span>
                      {tc.success !== undefined && (
                        <span className={tc.success ? 'text-success' : 'text-destructive'}>
                          {tc.success ? '\u2713' : '\u2717'}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            ))}
            {isLoading && (
              <div className="self-start">
                <div className="rounded-2xl rounded-bl-sm px-3 py-2 text-sm bg-muted/60 text-muted-foreground flex items-center gap-1.5">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Processing...
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Input bar — always visible */}
      <div
        className="pointer-events-auto w-full flex items-center gap-2 rounded-xl border border-border/60
          bg-card/80 backdrop-blur-xl shadow-lg px-4 py-2.5
          transition-all duration-200 focus-within:border-ring/50 focus-within:shadow-xl cursor-text"
        onClick={() => { open(); inputRef.current?.focus() }}
        onMouseEnter={() => { if (messages.length > 0) open() }}
      >
        {/* Drag handle */}
        <div
          onMouseDown={handleDragMouseDown}
          className="shrink-0 cursor-grab active:cursor-grabbing text-muted-foreground/40 hover:text-muted-foreground/70 transition-colors"
        >
          <GripVertical className="h-4 w-4" />
        </div>

        <Sparkles className="h-4 w-4 text-muted-foreground/60 shrink-0" />
        <input
          ref={inputRef}
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={open}
          placeholder={chatContext.placeholder}
          disabled={isLoading}
          className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground/50 outline-none disabled:opacity-50"
        />
        {/* Page label badge */}
        {pageLabel && !isOpen && (
          <span className="text-[10px] text-muted-foreground/40 shrink-0">{pageLabel}</span>
        )}
        {/* Message count badge when collapsed */}
        {!isOpen && messages.length > 0 && (
          <span className="text-[10px] text-muted-foreground/50 shrink-0">{messages.length}</span>
        )}
        <button
          onClick={(e) => { e.stopPropagation(); handleSubmit() }}
          disabled={!input.trim() || isLoading}
          className="shrink-0 h-8 w-8 rounded-lg flex items-center justify-center bg-primary text-primary-foreground
            transition-all duration-150 hover:bg-primary/90 disabled:opacity-30 disabled:pointer-events-none"
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
