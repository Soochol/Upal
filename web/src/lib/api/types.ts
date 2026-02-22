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
  category: 'text' | 'image'
  options: OptionSchema[]
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

// --- Run History ---

export type RunRecord = {
  id: string
  workflow_name: string
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
  status: string
  started_at: string
  completed_at?: string
  error?: string
  retry_count: number
}

export type RunListResponse = {
  runs: RunRecord[]
  total: number
}

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

// --- Workflow Execution Events ---

export type ToolCall = { name: string; args?: Record<string, unknown> }
export type ToolResult = { name: string; response?: Record<string, unknown> }

export type NodeStartedEvent   = { type: 'node_started';   nodeId: string; startedAt?: number }
export type ToolCallEvent      = { type: 'tool_call';      nodeId: string; calls: ToolCall[] }
export type ToolResultEvent    = { type: 'tool_result';    nodeId: string; results: ToolResult[] }
export type NodeCompletedEvent = { type: 'node_completed'; nodeId: string; output: string; stateDelta: Record<string, unknown> }
export type NodeSkippedEvent  = { type: 'node_skipped';  nodeId: string }
export type NodeWaitingEvent  = { type: 'node_waiting';  nodeId: string }
export type NodeResumedEvent  = { type: 'node_resumed';  nodeId: string }
export type WorkflowDoneEvent  = { type: 'done';           status: string; sessionId: string; state: Record<string, unknown> }
export type WorkflowErrorEvent = { type: 'error';          message: string }
export type InfoEvent          = { type: 'info';           message: string }
export type LogEvent           = { type: 'log';            nodeId: string; message: string }

export type RunEvent =
  | NodeStartedEvent | ToolCallEvent | ToolResultEvent
  | NodeCompletedEvent | NodeSkippedEvent | NodeWaitingEvent | NodeResumedEvent
  | WorkflowDoneEvent | WorkflowErrorEvent
  | InfoEvent | LogEvent

// --- Pipeline ---

export type CollectSource = {
  id: string
  type: 'rss' | 'http' | 'scrape'
  url: string
  limit?: number         // RSS: max items (default 20)
  method?: string        // HTTP: GET/POST
  headers?: Record<string, string>
  body?: string
  selector?: string      // Scrape: CSS selector
  attribute?: string     // Scrape: attr to extract (default: text content)
  scrape_limit?: number  // Scrape: max elements (default 30)
}

export type Pipeline = {
  id: string
  name: string
  description?: string
  stages: Stage[]
  thumbnail_svg?: string
  created_at: string
  updated_at: string
}

export type Stage = {
  id: string
  name: string
  type: 'workflow' | 'approval' | 'notification' | 'schedule' | 'trigger' | 'transform' | 'collect'
  config: StageConfig
  depends_on?: string[]
}

export type StageConfig = {
  workflow_name?: string
  input_mapping?: Record<string, string>
  message?: string
  connection_id?: string
  subject?: string
  timeout?: number
  cron?: string
  timezone?: string
  schedule_id?: string
  trigger_id?: string
  expression?: string
  sources?: CollectSource[]
}

export type PipelineRun = {
  id: string
  pipeline_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'failed'
  current_stage?: string
  stage_results?: Record<string, StageResult>
  started_at: string
  completed_at?: string
}

export type StageResult = {
  stage_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'skipped' | 'failed'
  output?: Record<string, unknown>
  error?: string
  started_at: string
  completed_at?: string
}

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
