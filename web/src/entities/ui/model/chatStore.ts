import { create } from 'zustand'
import { API_BASE } from '@/shared/api/client'
import { useAuthStore } from '@/entities/auth/store'

// Types

export type ChatContext = {
  page: string
  context: Record<string, unknown>
  applyResult: (toolName: string, result: unknown) => void
  placeholder: string
}

export type ToolCallInfo = {
  id: string
  name: string
  args: unknown
  result?: unknown
  success?: boolean
}

export type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
  toolCalls?: ToolCallInfo[]
}

type ChatBarState = {
  isOpen: boolean
  isLoading: boolean
  messages: ChatMessage[]
  chatContext: ChatContext | null

  // Draggable position (null = default center)
  position: { x: number; y: number } | null

  // Actions
  registerContext: (ctx: ChatContext) => void
  unregisterContext: () => void
  open: () => void
  close: () => void
  submit: (message: string) => Promise<void>
  setPosition: (pos: { x: number; y: number }) => void
  clearMessages: () => void
}

function updateLastAssistantMessage(
  messages: ChatMessage[],
  updater: (msg: ChatMessage) => ChatMessage,
): ChatMessage[] {
  const result = [...messages]
  for (let i = result.length - 1; i >= 0; i--) {
    if (result[i].role === 'assistant') {
      result[i] = updater(result[i])
      break
    }
  }
  return result
}

export const useChatBarStore = create<ChatBarState>((set, get) => ({
  isOpen: false,
  isLoading: false,
  messages: [],
  chatContext: null,
  position: null,

  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),

  registerContext: (ctx) => {
    const savedPos = localStorage.getItem(`chatbar-pos-${ctx.page}`)
    set({
      chatContext: ctx,
      position: savedPos ? JSON.parse(savedPos) : null,
      messages: [],
    })
  },

  unregisterContext: () => {
    set({
      chatContext: null,
      messages: [],
      isLoading: false,
      isOpen: false,
      position: null,
    })
  },

  setPosition: (pos) => {
    const page = get().chatContext?.page
    if (page) {
      localStorage.setItem(`chatbar-pos-${page}`, JSON.stringify(pos))
    }
    set({ position: pos })
  },

  clearMessages: () => set({ messages: [] }),

  submit: async (message: string) => {
    const { chatContext, messages, isLoading } = get()
    const trimmed = message.trim()
    if (!chatContext || isLoading || !trimmed) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    const assistantMsg: ChatMessage = { role: 'assistant', content: '' }
    const history = messages
      .filter((m) => !m.isError)
      .map((m) => ({ role: m.role, content: m.content }))

    set((s) => ({
      messages: [...s.messages, userMsg, assistantMsg],
      isLoading: true,
    }))

    const payload = {
      message: trimmed,
      page: chatContext.page,
      context: chatContext.context,
      history,
    }

    try {
      const token = useAuthStore.getState().token
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }

      const response = await fetch(`${API_BASE}/chat`, {
        method: 'POST',
        headers,
        body: JSON.stringify(payload),
      })

      if (!response.ok) {
        const text = await response.text().catch(() => response.statusText)
        throw new Error(text || response.statusText)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('No response body')
      }

      const decoder = new TextDecoder()
      let buffer = ''
      let currentEvent = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })

        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''

        for (const line of lines) {
          const trimmedLine = line.replace(/\r$/, '')

          if (trimmedLine.startsWith('event: ')) {
            currentEvent = trimmedLine.slice(7).trim()
          } else if (trimmedLine.startsWith('data: ')) {
            const dataStr = trimmedLine.slice(6)
            try {
              const data = JSON.parse(dataStr)
              handleSSEEvent(currentEvent, data, get, set)
            } catch {
              // Non-JSON data line, skip
            }
            currentEvent = ''
          } else if (trimmedLine === '') {
            currentEvent = ''
          }
        }
      }

      // Stream ended — ensure loading is off
      set({ isLoading: false })
    } catch (err) {
      set((s) => ({
        messages: updateLastAssistantMessage(s.messages, (msg) => ({
          ...msg,
          content: msg.content || (err instanceof Error ? err.message : 'An unknown error occurred'),
          isError: true,
        })),
        isLoading: false,
      }))
    }
  },
}))

function handleSSEEvent(
  eventType: string,
  data: Record<string, unknown>,
  get: () => ChatBarState,
  set: (updater: Partial<ChatBarState> | ((s: ChatBarState) => Partial<ChatBarState>)) => void,
) {
  switch (eventType) {
    case 'text_delta': {
      const chunk = (data.text ?? data.content ?? '') as string
      set((s) => ({
        messages: updateLastAssistantMessage(s.messages, (msg) => ({
          ...msg,
          content: msg.content + chunk,
        })),
      }))
      break
    }

    case 'tool_call': {
      const toolCall: ToolCallInfo = {
        id: data.id as string,
        name: data.name as string,
        args: data.args,
      }
      set((s) => ({
        messages: updateLastAssistantMessage(s.messages, (msg) => ({
          ...msg,
          toolCalls: [...(msg.toolCalls ?? []), toolCall],
        })),
      }))
      break
    }

    case 'tool_result': {
      const toolId = data.id as string
      const toolName = data.name as string
      const result = data.result
      const success = data.success as boolean | undefined

      set((s) => ({
        messages: updateLastAssistantMessage(s.messages, (msg) => ({
          ...msg,
          toolCalls: (msg.toolCalls ?? []).map((tc) =>
            tc.id === toolId ? { ...tc, result, success } : tc,
          ),
        })),
      }))

      // Apply the result via the context callback
      const ctx = get().chatContext
      if (ctx) {
        ctx.applyResult(toolName, result)
      }
      break
    }

    case 'done': {
      set({ isLoading: false })
      break
    }

    case 'error': {
      const errorMsg = (data.error ?? data.message ?? 'Unknown error') as string
      set((s) => ({
        messages: updateLastAssistantMessage(s.messages, (msg) => ({
          ...msg,
          content: msg.content || errorMsg,
          isError: true,
        })),
        isLoading: false,
      }))
      break
    }
  }
}
