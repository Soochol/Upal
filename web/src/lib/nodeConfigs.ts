export type InputNodeConfig = {
  placeholder?: string
  value?: string
}

export type AgentNodeConfig = {
  model?: string
  prompt?: string
  system_prompt?: string
  output?: string
  tools?: string[]
  description?: string
  // Text model options
  temperature?: number
  max_tokens?: number
  top_p?: number
  // Image model options
  aspect_ratio?: string
  steps?: number
  quality?: string
}

export type OutputNodeConfig = {
  output_format?: 'html' | 'md'
  system_prompt?: string
  prompt?: string
  model?: string
  description?: string
}

export type BranchNodeConfig = {
  mode?: 'expression' | 'llm'
  expression?: string
  prompt?: string
  model?: string
  description?: string
}

export type IteratorNodeConfig = {
  source?: string
  item_key?: string
  max_iterations?: number
  description?: string
}

export type NotificationNodeConfig = {
  connection_id?: string
  message?: string
  channel?: string
  description?: string
}

export type SensorNodeConfig = {
  mode?: 'poll' | 'webhook'
  url?: string
  connection_id?: string
  condition?: string
  interval?: number
  timeout?: number
  description?: string
}

export type ApprovalNodeConfig = {
  connection_id?: string
  message?: string
  channel?: string
  timeout?: number
  actions?: string[]
  description?: string
}

export type SubWorkflowNodeConfig = {
  workflow_name?: string
  input_mapping?: Record<string, string>
  description?: string
}
