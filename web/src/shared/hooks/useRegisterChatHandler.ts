import { useEffect } from 'react'
import { useChatBarStore } from '@/entities/ui/model/chatStore'
import type { ChatHandler } from '@/entities/ui/model/chatStore'

export function useRegisterChatHandler(
  handler: ChatHandler | null,
  placeholder: string,
  pageLabel: string,
) {
  const registerHandler = useChatBarStore((s) => s.registerHandler)
  const unregisterHandler = useChatBarStore((s) => s.unregisterHandler)

  useEffect(() => {
    if (handler) {
      registerHandler(handler, placeholder, pageLabel)
    } else {
      unregisterHandler()
    }
    return () => unregisterHandler()
  }, [handler]) // eslint-disable-line react-hooks/exhaustive-deps
}
