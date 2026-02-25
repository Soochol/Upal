// --- Run History ---

export type RunRecord = {
  id: string
  workflow_name: string
  workflow_definition?: {
    name: string
    version: number
    nodes: { id: string; type: string; config: Record<string, unknown>; group?: string }[]
    edges: { from: string; to: string; loop?: { max_iterations: number; exit_when: string }; condition?: string; trigger_rule?: 'on_success' | 'on_failure' | 'always' }[]
    groups?: { id: string; label: string; color?: string }[]
  }
  trigger_type: string
  trigger_ref: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'retrying'
  inputs: Record<string, unknown>
  outputs?: Record<string, unknown>
  error?: string
  retry_of?: string
  retry_count: number
  created_at: string
  started_at?: string
  completed_at?: string
  node_runs?: NodeRunRecord[]
}

export type NodeRunRecord = {
  node_id: string
  status: 'idle' | 'running' | 'completed' | 'error' | 'waiting' | 'skipped'
  started_at: string
  completed_at?: string
  error?: string
  retry_count: number
}

export type RunListResponse = {
  runs: RunRecord[]
  total: number
}

// --- Workflow Execution Events ---

export type ToolCall = { name: string; args?: Record<string, unknown> }
export type ToolResult = { name: string; response?: Record<string, unknown> }

export type NodeStartedEvent = { type: 'node_started'; nodeId: string; startedAt?: number }
export type ToolCallEvent = { type: 'tool_call'; nodeId: string; calls: ToolCall[] }
export type ToolResultEvent = { type: 'tool_result'; nodeId: string; results: ToolResult[] }
export type TokenUsage = {
  input: number
  output: number
  total: number
}

export type NodeCompletedEvent = {
  type: 'node_completed'
  nodeId: string
  output: string
  stateDelta: Record<string, unknown>
  tokens?: TokenUsage
  finishReason?: string
  completedAt?: number
}
export type NodeSkippedEvent = { type: 'node_skipped'; nodeId: string }
export type NodeWaitingEvent = { type: 'node_waiting'; nodeId: string }
export type NodeResumedEvent = { type: 'node_resumed'; nodeId: string }
export type WorkflowDoneEvent = { type: 'done'; status: string; sessionId: string; state: Record<string, unknown>; error?: string }
export type WorkflowErrorEvent = { type: 'error'; message: string }
export type InfoEvent = { type: 'info'; message: string }
export type LogEvent = { type: 'log'; nodeId: string; message: string }

export type RunEvent =
  | NodeStartedEvent | ToolCallEvent | ToolResultEvent
  | NodeCompletedEvent | NodeSkippedEvent | NodeWaitingEvent | NodeResumedEvent
  | WorkflowDoneEvent | WorkflowErrorEvent
  | InfoEvent | LogEvent
