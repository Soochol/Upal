import { useCallback, useRef } from 'react'
import { useWorkflowStore, serializeWorkflow } from '@/entities/workflow'
import { useExecutionStore, startRun, connectToRunEvents } from '@/entities/run'
import type { RunEvent } from '@/shared/types'
import { runEventBus } from './eventBus'

// sessionStorage helpers
export function setActiveRun(runId: string, workflowName: string) {
  sessionStorage.setItem('active_run_id', runId)
  sessionStorage.setItem('active_run_workflow', workflowName)
}
export function getActiveRun(): { runId: string; workflowName: string } | null {
  const runId = sessionStorage.getItem('active_run_id')
  const workflowName = sessionStorage.getItem('active_run_workflow')
  if (!runId) return null
  return { runId, workflowName: workflowName || '' }
}
export function clearActiveRun() {
  sessionStorage.removeItem('active_run_id')
  sessionStorage.removeItem('active_run_workflow')
}

/** Translate a raw RunEvent from SSE into runEventBus emissions + raw event storage */
export function dispatchRunEvent(event: RunEvent, addRunEvent: (e: RunEvent) => void) {
  addRunEvent(event)
  switch (event.type) {
    case 'node_started':   runEventBus.emit('node_started',   { nodeId: event.nodeId, startedAt: event.startedAt }); break
    case 'node_completed': runEventBus.emit('node_completed', { nodeId: event.nodeId }); break
    case 'node_skipped':   runEventBus.emit('node_skipped',   { nodeId: event.nodeId }); break
    case 'node_waiting':   runEventBus.emit('node_waiting',   { nodeId: event.nodeId }); break
    case 'node_resumed':   runEventBus.emit('node_resumed',   { nodeId: event.nodeId }); break
  }
}

export function useExecuteRun() {
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const setSessionState = useExecutionStore((s) => s.setSessionState)
  const abortRef = useRef<AbortController | null>(null)

  const executeRun = useCallback(async (inputs: Record<string, string>) => {
    const name = workflowName
    if (!name) return
    abortRef.current?.abort()
    const { nodes, edges } = useWorkflowStore.getState()
    const workflow = serializeWorkflow(name, nodes, edges)
    clearRunEvents()
    clearNodeStatuses()
    setIsRunning(true)
    addRunEvent({ type: 'info', message: `Running workflow "${name}"...` })
    try {
      const { run_id } = await startRun(name, inputs, workflow)
      setActiveRun(run_id, name)
      const abort = new AbortController()
      abortRef.current = abort
      await connectToRunEvents(
        run_id,
        (event) => dispatchRunEvent(event, addRunEvent),
        (result) => {
          addRunEvent({ type: 'done', status: result.status as string, sessionId: result.session_id as string, state: (result.state ?? {}) as Record<string, unknown> })
          const statuses = useExecutionStore.getState().nodeStatuses
          for (const [id, status] of Object.entries(statuses)) {
            if (status === 'running') runEventBus.emit('node_completed', { nodeId: id })
          }
          if (result.status === 'failed') {
            const match = String(result.error || '').match(/node "([^"]+)"/)
            if (match) runEventBus.emit('node_error', { nodeId: match[1] })
          }
          if (result.state && typeof result.state === 'object') {
            runEventBus.emit('run_done', { sessionState: result.state as Record<string, unknown>, status: result.status as string, sessionId: result.session_id as string })
          }
          clearActiveRun()
          setIsRunning(false)
        },
        (error) => {
          addRunEvent({ type: 'error', message: error.message })
          clearActiveRun()
          setIsRunning(false)
        },
        { signal: abort.signal },
      )
    } catch (err) {
      addRunEvent({ type: 'error', message: err instanceof Error ? err.message : String(err) })
      clearActiveRun()
      setIsRunning(false)
    }
  }, [workflowName, clearRunEvents, clearNodeStatuses, setIsRunning, addRunEvent, setSessionState])

  return { executeRun, isRunning }
}
