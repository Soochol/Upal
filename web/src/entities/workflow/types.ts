export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'output' | 'asset' | 'group'
  description: string
  config: Record<string, unknown>
}

export type WorkflowDefinition = {
  name: string
  version: number
  nodes: { id: string; type: string; config: Record<string, unknown>; group?: string }[]
  edges: { from: string; to: string; condition?: string; trigger_rule?: string; loop?: boolean }[]
  groups?: { id: string; label: string; color: string }[]
  thumbnail_svg?: string
}
