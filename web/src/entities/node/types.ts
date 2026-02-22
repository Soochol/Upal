import type { ComponentType } from 'react'

export type NodeType = 'input' | 'agent' | 'output' | 'asset'

export type NodeEditorProps<C = Record<string, unknown>> = {
  nodeId: string
  config: C
  setConfig: (key: string, value: unknown) => void
}

export type NodeTypeDefinition = {
  type: NodeType
  label: string
  description: string
  icon: ComponentType<{ className?: string }>
  // Styling
  border: string
  borderSelected: string
  headerBg: string
  accent: string
  glow: string
  paletteBg: string
  cssVar: string
  // Editor component (registered separately to avoid circular deps)
  EditorComponent?: ComponentType<NodeEditorProps>
  defaultConfig?: Record<string, unknown>
}
