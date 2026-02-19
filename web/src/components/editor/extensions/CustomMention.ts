import { ReactNodeViewRenderer, NodeViewWrapper } from '@tiptap/react'
import type { ReactNodeViewProps } from '@tiptap/react'
import Mention from '@tiptap/extension-mention'
import type { SuggestionOptions } from '@tiptap/suggestion'
import { createElement } from 'react'
import { Bot } from 'lucide-react'
import { nodeIconMap } from '@/lib/nodeTypes'

const pillColorMap: Record<string, string> = {
  input: 'bg-node-input/15 text-node-input',
  agent: 'bg-node-agent/15 text-node-agent',
  tool: 'bg-node-tool/15 text-node-tool',
  output: 'bg-node-output/15 text-node-output',
  external: 'bg-purple-500/15 text-purple-500',
}

// ── Mention pill rendered inside the editor ──

function MentionPill({ node }: ReactNodeViewProps) {
  const Icon = nodeIconMap[node.attrs.nodeType as string] || Bot
  const colors = pillColorMap[node.attrs.nodeType as string] || 'bg-muted text-muted-foreground'
  return createElement(
    NodeViewWrapper,
    {
      as: 'span',
      className: `mention-pill inline-flex items-center gap-0.5 rounded px-1 py-px text-[11px] font-medium align-baseline cursor-default ${colors}`,
    },
    createElement(Icon, { className: 'h-3 w-3 shrink-0' }),
    createElement(
      'span',
      { className: 'truncate max-w-[120px]' },
      (node.attrs.label as string) || (node.attrs.id as string),
    ),
  )
}

// ── Custom findSuggestionMatch for multi-char `{{` trigger ──

export const customFindSuggestionMatch: SuggestionOptions['findSuggestionMatch'] = (config) => {
  const { $position } = config
  const text = $position.parent.textBetween(
    Math.max(0, $position.parentOffset - 500),
    $position.parentOffset,
    null,
    '\ufffc',
  )
  const match = text.match(/\{\{(\w*)$/)
  if (!match) return null
  return {
    range: {
      from: $position.pos - match[0].length,
      to: $position.pos,
    },
    query: match[1],
    text: match[0],
  }
}

// ── Custom Mention extension with nodeType attribute + React pill ──

export const CustomMention = Mention.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      nodeType: {
        default: 'agent',
        parseHTML: (element: HTMLElement) => element.getAttribute('data-node-type'),
        renderHTML: (attributes: Record<string, unknown>) => {
          if (!attributes.nodeType) return {}
          return { 'data-node-type': attributes.nodeType }
        },
      },
    }
  },
  renderHTML({ node, HTMLAttributes }) {
    return [
      'span',
      { 'data-type': this.name, ...HTMLAttributes },
      `{{${node.attrs.id as string}}}`,
    ]
  },
  addNodeView() {
    return ReactNodeViewRenderer(MentionPill, { as: 'span', className: '' })
  },
})
