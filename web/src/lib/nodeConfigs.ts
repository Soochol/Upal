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
  prompt?: string
  display_mode?: 'manual' | 'auto-layout'
  layout_model?: string
  system_prompt?: string
  description?: string
}
