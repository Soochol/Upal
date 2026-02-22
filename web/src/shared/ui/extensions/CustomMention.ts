import { ReactNodeViewRenderer, NodeViewWrapper } from '@tiptap/react'
import type { ReactNodeViewProps } from '@tiptap/react'
import Mention from '@tiptap/extension-mention'
import type { SuggestionOptions } from '@tiptap/suggestion'
import { createElement } from 'react'
import type { ComponentType } from 'react'
import { Bot } from 'lucide-react'
import { getNodeDefinition } from '@/entities/node'
import type { NodeType } from '@/entities/node'

const knownNodeTypes: NodeType[] = ['input', 'agent', 'output', 'asset']

const pillColorMap: Record<string, string> = Object.fromEntries(
  knownNodeTypes.map((t) => [t, `bg-node-${t}/15 text-node-${t}`]),
)

// ── Mention pill rendered inside the editor ──

function MentionPill({ node }: ReactNodeViewProps) {
  let Icon: ComponentType<{ className?: string }> = Bot
  try { Icon = getNodeDefinition(node.attrs.nodeType as NodeType).icon } catch { /* unknown type */ }
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
