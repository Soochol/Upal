import { useEffect } from 'react'
import { useChatBarStore } from '@/entities/ui/model/chatStore'
import type { ChatContext } from '@/entities/ui/model/chatStore'

export function useRegisterChatContext(ctx: ChatContext | null) {
  const register = useChatBarStore((s) => s.registerContext)
  const unregister = useChatBarStore((s) => s.unregisterContext)

  useEffect(() => {
    if (ctx) {
      register(ctx)
    } else {
      unregister()
    }
    return () => unregister()
  }, [ctx?.page, ctx?.placeholder, register, unregister])
}
