import { create } from 'zustand'
import type { RunEvent } from '@/shared/types'

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
}

export const useExecutionStore = create<ExecutionState>((set, get) => ({
  isRunning: false,
  runEvents: [],
  sessionState: {},
  nodeStatuses: {},
  nodeStartTimes: {},
  nodeDurations: {},

  setIsRunning: (running) => set({ isRunning: running }),

  addRunEvent: (event) => {
    set({ runEvents: [...get().runEvents, event] })
  },
  clearRunEvents: () => {
    set({ runEvents: [], sessionState: {} })
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
}))
