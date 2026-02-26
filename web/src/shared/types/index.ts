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

export type CreatedWorkflowInfo = {
  name: string
  status: 'success' | 'exists' | 'failed'
  error?: string
}

export type ConfigurePipelineResponse = {
  sources?: PipelineSource[]
  schedule?: string | null
  workflows?: PipelineWorkflow[]
  model?: string | null
  context?: PipelineContext
  explanation: string
  created_workflows?: CreatedWorkflowInfo[]
}

// --- Run (re-exported for use within shared layer) ---
export type { RunEvent } from '@/entities/run/types'

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

// --- Content Session (re-exported from entity) ---
export type { ContentSessionStatus, SourceType } from '@/entities/content-session'
