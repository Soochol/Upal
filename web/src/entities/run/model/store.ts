import { create } from 'zustand'
import type { RunEvent } from '../types'
import { startRun as apiStartRun, connectToRunEvents } from '../api/index'
import type { WorkflowDefinition } from '@/entities/workflow'

export type NodeRunStatus = 'idle' | 'running' | 'completed' | 'error' | 'waiting' | 'skipped'

type ExecutionState = {
  isRunning: boolean
  setIsRunning: (running: boolean) => void

  runEvents: RunEvent[]
  addRunEvent: (event: RunEvent) => void
  clearRunEvents: () => void

  sessionState: Record<string, unknown>
  setSessionState: (state: Record<string, unknown>) => void

  nodeStatuses: Record<string, NodeRunStatus>
  setNodeStatus: (nodeId: string, status: NodeRunStatus, startedAt?: number) => void
  clearNodeStatuses: () => void

  nodeStartTimes: Record<string, number>
  nodeDurations: Record<string, number>
  runStartTime: number | null
  setRunStartTime: (t: number | null) => void

  startRun: (name: string, inputs: Record<string, string>, workflow?: WorkflowDefinition) => Promise<void>
}

export const useExecutionStore = create<ExecutionState>((set, get) => ({
  isRunning: false,
  runEvents: [],
  sessionState: {},
  nodeStatuses: {},
  nodeStartTimes: {},
  nodeDurations: {},
  runStartTime: null,

  setIsRunning: (running) => set({ isRunning: running }),

  addRunEvent: (event) => {
    set({ runEvents: [...get().runEvents, event] })
  },
  clearRunEvents: () => {
    set({ runEvents: [], sessionState: {}, runStartTime: null })
  },

  setSessionState: (state) => {
    set({ sessionState: state })
  },

  setNodeStatus: (nodeId, status, startedAt?) => {
    const prev = get()
    const updates: Partial<ExecutionState> = {
      nodeStatuses: { ...prev.nodeStatuses, [nodeId]: status },
    }
    if (status === 'running') {
      // Use server timestamp when available (reconnection replays),
      // otherwise fall back to client time (fresh runs).
      updates.nodeStartTimes = { ...prev.nodeStartTimes, [nodeId]: startedAt ?? Date.now() }
    } else if (status === 'completed' || status === 'error') {
      const start = prev.nodeStartTimes[nodeId]
      if (start) {
        updates.nodeDurations = {
          ...prev.nodeDurations,
          [nodeId]: (Date.now() - start) / 1000,
        }
      }
    }
    set(updates)
  },
  clearNodeStatuses: () => {
    set({ nodeStatuses: {}, nodeStartTimes: {}, nodeDurations: {} })
  },

  setRunStartTime: (t) => set({ runStartTime: t }),

  startRun: async (name, inputs, workflow) => {
    const { addRunEvent, setNodeStatus, setSessionState } = get()

    // Reset previous run state
    set({
      runEvents: [],
      sessionState: {},
      nodeStatuses: {},
      nodeStartTimes: {},
      nodeDurations: {},
      runStartTime: Date.now(),
      isRunning: true,
    })

    try {
      const { run_id } = await apiStartRun(name, inputs, workflow)
      addRunEvent({ type: 'info', message: `Started local run: ${run_id}` })

      await connectToRunEvents(
        run_id,
        (event: RunEvent) => {
          addRunEvent(event)

          if (event.type === 'node_started' || event.type === 'tool_call' || event.type === 'tool_result') {
            const nodeId = 'nodeId' in event && event.nodeId ? event.nodeId : 'system'
            setNodeStatus(nodeId, 'running')
          } else if (event.type === 'node_completed') {
            const nodeId = 'nodeId' in event && event.nodeId ? event.nodeId : 'system'
            setNodeStatus(nodeId, 'completed')
          } else if (event.type === 'error') {
            // Workflow/General error
          }
          // Note: The backend may emit custom logical errors, which we log
        },
        (result: Record<string, unknown>) => {
          const stateData = (result.state ?? {}) as Record<string, unknown>
          addRunEvent({
            type: 'done',
            status: (result.status as string) ?? 'success',
            sessionId: (result.session_id as string) ?? run_id,
            state: stateData,
          } as RunEvent)
          setSessionState(stateData)
          set({ isRunning: false })
        },
        (error: Error) => {
          addRunEvent({
            type: 'error',
            message: `Run failed: ${error.message}`
          } as RunEvent)
          set({ isRunning: false })
        }
      )
    } catch (err) {
      addRunEvent({ type: 'error', message: `Failed to start run: ${err instanceof Error ? err.message : String(err)}` })
      set({ isRunning: false })
    }
  },
}))
