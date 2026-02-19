# Mention-based Prompt Editor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace plain `<Textarea>` in agent node's prompt fields with a TipTap-based editor that supports `{{` triggered mention pills for upstream node references.

**Architecture:** TipTap editor with custom Mention extension. Typing `{{` shows a dropdown of upstream nodes. Selecting inserts an inline pill (icon + label). Content serializes to `{{node_id}}` format for backend compatibility.

**Tech Stack:** TipTap (ProseMirror), @tiptap/extension-mention, @tiptap/react, Lucide icons, existing Tailwind + CSS variables.

---

### Task 1: Install TipTap Dependencies

**Step 1: Install packages**

Run: `cd /home/dev/code/Upal/web && npm install @tiptap/react @tiptap/pm @tiptap/starter-kit @tiptap/extension-mention @tiptap/extension-placeholder`

**Step 2: Verify installation**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | head -5`
Expected: No new type errors (existing errors OK)

**Step 3: Commit**

```bash
cd /home/dev/code/Upal/web && git add package.json package-lock.json
git commit -m "chore: add TipTap dependencies for mention prompt editor"
```

---

### Task 2: Create MentionList Component

**Files:**
- Create: `web/src/components/editor/MentionList.tsx`

**Step 1: Create the mention suggestion dropdown**

```tsx
// web/src/components/editor/MentionList.tsx
import { forwardRef, useEffect, useImperativeHandle, useState } from 'react'
import { Inbox, Bot, Wrench, ArrowRightFromLine, Globe } from 'lucide-react'

export type MentionItem = {
  id: string
  label: string
  nodeType: string
}

type MentionListProps = {
  items: MentionItem[]
  command: (item: MentionItem) => void
}

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
  external: Globe,
}

const colorMap: Record<string, string> = {
  input: 'text-node-input',
  agent: 'text-node-agent',
  tool: 'text-node-tool',
  output: 'text-node-output',
  external: 'text-purple-500',
}

export type MentionListRef = {
  onKeyDown: (event: KeyboardEvent) => boolean
}

export const MentionList = forwardRef<MentionListRef, MentionListProps>(
  ({ items, command }, ref) => {
    const [selectedIndex, setSelectedIndex] = useState(0)

    useEffect(() => setSelectedIndex(0), [items])

    useImperativeHandle(ref, () => ({
      onKeyDown: (event: KeyboardEvent) => {
        if (event.key === 'ArrowUp') {
          setSelectedIndex((i) => (i + items.length - 1) % items.length)
          return true
        }
        if (event.key === 'ArrowDown') {
          setSelectedIndex((i) => (i + 1) % items.length)
          return true
        }
        if (event.key === 'Enter') {
          const item = items[selectedIndex]
          if (item) command(item)
          return true
        }
        return false
      },
    }))

    if (items.length === 0) {
      return (
        <div className="rounded-lg border border-border bg-popover p-2 shadow-md">
          <p className="text-xs text-muted-foreground">No upstream nodes connected</p>
        </div>
      )
    }

    return (
      <div className="rounded-lg border border-border bg-popover shadow-md py-1 min-w-[180px]">
        {items.map((item, index) => {
          const Icon = iconMap[item.nodeType] || Bot
          const color = colorMap[item.nodeType] || 'text-muted-foreground'
          return (
            <button
              key={item.id}
              onClick={() => command(item)}
              className={`flex w-full items-center gap-2 px-2.5 py-1.5 text-left text-xs transition-colors ${
                index === selectedIndex ? 'bg-accent text-accent-foreground' : 'text-foreground'
              }`}
            >
              <Icon className={`h-3.5 w-3.5 shrink-0 ${color}`} />
              <span className="truncate font-medium">{item.label}</span>
              <span className="ml-auto text-[10px] text-muted-foreground/60 font-mono">{item.id}</span>
            </button>
          )
        })}
      </div>
    )
  },
)

MentionList.displayName = 'MentionList'
```

**Step 2: Type-check**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | grep MentionList`
Expected: No errors for MentionList.tsx

---

### Task 3: Create PromptEditor Component

**Files:**
- Create: `web/src/components/editor/PromptEditor.tsx`

This is the main component. It includes:
- TipTap editor setup with Mention extension
- Custom `{{` trigger via `findSuggestionMatch`
- MentionPill React NodeView for inline pill rendering
- Serialize (TipTap JSON → `{{id}}` string) and deserialize (`{{id}}` string → TipTap JSON)
- Suggestion state bridge between TipTap lifecycle and React

**Step 1: Create PromptEditor**

