import { useEffect } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'

type ShortcutHandlers = {
  onSave: () => void
  onRun: () => void
  onGenerate: () => void
}

export function useKeyboardShortcuts({ onSave, onRun, onGenerate }: ShortcutHandlers) {
  const selectNode = useWorkflowStore((s) => s.selectNode)
  const selectedNodeId = useWorkflowStore((s) => s.selectedNodeId)
  const onNodesChange = useWorkflowStore((s) => s.onNodesChange)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const isInput =
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLSelectElement

      // Ctrl/Cmd + S = Save
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        onSave()
        return
      }

      // Ctrl/Cmd + Enter = Run
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        onRun()
        return
      }

      // Ctrl/Cmd + G = Generate
      if ((e.ctrlKey || e.metaKey) && e.key === 'g') {
        e.preventDefault()
        onGenerate()
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
  }, [onSave, onRun, onGenerate, selectNode, selectedNodeId, onNodesChange])
}
