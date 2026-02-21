import { useCallback, useRef } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useExecutionStore } from '@/stores/executionStore'
import { startRun, connectToRunEvents } from '@/lib/api'
import { serializeWorkflow } from '@/lib/serializer'
import type { RunEvent } from '@/lib/api'

// Shared event dispatcher — used by both executeRun and reconnectRun.
export function dispatchRunEvent(
  event: RunEvent,
  setNodeStatus: (id: string, status: 'running' | 'completed' | 'error' | 'skipped' | 'waiting', startedAt?: number) => void,
  addRunEvent: (event: RunEvent) => void,
) {
  switch (event.type) {
    case 'node_started':
      setNodeStatus(event.nodeId, 'running', event.startedAt)
      addRunEvent(event)
      break
    case 'tool_call':
    case 'tool_result':
      addRunEvent(event)
      break
    case 'node_completed':
      setNodeStatus(event.nodeId, 'completed')
      addRunEvent(event)
      break
    case 'node_skipped':
      setNodeStatus(event.nodeId, 'skipped')
      addRunEvent(event)
      break
    case 'node_waiting':
      setNodeStatus(event.nodeId, 'waiting')
      addRunEvent(event)
      break
    case 'node_resumed':
      setNodeStatus(event.nodeId, 'running')
      addRunEvent(event)
      break
    default:
      addRunEvent(event)
  }
}

// Shared done handler — used by both executeRun and reconnectRun.
export function handleRunDone(
  result: Record<string, unknown>,
  setNodeStatus: (id: string, status: 'running' | 'completed' | 'error') => void,
  addRunEvent: (event: RunEvent) => void,
  setSessionState: (state: Record<string, unknown>) => void,
  setIsRunning: (running: boolean) => void,
) {
  addRunEvent({
    type: 'done',
    status: result.status as string,
    sessionId: result.session_id as string,
    state: (result.state ?? {}) as Record<string, unknown>,
  })
  const statuses = useExecutionStore.getState().nodeStatuses

  // Mark any still-running nodes as completed
  for (const [id, status] of Object.entries(statuses)) {
    if (status === 'running') {
      setNodeStatus(id, 'completed')
    }
  }

  // If the workflow failed, mark the specific failed node as error
  if (result.status === 'failed') {
    const errorMsg = String(result.error || '')
    const match = errorMsg.match(/node "([^"]+)"/)
    if (match) {
      setNodeStatus(match[1], 'error')
    }
  }

  if (result.state && typeof result.state === 'object') {
    setSessionState(result.state as Record<string, unknown>)
  }
  clearActiveRun()
  setIsRunning(false)
}

// sessionStorage helpers for active run tracking.
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

export function useExecuteRun() {
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const setNodeStatus = useExecutionStore((s) => s.setNodeStatus)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const setSessionState = useExecutionStore((s) => s.setSessionState)
  const abortRef = useRef<AbortController | null>(null)

  const executeRun = useCallback(
    async (inputs: Record<string, string>) => {
      const name = workflowName
      if (!name) return

      // Cancel any previous SSE connection.
      abortRef.current?.abort()

      // Serialize current canvas state to send directly to the backend
      const { nodes, edges } = useWorkflowStore.getState()
      const workflow = serializeWorkflow(name, nodes, edges)

      clearRunEvents()
      clearNodeStatuses()
      setIsRunning(true)

      addRunEvent({ type: 'info', message: `Running workflow "${name}"...` })

      try {
        // 1. Start the run — returns immediately with run_id.
        const { run_id } = await startRun(name, inputs, workflow)
        setActiveRun(run_id, name)

        // 2. Connect to the SSE event stream.
        const abort = new AbortController()
        abortRef.current = abort

        await connectToRunEvents(
          run_id,
          (event) => dispatchRunEvent(event, setNodeStatus, addRunEvent),
          (result) => handleRunDone(result, setNodeStatus, addRunEvent, setSessionState, setIsRunning),
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
    },
    [workflowName, clearRunEvents, clearNodeStatuses, setIsRunning, addRunEvent, setNodeStatus, setSessionState],
  )

  return { executeRun, isRunning }
}
