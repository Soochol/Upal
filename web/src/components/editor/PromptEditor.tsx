import { useEffect, useMemo, useRef } from 'react'
import { useEditor, EditorContent, ReactNodeViewRenderer, NodeViewWrapper, ReactRenderer } from '@tiptap/react'
import type { ReactNodeViewProps } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Mention from '@tiptap/extension-mention'
import Placeholder from '@tiptap/extension-placeholder'
import type { SuggestionOptions, SuggestionProps, SuggestionKeyDownProps } from '@tiptap/suggestion'
import type { JSONContent } from '@tiptap/core'
import { Bot } from 'lucide-react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { nodeIconMap } from '@/lib/nodeTypes'
import { MentionList } from './MentionList'
import type { MentionItem, MentionListRef } from './MentionList'

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
  return (
    <NodeViewWrapper
      as="span"
      className={`mention-pill inline-flex items-center gap-0.5 rounded px-1 py-px text-[11px] font-medium align-baseline cursor-default ${colors}`}
    >
      <Icon className="h-3 w-3 shrink-0" />
      <span className="truncate max-w-[120px]">
        {(node.attrs.label as string) || (node.attrs.id as string)}
      </span>
    </NodeViewWrapper>
  )
}

// ── Custom findSuggestionMatch for multi-char `{{` trigger ──

const customFindSuggestionMatch: SuggestionOptions['findSuggestionMatch'] = (config) => {
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

// ── Serialization: TipTap JSON → {{id}} string ──

function serializeContent(json: JSONContent): string {
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

function deserializeContent(
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

// ── Custom Mention extension with nodeType attribute + React pill ──

const CustomMention = Mention.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      nodeType: { default: 'agent' },
    }
  },
  addNodeView() {
    return ReactNodeViewRenderer(MentionPill, { as: 'span', className: '' })
  },
})

// ── Main PromptEditor component ──

type PromptEditorProps = {
  value: string
  onChange: (value: string) => void
  nodeId: string
  placeholder?: string
  className?: string
}

export function PromptEditor({
  value,
  onChange,
  nodeId,
  placeholder = 'Type {{ to reference a node...',
  className,
}: PromptEditorProps) {
  const edges = useWorkflowStore((s) => s.edges)
  const allNodes = useWorkflowStore((s) => s.nodes)

  // Compute upstream nodes
  const upstreamNodes = useMemo(() => {
    const sourceIds = new Set(
      edges.filter((e) => e.target === nodeId).map((e) => e.source),
    )
    return allNodes
      .filter((n) => sourceIds.has(n.id) && n.type !== 'groupNode')
      .map((n) => ({
        id: n.id,
        label: n.data.label,
        nodeType: n.data.nodeType,
      }))
  }, [edges, allNodes, nodeId])

  // Build a map for deserialization lookups
  const nodeMap = useMemo(() => {
    const map = new Map<string, MentionItem>()
    allNodes.forEach((n) => {
      if (n.type !== 'groupNode') {
        map.set(n.id, {
          id: n.id,
          label: n.data.label,
          nodeType: n.data.nodeType,
        })
      }
    })
    return map
  }, [allNodes])

  // Refs for upstream nodes (so suggestion items callback can access latest)
  const upstreamRef = useRef(upstreamNodes)
  upstreamRef.current = upstreamNodes

  // Ref to prevent update loops
  const isExternalUpdate = useRef(false)

  // Ref to track popup element for cleanup on unmount
  const popupRef = useRef<HTMLDivElement | null>(null)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        // Disable features we don't need in a prompt editor
        heading: false,
        bulletList: false,
        orderedList: false,
        blockquote: false,
        codeBlock: false,
        horizontalRule: false,
        bold: false,
        italic: false,
        strike: false,
        code: false,
      }),
      Placeholder.configure({
        placeholder,
      }),
      CustomMention.configure({
        suggestion: {
          items: ({ query }: { query: string }) => {
            const nodes = upstreamRef.current
            if (!query) return nodes
            return nodes.filter(
              (n) =>
                n.label.toLowerCase().includes(query.toLowerCase()) ||
                n.id.toLowerCase().includes(query.toLowerCase()),
            )
          },
          findSuggestionMatch: customFindSuggestionMatch,
          render: () => {
            let component: ReactRenderer<MentionListRef, { items: MentionItem[]; command: (item: MentionItem) => void }> | null = null
            let popup: HTMLDivElement | null = null

            return {
              onStart: (props: SuggestionProps<MentionItem>) => {
                component = new ReactRenderer(MentionList, {
                  props: { items: props.items, command: props.command },
                  editor: props.editor,
                })
                popup = document.createElement('div')
                popupRef.current = popup
                popup.style.position = 'fixed'
                popup.style.zIndex = '50'
                document.body.appendChild(popup)
                popup.appendChild(component.element)
                const rect = props.clientRect?.()
                if (rect) {
                  popup.style.left = `${rect.left}px`
                  popup.style.top = `${rect.bottom + 4}px`
                }
              },
              onUpdate: (props: SuggestionProps<MentionItem>) => {
                component?.updateProps({
                  items: props.items,
                  command: props.command,
                })
                const rect = props.clientRect?.()
                if (rect && popup) {
                  popup.style.left = `${rect.left}px`
                  popup.style.top = `${rect.bottom + 4}px`
                }
              },
              onKeyDown: (props: SuggestionKeyDownProps) => {
                if (props.event.key === 'Escape') {
                  popup?.remove()
                  component?.destroy()
                  return true
                }
                return component?.ref?.onKeyDown(props.event) ?? false
              },
              onExit: () => {
                popup?.remove()
                component?.destroy()
                popupRef.current = null
              },
            }
          },
        },
      }),
    ],
    content: deserializeContent(value, nodeMap),
    onUpdate: ({ editor: ed }) => {
      if (isExternalUpdate.current) return
      const serialized = serializeContent(ed.getJSON())
      onChange(serialized)
    },
    editorProps: {
      attributes: {
        class: 'prompt-editor-content outline-none text-xs min-h-[48px]',
      },
    },
  })

  // Sync external value changes into the editor
  useEffect(() => {
    if (!editor || editor.isDestroyed) return
    const current = serializeContent(editor.getJSON())
    if (current !== value) {
      isExternalUpdate.current = true
      try {
        editor.commands.setContent(deserializeContent(value, nodeMap))
      } finally {
        isExternalUpdate.current = false
      }
    }
  }, [value, editor, nodeMap])

  // Clean up orphaned popup on unmount
  useEffect(() => {
    return () => {
      popupRef.current?.remove()
      popupRef.current = null
    }
  }, [])

  return (
    <div
      className={`rounded-md border border-input bg-background px-3 py-2 ring-offset-background focus-within:ring-1 focus-within:ring-ring ${className ?? ''}`}
    >
      <EditorContent editor={editor} />
    </div>
  )
}
