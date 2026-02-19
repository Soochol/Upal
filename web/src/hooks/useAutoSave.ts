import { useState, useEffect, useRef, useCallback } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useExecutionStore } from '@/stores/executionStore'
import { serializeWorkflow } from '@/lib/serializer'
import { saveWorkflow, suggestWorkflowName } from '@/lib/api'

const DEBOUNCE_MS = 2000

export type SaveStatus = 'idle' | 'waiting' | 'saving' | 'saved' | 'error'

export function useAutoSave() {
  const [saveStatus, setSaveStatus] = useState<SaveStatus>('idle')
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isSavingRef = useRef(false)
  const lastSnapshotRef = useRef('')
  const savedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const performSave = useCallback(async () => {
    // Clear any pending "Saved" dismiss timer
    if (savedTimerRef.current) clearTimeout(savedTimerRef.current)

    const { nodes, edges, workflowName, setWorkflowName } =
      useWorkflowStore.getState()
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
      let name = workflowName

      // If unnamed, ask LLM for a name
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
      await saveWorkflow(wf)

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
