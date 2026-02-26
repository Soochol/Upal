import { Inbox, Bot, ArrowRightFromLine, FileBox, Wrench } from 'lucide-react'
import type { ComponentType } from 'react'
import type { NodeType, NodeTypeDefinition, NodeEditorProps } from '../types'

const registry = new Map<NodeType, NodeTypeDefinition>()

export function registerNode(def: NodeTypeDefinition): void {
  registry.set(def.type, def)
}

export function registerNodeEditor(
  type: NodeType,
  EditorComponent: ComponentType<NodeEditorProps>,
): void {
  const existing = registry.get(type)
  if (!existing) throw new Error(`Node type "${type}" not registered`)
  registry.set(type, { ...existing, EditorComponent })
}

export function getNodeDefinition(type: NodeType): NodeTypeDefinition {
  const def = registry.get(type)
  if (!def) throw new Error(`Unknown node type: "${type}"`)
  return def
}

export function getAllNodeDefinitions(): NodeTypeDefinition[] {
  return Array.from(registry.values())
}

// --- Initial registrations (styling only; EditorComponent registered by features/edit-node) ---

registerNode({
  type: 'input',
  label: 'User Input',
  description: 'User-provided data entry point',
  icon: Inbox,
  border: 'border-node-input/20',
  borderSelected: 'border-node-input/60',
  headerBg: 'bg-node-input/10',
  accent: 'bg-node-input text-node-input-foreground',
  glow: 'shadow-[0_0_20px_var(--color-node-input)/0.25]',
  paletteBg: 'bg-node-input/10 text-node-input border-node-input/20 hover:bg-node-input/20',
  cssVar: 'var(--node-input)',
})

registerNode({
  type: 'agent',
  label: 'Agent',
  description: 'AI model processing step',
  icon: Bot,
  border: 'border-node-agent/20',
  borderSelected: 'border-node-agent/60',
  headerBg: 'bg-node-agent/10',
  accent: 'bg-node-agent text-node-agent-foreground',
  glow: 'shadow-[0_0_20px_var(--color-node-agent)/0.25]',
  paletteBg: 'bg-node-agent/10 text-node-agent border-node-agent/20 hover:bg-node-agent/20',
  cssVar: 'var(--node-agent)',
})

registerNode({
  type: 'output',
  label: 'Output',
  description: 'Workflow result endpoint',
  icon: ArrowRightFromLine,
  border: 'border-node-output/20',
  borderSelected: 'border-node-output/60',
  headerBg: 'bg-node-output/10',
  accent: 'bg-node-output text-node-output-foreground',
  glow: 'shadow-[0_0_20px_var(--color-node-output)/0.25]',
  paletteBg: 'bg-node-output/10 text-node-output border-node-output/20 hover:bg-node-output/20',
  cssVar: 'var(--node-output)',
})

registerNode({
  type: 'tool',
  label: 'Tool',
  description: 'Direct tool execution (no LLM)',
  icon: Wrench,
  border: 'border-node-tool/20',
  borderSelected: 'border-node-tool/60',
  headerBg: 'bg-node-tool/10',
  accent: 'bg-node-tool text-node-tool-foreground',
  glow: 'shadow-[0_0_20px_var(--color-node-tool)/0.25]',
  paletteBg: 'bg-node-tool/10 text-node-tool border-node-tool/20 hover:bg-node-tool/20',
  cssVar: 'var(--node-tool)',
})

registerNode({
  type: 'asset',
  label: 'Asset',
  description: 'File or data asset reference',
  icon: FileBox,
  border: 'border-node-asset/20',
  borderSelected: 'border-node-asset/60',
  headerBg: 'bg-node-asset/10',
  accent: 'bg-node-asset text-node-asset-foreground',
  glow: 'shadow-[0_0_20px_var(--color-node-asset)/0.25]',
  paletteBg: 'bg-node-asset/10 text-node-asset border-node-asset/20 hover:bg-node-asset/20',
  cssVar: 'var(--node-asset)',
})
