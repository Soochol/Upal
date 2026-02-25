import type { PipelineSource, PipelineWorkflow, PipelineContext } from '@/entities/pipeline/types'

// --- Model & Tool Discovery ---

export type OptionChoice = {
  label: string
  value: string | number
}

export type OptionSchema = {
  key: string
  label: string
  type: 'slider' | 'number' | 'select'
  min?: number
  max?: number
  step?: number
  default?: unknown
  choices?: OptionChoice[]
}

export type ModelInfo = {
  id: string
  provider: string
  name: string
  category: 'text' | 'image' | 'tts'
  options: OptionSchema[]
  supportsTools: boolean
}

export type ToolInfo = {
  name: string
  description: string
}

// --- Upload ---

export type UploadResult = {
  id: string
  filename: string
  content_type: string
  size: number
  preview_text?: string
}

// --- AI Node Configuration ---

export type ConfigureNodeRequest = {
  node_type: string
  node_id: string
  current_config: Record<string, unknown>
  label: string
  description: string
  message: string
  model?: string
  thinking?: boolean
  history?: { role: string; content: string }[]
  upstream_nodes: { id: string; type: string; label: string }[]
}

export type ConfigureNodeResponse = {
  config: Record<string, unknown>
  label?: string
  description?: string
  explanation: string
}

// --- AI Pipeline Configuration ---

export type ConfigurePipelineRequest = {
  message: string
  model?: string
  thinking?: boolean
  history?: { role: string; content: string }[]
  current_sources: PipelineSource[]
  current_schedule: string
  current_workflows: PipelineWorkflow[]
  current_model: string
  current_context?: PipelineContext
}

export type ConfigurePipelineResponse = {
  sources?: PipelineSource[]
  schedule?: string | null
  workflows?: PipelineWorkflow[]
  model?: string | null
  context?: PipelineContext
  explanation: string
}

// --- Run (re-exported from entity) ---
export type {
  RunRecord, NodeRunRecord, RunListResponse,
  ToolCall, ToolResult, TokenUsage,
  NodeStartedEvent, ToolCallEvent, ToolResultEvent, NodeCompletedEvent,
  NodeSkippedEvent, NodeWaitingEvent, NodeResumedEvent,
  WorkflowDoneEvent, WorkflowErrorEvent, InfoEvent, LogEvent, RunEvent,
} from '@/entities/run/types'

// --- Schedules ---

export type Schedule = {
  id: string
  workflow_name: string
  cron_expr: string
  inputs?: Record<string, unknown>
  enabled: boolean
  timezone: string
  retry_policy?: RetryPolicy
  next_run_at: string
  last_run_at?: string
  created_at: string
  updated_at: string
}

export type RetryPolicy = {
  max_retries: number
  initial_delay: number
  max_delay: number
  backoff_factor: number
}

// --- Triggers ---

export type Trigger = {
  id: string
  workflow_name?: string
  pipeline_id?: string
  type: string
  config: { secret?: string; input_mapping?: Record<string, string> }
  enabled: boolean
  created_at: string
}

// --- Scheduler Stats ---

export type SchedulerStats = {
  concurrency?: { active_runs: number; global_max: number; per_workflow: number }
}

// --- Content Session ---

export type ContentSessionStatus =
  | 'collecting'
  | 'analyzing'
  | 'pending_review'
  | 'approved'
  | 'rejected'
  | 'producing'
  | 'published'
  | 'error'

export type SourceType = 'static' | 'signal'

// --- Pipeline (re-exported from entity) ---
export type {
  PipelineSourceType, PipelineSource, PipelineContext, PipelineWorkflow,
  Pipeline, Stage, CollectSource, StageConfig, PipelineRun, StageResult,
  SessionStage, SessionStatus, PipelineSession,
} from '@/entities/pipeline/types'


export type ConnectionType = 'telegram' | 'slack' | 'http' | 'smtp'

// Connection as returned by API (secrets masked)
export type Connection = {
  id: string
  name: string
  type: ConnectionType
  host?: string
  port?: number
  login?: string
  extras?: Record<string, string>
}

// Connection payload for create/update (includes secrets)
export type ConnectionCreate = {
  name: string
  type: ConnectionType
  host?: string
  port?: number
  login?: string
  password?: string
  token?: string
  extras?: Record<string, string>
}
