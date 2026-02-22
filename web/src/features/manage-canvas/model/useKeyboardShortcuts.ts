import { useEffect } from 'react'
import { useUIStore } from '@/entities/ui'

type ShortcutHandlers = {
  onSave: () => void
}

export function useKeyboardShortcuts({ onSave }: ShortcutHandlers) {
  const selectNode = useUIStore((s) => s.selectNode)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const isInput =
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLSelectElement

      // Ctrl/Cmd + S = Save immediately
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        onSave()
        return
      }

      // Ctrl/Cmd + Enter = Switch to Preview tab
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        useUIStore.getState().setForcePreviewTab(true)
        return
      }

      // Skip the rest if user is in an input field
      if (isInput) return

      // Escape = Deselect node
      if (e.key === 'Escape') {
        selectNode(null)
        return
      }

      // Delete/Backspace handled by React Flow's deleteKeyCode
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onSave, selectNode])
}
