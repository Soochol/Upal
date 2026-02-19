# Mention-based Prompt Editor Design

## Problem

Agent node's User Prompt requires typing `{{node_id}}` manually (e.g., `{{input_1}}`).
Users must memorize node IDs and type the double-curly syntax correctly. No autocomplete, no visual feedback. Poor UX compared to Opal which uses inline pill badges.

## Solution

Replace plain `<Textarea>` with a TipTap-based rich text editor that supports inline mention pills. Typing `{{` triggers a dropdown of upstream nodes; selecting one inserts a colored pill showing icon + label. Backend serialization stays unchanged (`{{node_id}}`).

## User Flow

1. User types normally in the prompt field
2. Types `{{` → dropdown appears with upstream nodes (icon + label + id)
3. Arrow keys / mouse to select, Enter to confirm
4. `{{` text replaced by colored inline pill: `[icon Label]`
5. Pill is atomic — Backspace deletes whole pill
6. On save: pills serialize to `{{node_id}}`
7. On load: `{{node_id}}` patterns deserialize back to pills

## Architecture

### New Files

- `web/src/components/editor/PromptEditor.tsx` — TipTap editor + mention integration
- `web/src/components/editor/MentionList.tsx` — Dropdown suggestion component

### Modified Files

- `web/src/components/editor/nodes/NodeEditor.tsx` — Replace `<Textarea>` with `<PromptEditor>`

### Component API

```tsx
type PromptEditorProps = {
  value: string                     // "Summarize {{input_1}}"
  onChange: (value: string) => void  // serialized string
  nodeId: string                    // current node (for upstream computation)
  placeholder?: string
  className?: string
}
```

### Serialization

```
[Editor]    "Summarize " + [pill:User Input] + " please"
     ↕
[Config]    "Summarize {{input_1}} please"
```

- **Serialize**: Walk TipTap JSON → text nodes as-is, mention nodes → `{{id}}`
- **Deserialize**: Regex `{{(\w+)}}` → TipTap mention nodes with label/type lookup

### Upstream Node Detection

Reuse existing pattern from `AIChatEditor.tsx`:
```ts
edges.filter(e => e.target === nodeId).map(e => e.source)
```

### Pill Styling

Use existing node-type semantic colors:
- Input: `--node-input`
- Agent: `--node-agent`
- Tool: `--node-tool`
- Output: `--node-output`

### Dependencies

```
@tiptap/react @tiptap/starter-kit @tiptap/extension-mention @tiptap/suggestion
```

## Scope

- **Phase 1**: Agent `prompt` field (User Prompt) — most frequent `{{}}` usage
- **Phase 2**: Agent `system_prompt`, Tool `input` — same component reused
