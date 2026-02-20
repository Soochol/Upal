import { useCallback } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { useExecutionStore } from '@/stores/executionStore'
import { runWorkflow } from '@/lib/api'
import { serializeWorkflow } from '@/lib/serializer'

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

      // Serialize current canvas state to send directly to the backend
      const { nodes, edges } = useWorkflowStore.getState()
      const workflow = serializeWorkflow(name, nodes, edges)

      clearRunEvents()
      clearNodeStatuses()
      setIsRunning(true)

      addRunEvent({ type: 'info', message: `Running workflow "${name}"...` })

      await runWorkflow(
        name,
        inputs,
        (event) => {
          switch (event.type) {
            case 'node_started':
              setNodeStatus(event.nodeId, 'running')
              addRunEvent(event)
              break
            case 'tool_call':
            case 'tool_result':
              // Log intermediate tool events without changing node status
              addRunEvent(event)
              break
            case 'node_completed':
              setNodeStatus(event.nodeId, 'completed')
              addRunEvent(event)
              break
            default:
              addRunEvent(event)
          }
        },
        (result) => {
          addRunEvent({
            type: 'done',
            status: result.status as string,
            sessionId: result.session_id as string,
            state: (result.state ?? {}) as Record<string, unknown>,
          })
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
          addRunEvent({ type: 'error', message: error.message })
          setIsRunning(false)
        },
        workflow,
      )
    },
    [workflowName, clearRunEvents, clearNodeStatuses, setIsRunning, addRunEvent, setNodeStatus, setSessionState],
  )

  return { executeRun, isRunning }
}
