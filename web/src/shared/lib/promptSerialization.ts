import type { JSONContent } from '@tiptap/core'
import type { MentionItem } from '@/components/editor/MentionList'

// ── Serialization: TipTap JSON → {{id}} string ──

export function serializeContent(json: JSONContent): string {
  if (!json.content) return ''
  return json.content
    .map((para) => {
      if (!para.content) return ''
      return para.content
        .map((node) => {
          if (node.type === 'mention') return `{{${node.attrs?.id}}}`
          if (node.type === 'text') return node.text ?? ''
          if (node.type === 'hardBreak') return '\n'
          return ''
        })
        .join('')
    })
    .join('\n')
}

// ── Deserialization: {{id}} string → TipTap JSON ──

export function deserializeContent(
  text: string,
  nodeMap: Map<string, MentionItem>,
): JSONContent {
  if (!text) return { type: 'doc', content: [{ type: 'paragraph' }] }
  const paragraphs = text.split('\n').map((line) => {
    const content: JSONContent[] = []
    let lastIndex = 0
    const regex = /\{\{(\w+)\}\}/g
    let match
    while ((match = regex.exec(line)) !== null) {
      if (match.index > lastIndex) {
        content.push({ type: 'text', text: line.slice(lastIndex, match.index) })
      }
      const id = match[1]
      const node = nodeMap.get(id)
      content.push({
        type: 'mention',
        attrs: {
          id,
          label: node?.label ?? id,
          nodeType: node?.nodeType ?? 'agent',
        },
      })
      lastIndex = match.index + match[0].length
    }
    if (lastIndex < line.length) {
      content.push({ type: 'text', text: line.slice(lastIndex) })
    }
    return {
      type: 'paragraph',
      content: content.length > 0 ? content : undefined,
    }
  })
  return { type: 'doc', content: paragraphs }
}
