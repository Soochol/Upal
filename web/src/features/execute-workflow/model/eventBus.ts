import { createEventBus } from '@/shared/lib/eventBus'
import { useExecutionStore } from '@/entities/run'

export type RunEventBusMap = {
  node_started:   { nodeId: string; startedAt?: number }
  node_completed: { nodeId: string }
  node_error:     { nodeId: string }
  node_skipped:   { nodeId: string }
  node_waiting:   { nodeId: string }
  node_resumed:   { nodeId: string }
  run_done:       { sessionState: Record<string, unknown>; status: string; sessionId: string }
  run_error:      { message: string }
}

export const runEventBus = createEventBus<RunEventBusMap>()

// Wire bus events → executionStore (one-time setup, runs at module load)
runEventBus.on('node_started',   ({ nodeId, startedAt }) => useExecutionStore.getState().setNodeStatus(nodeId, 'running', startedAt))
runEventBus.on('node_completed', ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'completed'))
runEventBus.on('node_error',     ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'error'))
runEventBus.on('node_skipped',   ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'skipped'))
runEventBus.on('node_waiting',   ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'waiting'))
runEventBus.on('node_resumed',   ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'running'))
runEventBus.on('run_done', ({ sessionState }) => useExecutionStore.getState().setSessionState(sessionState))
