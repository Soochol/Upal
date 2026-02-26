import { create } from 'zustand'

export type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
}

export type ChatSubmitParams = {
  message: string
  model: string
  thinking: boolean
  history: ChatMessage[]
}

export type ChatHandler = (params: ChatSubmitParams) => Promise<{ explanation: string }>

type ChatBarState = {
  // UI state
  isOpen: boolean
  isLoading: boolean
  messages: ChatMessage[]

  // Handler (registered by each page)
  handler: ChatHandler | null
  placeholder: string
  pageLabel: string

  // Actions
  open: () => void
  close: () => void
  registerHandler: (handler: ChatHandler, placeholder: string, pageLabel: string) => void
  unregisterHandler: () => void
  submit: (message: string) => Promise<void>
}

export const useChatBarStore = create<ChatBarState>((set, get) => ({
  isOpen: false,
  isLoading: false,
  messages: [],
  handler: null,
  placeholder: '',
  pageLabel: '',

  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),

  registerHandler: (handler, placeholder, pageLabel) => {
    set({ handler, placeholder, pageLabel, messages: [], isLoading: false })
  },

  unregisterHandler: () => {
    set({ handler: null, placeholder: '', pageLabel: '', messages: [], isLoading: false, isOpen: false })
  },

  submit: async (message: string) => {
    const { handler, messages, isLoading } = get()
    const trimmed = message.trim()
    if (!handler || isLoading || !trimmed) return

    const userMsg: ChatMessage = { role: 'user', content: trimmed }
    const history = messages.map((m) => ({ role: m.role, content: m.content }))

    set((s) => ({ messages: [...s.messages, userMsg], isLoading: true }))

    try {
      const response = await handler({
        message: trimmed,
        model: '',
        thinking: false,
        history,
      })
      set((s) => ({
        messages: [...s.messages, { role: 'assistant' as const, content: response.explanation || 'Done.' }],
        isLoading: false,
      }))
    } catch (err) {
      set((s) => ({
        messages: [
          ...s.messages,
          {
            role: 'assistant' as const,
            content: err instanceof Error ? err.message : 'An unknown error occurred',
            isError: true,
          },
        ],
        isLoading: false,
      }))
    }
  },
}))
