export type InputNodeConfig = {
  prompt?: string
  value?: string
}

export type OutputExtract = {
  mode: 'json' | 'tagged'
  key?: string   // json mode: JSON key name
  tag?: string   // tagged mode: XML tag name
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
  output_extract?: OutputExtract
}

export type OutputNodeConfig = {
  output_format?: 'html' | 'md'
  system_prompt?: string
  prompt?: string
  model?: string
  description?: string
}

export type AssetNodeConfig = {
  file_id?: string
  filename?: string
  content_type?: string
  preview_text?: string
}

