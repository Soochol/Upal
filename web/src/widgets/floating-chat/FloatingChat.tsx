import { useState, useRef, useEffect, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { MessageSquare, X, SendHorizontal, Loader2, Check, AlertCircle, Sparkles, Plus } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { fetchContentSession, updateSessionSettings } from '@/entities/content-session/api'
import { configurePipeline } from '@/features/configure-pipeline/api'
import type { PipelineSource, PipelineWorkflow, PipelineContext, CreatedWorkflowInfo } from '@/shared/types'

type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
  createdWorkflows?: CreatedWorkflowInfo[]
}

interface FloatingChatProps {
  pipelineId?: string
  sessionId?: string
}

export function FloatingChat({ pipelineId, sessionId }: FloatingChatProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const queryClient = useQueryClient()
  const scrollRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const { data: session } = useQuery({
    queryKey: ['content-session', sessionId],
    queryFn: () => fetchContentSession(sessionId!),
    enabled: !!sessionId,
  })

  const invalidate = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['content-session', sessionId] })
    queryClient.invalidateQueries({ queryKey: ['content-sessions', { pipelineId }] })
  }, [queryClient, sessionId, pipelineId])

  // Auto-scroll to bottom on new messages or loading
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, isLoading])

  // Focus input when opening
  useEffect(() => {
    if (isOpen && textareaRef.current) {
      textareaRef.current.focus()
    }
  }, [isOpen])

  const handleSubmit = async () => {
    const trimmed = input.trim()
    if (!trimmed || isLoading || !session || !sessionId) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    const nextMessages = [...messages, userMsg]
    setMessages(nextMessages)
    setInput('')
    setIsLoading(true)

    try {
      const history = messages.map((m) => ({ role: m.role, content: m.content }))
      const effectivePipelineId = pipelineId ?? session.pipeline_id

      const response = await configurePipeline(effectivePipelineId, {
        message: trimmed,
        history,
        current_sources: session.session_sources ?? [],
        current_schedule: session.schedule ?? '',
        current_workflows: session.session_workflows ?? [],
        current_model: session.model ?? '',
        current_context: session.context,
      })

      // Apply changes to session
      const settings: Record<string, unknown> = {}
      if (response.sources) settings.sources = response.sources
      if (response.schedule !== undefined && response.schedule !== null) settings.schedule = response.schedule
      if (response.workflows) settings.workflows = response.workflows
      if (response.model !== undefined && response.model !== null) settings.model = response.model
      if (response.context) settings.context = response.context

      if (Object.keys(settings).length > 0) {
        await updateSessionSettings(sessionId, settings as {
          sources?: PipelineSource[]
          schedule?: string
          workflows?: PipelineWorkflow[]
          model?: string
          context?: PipelineContext
        })
        invalidate()
      }

      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: response.explanation || 'Settings updated.',
          createdWorkflows: response.created_workflows,
        },
      ])
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'An unknown error occurred'
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

  if (!sessionId) return null

  return (
    <>
      {/* Backdrop */}
      {isOpen && (
        <div className="fixed inset-0 z-40" onClick={() => setIsOpen(false)} />
      )}

      {/* Panel */}
      <div
        className={cn(
          'fixed bottom-20 right-6 z-50 w-[400px] h-[500px]',
          'bg-card border border-border rounded-2xl shadow-2xl',
          'flex flex-col overflow-hidden',
          'transition-all duration-200 ease-out origin-bottom-right',
          isOpen
            ? 'scale-100 opacity-100 pointer-events-auto'
            : 'scale-95 opacity-0 pointer-events-none',
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-background/80 backdrop-blur-sm shrink-0">
          <span className="text-sm font-semibold flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5 text-muted-foreground" />
            AI Assistant
          </span>
          <button
            onClick={() => setIsOpen(false)}
            className="text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Message log */}
        <div ref={scrollRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-3">
          {messages.length === 0 && !isLoading && (
            <div className="flex flex-col items-center justify-center h-full text-center text-muted-foreground/50 gap-2">
              <MessageSquare className="h-8 w-8" />
              <p className="text-xs">Change session settings through conversation</p>
            </div>
          )}

          {messages.map((msg, i) =>
            msg.role === 'user' ? (
              <div key={i} className="text-[13px]">
                <span className="text-muted-foreground font-medium">You:</span>{' '}
                <span className="text-foreground">{msg.content}</span>
              </div>
            ) : (
              <div key={i} className="space-y-1">
                <div
                  className={cn(
                    'flex items-start gap-1.5 text-[13px]',
                    msg.isError ? 'text-destructive' : 'text-muted-foreground',
                  )}
                >
                  {msg.isError ? (
                    <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
                  ) : (
                    <Check className="h-3.5 w-3.5 mt-0.5 shrink-0 text-success" />
                  )}
                  <span>{msg.content}</span>
                </div>
                {msg.createdWorkflows?.map((cw) => (
                  <div
                    key={cw.name}
                    className={cn(
                      'flex items-center gap-1.5 text-[12px] ml-5',
                      cw.status === 'failed' ? 'text-destructive' : 'text-success',
                    )}
                  >
                    <Plus className="h-3 w-3 shrink-0" />
                    <span>
                      Workflow &lsquo;{cw.name}&rsquo;
                      {cw.status === 'success' && ' created'}
                      {cw.status === 'exists' && ' (already exists)'}
                      {cw.status === 'failed' && ` failed${cw.error ? `: ${cw.error}` : ''}`}
                    </span>
                  </div>
                ))}
              </div>
            ),
          )}

          {isLoading && (
            <div className="flex items-center gap-1.5 text-muted-foreground">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              <span className="text-[13px]">Updating settings...</span>
            </div>
          )}
        </div>

        {/* Input */}
        <div className="shrink-0 border-t border-border px-3 py-2.5">
          <div className="flex items-end gap-2">
            <textarea
              ref={textareaRef}
              className="flex-1 min-h-[36px] max-h-[80px] resize-none text-[13px] bg-transparent
                outline-none placeholder:text-muted-foreground/40 py-1.5"
              rows={1}
              placeholder="Ask to change settings..."
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={isLoading || !session}
            />
            <button
              onClick={handleSubmit}
              disabled={!input.trim() || isLoading || !session}
              className="h-8 w-8 shrink-0 rounded-lg flex items-center justify-center
                text-muted-foreground hover:text-foreground hover:bg-muted/50
                disabled:opacity-30 disabled:cursor-not-allowed
                transition-colors cursor-pointer"
            >
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <SendHorizontal className="h-4 w-4" />
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Floating toggle */}
      <button
        onClick={() => setIsOpen((v) => !v)}
        className={cn(
          'fixed bottom-6 right-6 z-50',
          'h-12 w-12 rounded-full',
          'bg-primary text-primary-foreground',
          'shadow-lg hover:shadow-xl',
          'flex items-center justify-center',
          'transition-all duration-200 cursor-pointer',
          'hover:scale-105 active:scale-95',
        )}
      >
        {isLoading && !isOpen && (
          <span className="absolute inset-0 rounded-full animate-ping bg-primary/30" />
        )}
        {isOpen ? (
          <X className="h-5 w-5" />
        ) : isLoading ? (
          <Loader2 className="h-5 w-5 animate-spin" />
        ) : (
          <MessageSquare className="h-5 w-5" />
        )}
      </button>
    </>
  )
}