```tsx
// web/src/components/editor/PromptEditor.tsx
import { useCallback, useEffect, useMemo, useRef } from 'react'
import { useEditor, EditorContent, ReactNodeViewRenderer, NodeViewWrapper, type JSONContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Mention from '@tiptap/extension-mention'
import Placeholder from '@tiptap/extension-placeholder'
import { type SuggestionOptions, type SuggestionProps } from '@tiptap/suggestion'
import { ReactRenderer } from '@tiptap/react'
import { Inbox, Bot, Wrench, ArrowRightFromLine, Globe } from 'lucide-react'
import { useWorkflowStore } from '@/stores/workflowStore'
import { MentionList, type MentionItem, type MentionListRef } from './MentionList'

// ── Icon / color maps for pills ──

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  input: Inbox,
  agent: Bot,
  tool: Wrench,
  output: ArrowRightFromLine,
  external: Globe,
}

const pillColorMap: Record<string, string> = {
  input: 'bg-node-input/15 text-node-input',
  agent: 'bg-node-agent/15 text-node-agent',
  tool: 'bg-node-tool/15 text-node-tool',
  output: 'bg-node-output/15 text-node-output',
  external: 'bg-purple-500/15 text-purple-500',
}

// ── Mention Pill NodeView ──

function MentionPill({ node }: { node: { attrs: Record<string, string> } }) {
  const nodeType = node.attrs.nodeType || 'agent'
  const Icon = iconMap[nodeType] || Bot
  const colors = pillColorMap[nodeType] || 'bg-muted text-muted-foreground'

  return (
    <NodeViewWrapper
      as="span"
      className={`mention-pill inline-flex items-center gap-0.5 rounded px-1 py-px text-[11px] font-medium align-baseline cursor-default ${colors}`}
    >
      <Icon className="h-3 w-3 shrink-0" />
      <span className="truncate max-w-[120px]">{node.attrs.label || node.attrs.id}</span>
    </NodeViewWrapper>
  )
}

// ── Serialization ──

function serializeContent(json: JSONContent): string {
  if (!json.content) return ''
  return json.content
    .map((paragraph) => {
      if (!paragraph.content) return ''
      return paragraph.content
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

function deserializeContent(text: string, nodeMap: Map<string, MentionItem>): JSONContent {
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

// ── Custom findSuggestionMatch for {{ trigger ──

const findSuggestionMatch: SuggestionOptions['findSuggestionMatch'] = (config) => {
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

// ── PromptEditor Component ──

type PromptEditorProps = {
  value: string
  onChange: (value: string) => void
  nodeId: string
  placeholder?: string
  className?: string
}

export function PromptEditor({ value, onChange, nodeId, placeholder, className }: PromptEditorProps) {
  const edges = useWorkflowStore((s) => s.edges)
  const allNodes = useWorkflowStore((s) => s.nodes)
  const isExternalUpdate = useRef(false)

  // Compute upstream nodes (direct predecessors via edges)
  const upstreamNodes = useMemo<MentionItem[]>(() => {
    const sourceIds = new Set(edges.filter((e) => e.target === nodeId).map((e) => e.source))
    return allNodes
      .filter((n) => sourceIds.has(n.id) && n.type !== 'groupNode')
      .map((n) => ({ id: n.id, label: n.data.label, nodeType: n.data.nodeType }))
  }, [edges, allNodes, nodeId])

  // Node map for deserialization (all nodes, not just upstream)
  const nodeMap = useMemo(() => {
    const map = new Map<string, MentionItem>()
    for (const n of allNodes) {
      if (n.type !== 'groupNode') {
        map.set(n.id, { id: n.id, label: n.data.label, nodeType: n.data.nodeType })
      }
    }
    return map
  }, [allNodes])

  // Ref to hold upstream nodes for suggestion (avoids stale closure in TipTap config)
  const upstreamRef = useRef(upstreamNodes)
  upstreamRef.current = upstreamNodes

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        // Disable block-level nodes we don't need
        heading: false,
        blockquote: false,
        codeBlock: false,
        bulletList: false,
        orderedList: false,
        listItem: false,
        horizontalRule: false,
      }),
      Placeholder.configure({ placeholder: placeholder ?? 'Type {{ to reference a node...' }),
      Mention.extend({
        addAttributes() {
          return {
            id: { default: null },
            label: { default: null },
            nodeType: { default: 'agent' },
          }
        },
        addNodeView() {
          return ReactNodeViewRenderer(MentionPill, { as: 'span', className: '' })
        },
      }).configure({
        HTMLAttributes: { class: 'mention-pill-wrapper' },
        suggestion: {
          findSuggestionMatch,
          items: ({ query }: { query: string }) => {
            const q = query.toLowerCase()
            return upstreamRef.current.filter(
              (n) => n.label.toLowerCase().includes(q) || n.id.toLowerCase().includes(q),
            )
          },
          render: () => {
            let component: ReactRenderer<MentionListRef> | null = null
            let popup: HTMLDivElement | null = null

            return {
              onStart: (props: SuggestionProps) => {
                component = new ReactRenderer(MentionList, {
                  props: { items: props.items, command: props.command },
                  editor: props.editor,
                })

                popup = document.createElement('div')
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
              onUpdate: (props: SuggestionProps) => {
                component?.updateProps({ items: props.items, command: props.command })
                const rect = props.clientRect?.()
                if (rect && popup) {
                  popup.style.left = `${rect.left}px`
                  popup.style.top = `${rect.bottom + 4}px`
                }
              },
              onKeyDown: (props: { event: KeyboardEvent }) => {
                if (props.event.key === 'Escape') {
                  popup?.remove()
                  component?.destroy()
                  popup = null
                  component = null
                  return true
                }
                return component?.ref?.onKeyDown(props.event) ?? false
              },
              onExit: () => {
                popup?.remove()
                component?.destroy()
                popup = null
                component = null
              },
            }
          },
        },
      }),
    ],
    content: deserializeContent(value, nodeMap),
    onUpdate: ({ editor }) => {
      if (isExternalUpdate.current) return
      onChange(serializeContent(editor.getJSON()))
    },
    editorProps: {
      attributes: {
        class: `prompt-editor-content min-h-[48px] text-xs outline-none ${className ?? ''}`,
      },
    },
  })

  // Sync external value changes (e.g., AI assistant updating the prompt)
  useEffect(() => {
    if (!editor || editor.isDestroyed) return
    const current = serializeContent(editor.getJSON())
    if (current !== value) {
      isExternalUpdate.current = true
      editor.commands.setContent(deserializeContent(value, nodeMap))
      isExternalUpdate.current = false
    }
  }, [value, editor, nodeMap])

  return (
    <div className="rounded-md border border-input bg-background px-3 py-2 ring-offset-background focus-within:ring-1 focus-within:ring-ring">
      <EditorContent editor={editor} />
    </div>
  )
}
```

