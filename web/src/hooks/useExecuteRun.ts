import { useCallback } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useExecutionStore } from '@/stores/executionStore'
import { runWorkflow } from '@/lib/api'

export function useExecuteRun() {
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const setNodeStatus = useExecutionStore((s) => s.setNodeStatus)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const setSessionState = useExecutionStore((s) => s.setSessionState)

  const executeRun = useCallback(
    async (inputs: Record<string, string>) => {
      const name = workflowName
      if (!name) return

      clearRunEvents()
      clearNodeStatuses()
      setIsRunning(true)
      addRunEvent({ type: 'info', data: { message: `Running workflow "${name}"...` } })

      await runWorkflow(
        name,
        inputs,
        (event) => {
          const nodeId = event.data.Author as string | undefined
          if (nodeId) {
            // Distinguish "started" events (no content) from "completed" events.
            // The backend emits a lightweight event with just Author when a node
            // begins, and a full event with Content/StateDelta when it finishes.
            const actions = event.data.Actions as { StateDelta?: Record<string, unknown> } | undefined
            const content = event.data.Content as { parts?: { text?: string }[] } | undefined
            const hasOutput =
              content?.parts?.some((p) => p?.text) ||
              (actions?.StateDelta && Object.keys(actions.StateDelta).length > 0)

            if (hasOutput) {
              // Node produced output — mark completed and log to console
              setNodeStatus(nodeId, 'completed')
              addRunEvent(event)
            } else {
              // Node just started — show running spinner (don't log bare start events)
              const current = useExecutionStore.getState().nodeStatuses[nodeId]
              if (current !== 'completed') {
                setNodeStatus(nodeId, 'running')
              }
            }
          } else {
            addRunEvent(event)
          }
        },
        (result) => {
          addRunEvent({ type: 'done', data: result })
          const statuses = useExecutionStore.getState().nodeStatuses

          // Mark any still-running nodes as completed (edge case: multi-event nodes)
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
          setIsRunning(false)
        },
        (error) => {
          addRunEvent({ type: 'error', data: { message: error.message } })
          setIsRunning(false)
        },
      )
    },
    [workflowName, clearRunEvents, clearNodeStatuses, setIsRunning, addRunEvent, setNodeStatus, setSessionState],
  )

  return { executeRun, isRunning }
}
