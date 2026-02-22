import { Inbox, Bot, ArrowRightFromLine, FileBox } from 'lucide-react'
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
  border: 'border-node-input/30',
  borderSelected: 'border-node-input',
  headerBg: 'bg-node-input/15',
  accent: 'bg-node-input text-node-input-foreground',
  glow: 'shadow-[0_0_16px_oklch(0.795_0.184_86.047/0.4)]',
  paletteBg: 'bg-node-input/15 text-node-input border-node-input/30 hover:bg-node-input/25',
  cssVar: 'var(--node-input)',
})

registerNode({
  type: 'agent',
  label: 'Agent',
  description: 'AI model processing step',
  icon: Bot,
  border: 'border-node-agent/30',
  borderSelected: 'border-node-agent',
  headerBg: 'bg-node-agent/15',
  accent: 'bg-node-agent text-node-agent-foreground',
  glow: 'shadow-[0_0_16px_oklch(0.588_0.158_241.966/0.4)]',
  paletteBg: 'bg-node-agent/15 text-node-agent border-node-agent/30 hover:bg-node-agent/25',
  cssVar: 'var(--node-agent)',
})

registerNode({
  type: 'output',
  label: 'Output',
  description: 'Workflow result endpoint',
  icon: ArrowRightFromLine,
  border: 'border-node-output/30',
  borderSelected: 'border-node-output',
  headerBg: 'bg-node-output/15',
  accent: 'bg-node-output text-node-output-foreground',
  glow: 'shadow-[0_0_16px_oklch(0.648_0.2_142.495/0.4)]',
  paletteBg: 'bg-node-output/15 text-node-output border-node-output/30 hover:bg-node-output/25',
  cssVar: 'var(--node-output)',
})

registerNode({
  type: 'asset',
  label: 'Asset',
  description: 'File or data asset reference',
  icon: FileBox,
  border: 'border-node-asset/30',
  borderSelected: 'border-node-asset',
  headerBg: 'bg-node-asset/15',
  accent: 'bg-node-asset text-node-asset-foreground',
  glow: 'shadow-[0_0_16px_oklch(0.75_0.08_200/0.4)]',
  paletteBg: 'bg-node-asset/15 text-node-asset border-node-asset/30 hover:bg-node-asset/25',
  cssVar: 'var(--node-asset)',
})
