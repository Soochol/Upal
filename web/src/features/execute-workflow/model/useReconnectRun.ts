import { useEffect, useRef } from 'react'
import { useExecutionStore } from '@/entities/run'
import { useWorkflowStore } from '@/entities/workflow'
import { connectToRunEvents, fetchWorkflowRuns } from '@/entities/run'
import {
  dispatchRunEvent,
  getActiveRun,
  setActiveRun,
  clearActiveRun,
} from './useExecuteRun'
import { runEventBus } from './eventBus'

// useReconnectRun checks for an active background run on mount and
// reconnects to its SSE event stream. It uses two strategies:
// 1. sessionStorage (fastest — available if the same tab refreshes)
// 2. Runs API fallback (covers the case where sessionStorage was lost)
export function useReconnectRun() {
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    let cancelled = false

    async function tryReconnect() {
      // Strategy 1: sessionStorage has the run_id
      const active = getActiveRun()
      if (active) {
        reconnect(active.runId)
        return
      }

      // Strategy 2: query the runs API for the current workflow
      const workflowName = useWorkflowStore.getState().workflowName
      if (!workflowName) return

      try {
        const { runs } = await fetchWorkflowRuns(workflowName, 5, 0)
        const running = runs?.find((r) => r.status === 'running')
        if (running && !cancelled) {
          setActiveRun(running.id, workflowName)
          reconnect(running.id)
        }
      } catch {
        // API unavailable — nothing to reconnect to
      }
    }

    function reconnect(runId: string) {
      if (cancelled) return

      // Cancel any previous connection.
      abortRef.current?.abort()
      const abort = new AbortController()
      abortRef.current = abort

      setIsRunning(true)
      addRunEvent({ type: 'info', message: 'Reconnecting to in-progress run...' })

      connectToRunEvents(
        runId,
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
    }

    tryReconnect()

    return () => {
      cancelled = true
      abortRef.current?.abort()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
}
