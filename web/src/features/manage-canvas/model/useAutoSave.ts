import { useState, useEffect, useRef, useCallback } from 'react'
import { useWorkflowStore, serializeWorkflow, saveWorkflow, suggestWorkflowName } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'

const DEBOUNCE_MS = 2000

export type SaveStatus = 'idle' | 'waiting' | 'saving' | 'saved' | 'error'

/**
 * Save the current canvas state to the backend immediately.
 * Reads directly from Zustand stores — safe to call outside React components.
 */
async function flushSave(): Promise<void> {
  const { nodes, edges, workflowName, originalName, setWorkflowName, setOriginalName } =
    useWorkflowStore.getState()

  if (nodes.length === 0) return

  let name = workflowName
  if (!name) {
    const tempWf = serializeWorkflow('untitled', nodes, edges)
    try {
      name = await suggestWorkflowName(tempWf)
    } catch {
      name = 'untitled-workflow'
    }
    setWorkflowName(name)
  }

  const wf = serializeWorkflow(name, nodes, edges)
  await saveWorkflow(wf, originalName || undefined)

  // After successful save, sync originalName to current name
  if (originalName !== name) {
    setOriginalName(name)
  }
}

export function useAutoSave() {
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isSavingRef = useRef(false)
  const lastSnapshotRef = useRef('')
  const savedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const performSave = useCallback(async () => {
    // Clear any pending "Saved" dismiss timer
    if (savedTimerRef.current) clearTimeout(savedTimerRef.current)

    const { nodes, edges, workflowName } = useWorkflowStore.getState()
    const { isRunning } = useExecutionStore.getState()

    // Don't save during execution or with empty canvas
    if (isRunning || nodes.length === 0) return

    // Snapshot check: skip if nothing changed
    const snapshot = JSON.stringify({ nodes: nodes.map(n => ({ id: n.id, data: n.data, position: n.position })), edges, workflowName })
    if (snapshot === lastSnapshotRef.current) return

    if (isSavingRef.current) return
    isSavingRef.current = true
    setSaveStatus('saving')

    try {
      await flushSave()

      lastSnapshotRef.current = snapshot
      setSaveStatus('saved')
      // Auto-dismiss "Saved" after 2 seconds
      savedTimerRef.current = setTimeout(() => setSaveStatus('idle'), 2000)
    } catch {
      setSaveStatus('error')
    } finally {
      isSavingRef.current = false
    }
  }, [])

  const debouncedSave = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setSaveStatus('waiting')
    timerRef.current = setTimeout(performSave, DEBOUNCE_MS)
  }, [performSave])

  // Immediate save (for Ctrl+S)
  const saveNow = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    performSave()
  }, [performSave])

  useEffect(() => {
    const unsub = useWorkflowStore.subscribe(
      (state, prevState) => {
        if (
          state.nodes !== prevState.nodes ||
          state.edges !== prevState.edges ||
          state.workflowName !== prevState.workflowName
        ) {
          debouncedSave()
        }
      },
    )
    return () => {
      unsub()
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [debouncedSave])

  return { saveStatus, saveNow }
}