**Step 2: Type-check**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit 2>&1 | grep -E "PromptEditor|MentionList"`
Expected: No errors (may need minor type adjustments)

---

### Task 4: Add CSS Styles for Mention Pills

**Files:**
- Modify: `web/src/index.css`

**Step 1: Add prompt editor styles**

Add at the end of `web/src/index.css` (before any closing comment):

```css
/* ── TipTap Prompt Editor ── */

.prompt-editor-content p {
  margin: 0;
}

.prompt-editor-content .is-empty::before {
  color: var(--muted-foreground);
  opacity: 0.5;
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}

/* Mention pill inline wrapper — remove ProseMirror NodeView chrome */
.mention-pill-wrapper {
  display: inline;
}
```

**Step 2: Verify styles applied**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS (CSS changes don't affect type-check)

---

### Task 5: Replace Textarea in NodeEditor

**Files:**
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`

**Step 1: Replace User Prompt textarea with PromptEditor**

In `web/src/components/editor/nodes/NodeEditor.tsx`:

1. Add import:
```tsx
import { PromptEditor } from '@/components/editor/PromptEditor'
```

2. Replace the User Prompt `<Textarea>` block (around line 142-151):

Before:
```tsx
<div className="space-y-1">
  <Label htmlFor="node-user-prompt" className="text-xs">User Prompt</Label>
  <Textarea
    id="node-user-prompt"
    className="min-h-[48px] text-xs"
    value={(config.prompt as string) ?? ''}
    placeholder="Type {{ to reference a node..."
    onChange={(e) => setConfig('prompt', e.target.value)}
  />
</div>
```

After:
```tsx
<div className="space-y-1">
  <Label className="text-xs">User Prompt</Label>
  <PromptEditor
    value={(config.prompt as string) ?? ''}
    onChange={(v) => setConfig('prompt', v)}
    nodeId={nodeId}
    placeholder="Type {{ to reference a node..."
  />
</div>
```

**Step 2: Type-check**

Run: `cd /home/dev/code/Upal/web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/components/editor/MentionList.tsx web/src/components/editor/PromptEditor.tsx web/src/components/editor/nodes/NodeEditor.tsx web/src/index.css
git commit -m "feat: add TipTap mention prompt editor with inline node pills

Replace plain textarea in agent User Prompt with a TipTap-based editor.
Typing {{ triggers a dropdown of upstream nodes; selecting inserts an
inline colored pill showing icon + label. Serializes to {{node_id}} format."
```

---

### Task 6: Manual Smoke Test

**Step 1: Start dev server**

Run: `cd /home/dev/code/Upal && make dev-frontend` (in one terminal)
Run: `cd /home/dev/code/Upal && make dev-backend` (in another terminal)

**Step 2: Test mention flow**

1. Open `http://localhost:5173/editor`
2. Use prompt bar to generate a workflow (e.g., "summarize user input")
3. Click the agent node to open properties
4. In User Prompt field, type `{{`
5. Verify dropdown appears with upstream node(s)
6. Click or press Enter to select
7. Verify pill appears inline with icon + label + color
8. Verify Backspace deletes pill as one unit
9. Save workflow and reload — verify pills re-render correctly

**Step 3: Test edge cases**

1. Empty prompt → placeholder shows
2. No upstream nodes → dropdown shows "No upstream nodes connected"
3. Type `{{` then Escape → dropdown closes
4. Multiple pills in one prompt → all render correctly
5. Delete node referenced by pill → pill shows ID as fallback

---

### Task 7: Build Verification

**Step 1: Full type-check and build**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: Build succeeds with no errors
