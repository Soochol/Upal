import { useEffect } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useUIStore } from '@/stores/uiStore'

type ShortcutHandlers = {
  onSave: () => void
}

export function useKeyboardShortcuts({ onSave }: ShortcutHandlers) {
  const selectNode = useUIStore((s) => s.selectNode)
  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const onNodesChange = useWorkflowStore((s) => s.onNodesChange)

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

      // Delete/Backspace = Delete selected node
      if ((e.key === 'Delete' || e.key === 'Backspace') && selectedNodeId) {
        onNodesChange([{ id: selectedNodeId, type: 'remove' }])
        selectNode(null)
        return
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onSave, selectNode, selectedNodeId, onNodesChange])
}
