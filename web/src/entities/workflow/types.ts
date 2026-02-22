export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'output' | 'asset' | 'group'
  description: string
  config: Record<string, unknown>
}
