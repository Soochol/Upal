# Upal UI/UX Phase 2-4: Opal-Level Professional UI

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the remaining visual gap between Upal and Google Opal through horizontal flow layout, enhanced node cards, dashed edge styling, tabbed right panel, and micro-interaction polish.

**Architecture:** All changes are frontend-only (React/TypeScript). Three phases: Phase 2 (node/flow), Phase 3 (tabbed panel), Phase 4 (polish). No backend changes. No new API endpoints.

**Tech Stack:** React 19 / @xyflow/react 12.10 / Zustand / Tailwind CSS 4.2 / shadcn/ui / Lucide Icons

---

## Phase 2: Node & Flow Enhancement

### Task 1: Install dagre layout library

**Files:**
- Modify: `web/package.json`

**Step 1: Install dagre for automatic graph layout**

Run from `web/`:
```bash
npm install @dagrejs/dagre
```

**Step 2: Verify install**

Run: `npm ls @dagrejs/dagre`
Expected: `@dagrejs/dagre@...`

**Step 3: Commit**

```bash
git add web/package.json web/package-lock.json
git commit -m "feat(web): add dagre for automatic graph layout"
```

---

### Task 2: Create auto-layout utility

**Files:**
- Create: `web/src/lib/layout.ts`

**Step 1: Create the layout utility**

Create `web/src/lib/layout.ts` with:

```typescript
import Dagre from '@dagrejs/dagre'
import type { Node, Edge } from '@xyflow/react'

type LayoutDirection = 'LR' | 'TB'

export function getLayoutedElements(
  nodes: Node[],
  edges: Edge[],
  direction: LayoutDirection = 'LR',
): { nodes: Node[]; edges: Edge[] } {
  const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}))

  g.setGraph({
    rankdir: direction,
    nodesep: 80,
    ranksep: 120,
    marginx: 50,
    marginy: 50,
  })

  nodes.forEach((node) => {
    g.setNode(node.id, { width: 280, height: 80 })
  })

  edges.forEach((edge) => {
    g.setEdge(edge.source, edge.target)
  })

  Dagre.layout(g)

  const layoutedNodes = nodes.map((node) => {
    const pos = g.node(node.id)
    return {
      ...node,
      position: { x: pos.x - 140, y: pos.y - 40 },
    }
  })

  return { nodes: layoutedNodes, edges }
}
```

**Step 2: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds with no errors.

**Step 3: Commit**

```bash
git add web/src/lib/layout.ts
git commit -m "feat(web): add dagre auto-layout utility for LR/TB flow"
```

---

### Task 3: Integrate auto-layout into store and switch to horizontal flow

**Files:**
- Modify: `web/src/stores/workflowStore.ts`
- Modify: `web/src/App.tsx`
- Modify: `web/src/lib/serializer.ts`

**Step 1: Add `applyAutoLayout` action to workflowStore**

In `web/src/stores/workflowStore.ts`:

1. Add import at top:
```typescript
import { getLayoutedElements } from '@/lib/layout'
```

2. Add to `WorkflowState` type:
```typescript
applyAutoLayout: () => void
```

3. Add implementation inside `create(...)`:
```typescript
applyAutoLayout: () => {
  const { nodes, edges } = get()
  if (nodes.length === 0) return
  const { nodes: layouted } = getLayoutedElements(nodes, edges, 'LR')
  set({ nodes: layouted })
},
```

4. Modify `addNode` to auto-layout after adding:
Change the `addNode` implementation so that after adding a node, it calls auto-layout:
```typescript
addNode: (type, position) => {
  const id = getId()
  const labels: Record<string, string> = {
    input: 'User Input',
    agent: 'Agent',
    tool: 'Tool',
    output: 'Output',
  }
  const newNode: Node<NodeData> = {
    id,
    type: 'upalNode',
    position,
    data: {
      label: labels[type] || type,
      nodeType: type,
      config: {},
    },
  }
  set({ nodes: [...get().nodes, newNode] })
  // Auto-layout after adding
  get().applyAutoLayout()
},
```

5. Also modify `onConnect` to auto-layout after connecting:
```typescript
onConnect: (connection) => {
  set({ edges: addEdge(connection, get().edges) })
  get().applyAutoLayout()
},
```

**Step 2: Update deserializer to use horizontal layout**

In `web/src/lib/serializer.ts`, add import and use layout:

```typescript
import { getLayoutedElements } from '@/lib/layout'
```

Change the `deserializeWorkflow` function to use auto-layout instead of hardcoded vertical positions:

```typescript
export function deserializeWorkflow(
  wf: WorkflowDefinition,
): { nodes: Node<NodeData>[]; edges: Edge[] } {
  const labels: Record<string, string> = {
    input: 'User Input',
    agent: 'Agent',
    tool: 'Tool',
    output: 'Output',
  }

  const nodes: Node<NodeData>[] = wf.nodes.map((n, i) => ({
    id: n.id,
    type: 'upalNode',
    position: { x: i * 350, y: 0 }, // placeholder, will be overridden by layout
    data: {
      label: labels[n.type] || n.type,
      nodeType: n.type as NodeData['nodeType'],
      config: n.config,
    },
  }))

  const edges: Edge[] = wf.edges.map((e, i) => ({
    id: `edge-${i}`,
    source: e.from,
    target: e.to,
  }))

  // Apply auto-layout for horizontal flow
  return getLayoutedElements(nodes, edges, 'LR')
}
```

**Step 3: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add web/src/stores/workflowStore.ts web/src/lib/serializer.ts
git commit -m "feat(web): integrate horizontal auto-layout into store and deserializer"
```

---

### Task 4: Update node handles for horizontal flow (Left/Right)

**Files:**
- Modify: `web/src/components/editor/nodes/UpalNode.tsx`

**Step 1: Change handle positions from Top/Bottom to Left/Right**

In `web/src/components/editor/nodes/UpalNode.tsx`:

Change the target Handle from `Position.Top` to `Position.Left`:
```typescript
<Handle
  type="target"
  position={Position.Left}
  className="!w-3 !h-3 !bg-border !border-2 !border-background !-left-1.5"
/>
```

Change the source Handle from `Position.Bottom` to `Position.Right`:
```typescript
<Handle
  type="source"
  position={Position.Right}
  className="!w-3 !h-3 !bg-border !border-2 !border-background !-right-1.5"
/>
```

**Step 2: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add web/src/components/editor/nodes/UpalNode.tsx
git commit -m "feat(web): switch node handles to left/right for horizontal flow"
```

---

### Task 5: Enlarge node cards and add description field

**Files:**
- Modify: `web/src/stores/workflowStore.ts`
- Modify: `web/src/components/editor/nodes/UpalNode.tsx`
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`

**Step 1: Add `description` to NodeData**

In `web/src/stores/workflowStore.ts`, update the `NodeData` type:

```typescript
export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'tool' | 'output'
  description: string
  config: Record<string, unknown>
}
```

Update the `addNode` action to include default descriptions:
```typescript
addNode: (type, position) => {
  const id = getId()
  const labels: Record<string, string> = {
    input: 'User Input',
    agent: 'Agent',
    tool: 'Tool',
    output: 'Output',
  }
  const descriptions: Record<string, string> = {
    input: 'User-provided data entry point',
    agent: 'AI model processing step',
    tool: 'External tool or function call',
    output: 'Workflow result endpoint',
  }
  const newNode: Node<NodeData> = {
    id,
    type: 'upalNode',
    position,
    data: {
      label: labels[type] || type,
      nodeType: type,
      description: descriptions[type] || '',
      config: {},
    },
  }
  set({ nodes: [...get().nodes, newNode] })
  get().applyAutoLayout()
},
```

Add `updateNodeDescription` action:
```typescript
// In WorkflowState type:
updateNodeDescription: (nodeId: string, description: string) => void

// In create(...):
updateNodeDescription: (nodeId, description) => {
  set({
    nodes: get().nodes.map((n) =>
      n.id === nodeId ? { ...n, data: { ...n.data, description } } : n,
    ),
  })
},
```

**Step 2: Enlarge node card and show description in UpalNode**

In `web/src/components/editor/nodes/UpalNode.tsx`, update the outer div `min-w-[220px]` to `min-w-[280px]` and add description display:

Replace the inner content div:
```typescript
<div className="flex items-center gap-3 px-4 py-3">
  <div
    className={cn(
      'h-9 w-9 rounded-lg flex items-center justify-center shrink-0',
      accentMap[data.nodeType],
    )}
  >
    {Icon && <Icon className="h-4.5 w-4.5" />}
  </div>

  <div className="flex-1 min-w-0">
    <p className="text-sm font-medium text-card-foreground truncate">
      {data.label}
    </p>
    {data.description && (
      <p className="text-xs text-muted-foreground line-clamp-2 mt-0.5">
        {data.description}
      </p>
    )}
  </div>

  {StatusIcon && runStatus === 'running' && (
    <Loader2 className="h-4 w-4 animate-spin text-amber-400 shrink-0" />
  )}
  {StatusIcon && runStatus === 'completed' && (
    <Check className="h-4 w-4 text-green-400 shrink-0" />
  )}
  {StatusIcon && runStatus === 'error' && (
    <X className="h-4 w-4 text-destructive shrink-0" />
  )}
</div>
```

**Step 3: Add description field to NodeEditor**

In `web/src/components/editor/nodes/NodeEditor.tsx`, add after the Label field:

```typescript
const updateNodeDescription = useWorkflowStore((s) => s.updateNodeDescription)
```

And the description field JSX (after the label field's closing `</div>`):
```typescript
<div className="space-y-2">
  <Label htmlFor="node-description">Description</Label>
  <Textarea
    id="node-description"
    className="min-h-[60px] resize-y"
    value={data.description ?? ''}
    placeholder="Describe what this node does..."
    onChange={(e) => updateNodeDescription(nodeId, e.target.value)}
  />
</div>
```

**Step 4: Update deserializer for description field**

In `web/src/lib/serializer.ts`, update the `deserializeWorkflow` function to include description:
```typescript
data: {
  label: labels[n.type] || n.type,
  nodeType: n.type as NodeData['nodeType'],
  description: (n.config.description as string) || '',
  config: n.config,
},
```

**Step 5: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 6: Commit**

```bash
git add web/src/stores/workflowStore.ts web/src/components/editor/nodes/UpalNode.tsx web/src/components/editor/nodes/NodeEditor.tsx web/src/lib/serializer.ts
git commit -m "feat(web): enlarge node cards and add description field"
```

---

### Task 6: Dashed edge styling with animated flow

**Files:**
- Modify: `web/src/index.css`
- Modify: `web/src/stores/workflowStore.ts`

**Step 1: Add dashed animated edge styles in CSS**

In `web/src/index.css`, replace the existing `.react-flow__edge-path` rule with:

```css
.react-flow__edge-path {
  stroke: var(--border);
  stroke-width: 2;
  stroke-dasharray: 6 4;
  animation: dashmove 0.5s linear infinite;
}

@keyframes dashmove {
  to {
    stroke-dashoffset: -10;
  }
}

.react-flow__edge.selected .react-flow__edge-path {
  stroke: var(--ring);
  stroke-width: 2.5;
}
```

**Step 2: Set default edge type to smoothstep in store**

In `web/src/stores/workflowStore.ts`, update the `onConnect` handler to specify edge type:

```typescript
onConnect: (connection) => {
  set({ edges: addEdge({ ...connection, type: 'smoothstep' }, get().edges) })
  get().applyAutoLayout()
},
```

Also update `deserializeWorkflow` in `web/src/lib/serializer.ts` to set edge type:

```typescript
const edges: Edge[] = wf.edges.map((e, i) => ({
  id: `edge-${i}`,
  source: e.from,
  target: e.to,
  type: 'smoothstep',
}))
```

**Step 3: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add web/src/index.css web/src/stores/workflowStore.ts web/src/lib/serializer.ts
git commit -m "feat(web): add dashed animated edges with smoothstep type"
```

---

### Task 7: Selection glow effect on nodes

**Files:**
- Modify: `web/src/components/editor/nodes/UpalNode.tsx`

**Step 1: Add glow CSS classes per node type**

In `web/src/components/editor/nodes/UpalNode.tsx`, add a `glowMap`:

```typescript
const glowMap: Record<string, string> = {
  input: 'shadow-[0_0_16px_oklch(0.795_0.184_86.047/0.4)]',
  agent: 'shadow-[0_0_16px_oklch(0.588_0.158_241.966/0.4)]',
  tool: 'shadow-[0_0_16px_oklch(0.637_0.237_25.331/0.4)]',
  output: 'shadow-[0_0_16px_oklch(0.648_0.2_142.495/0.4)]',
}
```

**Step 2: Apply glow when selected**

Update the outer div's `className` to include the glow on selection:

Change:
```typescript
isSelected && 'ring-2 ring-ring',
```
To:
```typescript
isSelected && `ring-2 ring-ring ${glowMap[data.nodeType]}`,
```

**Step 3: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add web/src/components/editor/nodes/UpalNode.tsx
git commit -m "feat(web): add glow effect on selected nodes by type color"
```

---

### Task 8: Visual verification of Phase 2

**Step 1: Start dev server and verify**

Run from `web/`: `npm run dev`

Open http://localhost:5173 and verify:
1. Canvas is empty initially — empty state visible
2. Add 4 nodes (Input, Agent, Tool, Output) — they should auto-layout **left to right**
3. Connect Input → Agent → Tool → Output — edges appear as **dashed lines with animation**
4. Click a node — **glow effect** visible around the selected node
5. Node cards are wider (~280px) and show a description line
6. Right panel shows Description textarea field
7. Both light and dark themes work correctly
8. Nodes have handles on **left and right** sides (not top/bottom)

**Step 2: Commit any fixups if needed**

---

## Phase 3: Tabbed Right Panel

### Task 9: Install shadcn Tabs component

**Files:**
- Creates: `web/src/components/ui/tabs.tsx`

**Step 1: Install tabs**

Run from `web/`:
```bash
npx shadcn@latest add tabs
```

**Step 2: Verify file created**

Check that `web/src/components/ui/tabs.tsx` exists.

**Step 3: Commit**

```bash
git add web/src/components/ui/tabs.tsx
git commit -m "feat(web): add shadcn tabs component"
```

---

### Task 10: Create RightPanel container with tabs

**Files:**
- Create: `web/src/components/panel/RightPanel.tsx`
- Modify: `web/src/App.tsx`

**Step 1: Create the RightPanel component**

Create `web/src/components/panel/RightPanel.tsx`:

```typescript
import { useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { NodeEditor } from '@/components/editor/nodes/NodeEditor'
import { PanelConsole } from '@/components/panel/PanelConsole'
import { PanelPreview } from '@/components/panel/PanelPreview'
import { Button } from '@/components/ui/button'
import { X, Settings2, Terminal, Eye } from 'lucide-react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { NodeData } from '@/stores/workflowStore'

type RightPanelProps = {
  selectedNode: { id: string; data: NodeData } | null
  onCloseNode: () => void
}

export function RightPanel({ selectedNode, onCloseNode }: RightPanelProps) {
  const [activeTab, setActiveTab] = useState('properties')
  const isRunning = useWorkflowStore((s) => s.isRunning)

  // Auto-switch to console when running
  if (isRunning && activeTab === 'properties') {
    setActiveTab('console')
  }

  return (
    <aside className="w-80 border-l border-border bg-background flex flex-col">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1">
        <div className="flex items-center justify-between border-b border-border px-2">
          <TabsList className="h-10 bg-transparent p-0 gap-0">
            <TabsTrigger
              value="properties"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 text-xs"
            >
              <Settings2 className="h-3.5 w-3.5 mr-1.5" />
              Properties
            </TabsTrigger>
            <TabsTrigger
              value="console"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 text-xs"
            >
              <Terminal className="h-3.5 w-3.5 mr-1.5" />
              Console
            </TabsTrigger>
            <TabsTrigger
              value="preview"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 text-xs"
            >
              <Eye className="h-3.5 w-3.5 mr-1.5" />
              Preview
            </TabsTrigger>
          </TabsList>
          {selectedNode && activeTab === 'properties' && (
            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onCloseNode}>
              <X className="h-4 w-4" />
            </Button>
          )}
        </div>

        <TabsContent value="properties" className="flex-1 overflow-y-auto mt-0">
          {selectedNode ? (
            <NodeEditor
              nodeId={selectedNode.id}
              data={selectedNode.data}
              onClose={onCloseNode}
              embedded
            />
          ) : (
            <div className="flex items-center justify-center h-full text-sm text-muted-foreground p-4">
              Select a node to edit its properties.
            </div>
          )}
        </TabsContent>

        <TabsContent value="console" className="flex-1 overflow-hidden mt-0">
          <PanelConsole />
        </TabsContent>

        <TabsContent value="preview" className="flex-1 overflow-hidden mt-0">
          <PanelPreview />
        </TabsContent>
      </Tabs>
    </aside>
  )
}
```

**Step 2: Update App.tsx to use RightPanel**

In `web/src/App.tsx`:

1. Remove the import for `NodeEditor` and `Console`.
2. Add import for `RightPanel`:
```typescript
import { RightPanel } from '@/components/panel/RightPanel'
```

3. Replace the right panel section and Console in the JSX.

Old:
```tsx
{selectedNode && (
  <NodeEditor
    nodeId={selectedNode.id}
    data={selectedNode.data}
    onClose={() => selectNode(null)}
  />
)}
</div>

<Console />
```

New:
```tsx
<RightPanel
  selectedNode={selectedNode ? { id: selectedNode.id, data: selectedNode.data } : null}
  onCloseNode={() => selectNode(null)}
/>
</div>
```

(Remove `<Console />` entirely — it's now inside RightPanel.)

**Step 3: Verify build (will fail — PanelConsole and PanelPreview don't exist yet)**

This is expected. Continue to next tasks.

---

### Task 11: Create PanelConsole (console tab content)

**Files:**
- Create: `web/src/components/panel/PanelConsole.tsx`

**Step 1: Create PanelConsole**

This is the Console component adapted for the panel tab. Create `web/src/components/panel/PanelConsole.tsx`:

```typescript
import { useEffect, useRef } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'
import type { RunEvent } from '@/stores/workflowStore'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Trash2 } from 'lucide-react'
import { cn } from '@/lib/utils'

const eventColorMap: Record<string, string> = {
  'node.started': 'text-node-agent',
  'node.completed': 'text-node-output',
  'node.error': 'text-destructive',
  'model.request': 'text-purple-500 dark:text-purple-400',
  'model.response': 'text-purple-400 dark:text-purple-300',
  'tool.call': 'text-node-tool',
  'tool.result': 'text-amber-600 dark:text-amber-300',
  done: 'text-node-output font-semibold',
  error: 'text-destructive',
  info: 'text-muted-foreground',
}

function formatEvent(event: RunEvent): string {
  const data = event.data
  if (data.message && typeof data.message === 'string') return data.message
  const parts: string[] = []
  if (data.node_id) parts.push(`[${data.node_id}]`)
  if (data.node_type) parts.push(`(${data.node_type})`)
  if (data.model) parts.push(`model=${data.model}`)
  if (data.tool) parts.push(`tool=${data.tool}`)
  if (data.error) parts.push(`error: ${data.error}`)
  if (data.result !== undefined) parts.push(`result: ${typeof data.result === 'string' ? data.result : JSON.stringify(data.result)}`)
  if (data.output !== undefined) parts.push(`output: ${typeof data.output === 'string' ? data.output : JSON.stringify(data.output)}`)
  if (parts.length === 0) return JSON.stringify(data)
  return parts.join(' ')
}

export function PanelConsole() {
  const runEvents = useWorkflowStore((s) => s.runEvents)
  const isRunning = useWorkflowStore((s) => s.isRunning)
  const clearRunEvents = useWorkflowStore((s) => s.clearRunEvents)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight })
    }
  }, [runEvents])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border">
        <div className="flex items-center gap-2">
          {isRunning && (
            <Badge variant="default" className="animate-pulse text-xs">running</Badge>
          )}
          {!isRunning && runEvents.length > 0 && (
            <Badge variant="secondary" className="text-xs">{runEvents.length} events</Badge>
          )}
        </div>
        {runEvents.length > 0 && (
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={clearRunEvents}>
            <Trash2 className="h-3 w-3" />
          </Button>
        )}
      </div>
      <div ref={scrollRef} className="flex-1 overflow-y-auto px-3 py-2 font-mono text-xs space-y-0.5">
        {runEvents.length === 0 ? (
          <p className="text-muted-foreground">Ready. Run a workflow to see events...</p>
        ) : (
          runEvents.map((event, i) => (
            <div key={i} className={cn(eventColorMap[event.type] ?? 'text-muted-foreground')}>
              <span className="text-muted-foreground mr-2">{event.type}</span>
              {formatEvent(event)}
            </div>
          ))
        )}
      </div>
    </div>
  )
}
```

**Step 2: Commit**

```bash
git add web/src/components/panel/PanelConsole.tsx
git commit -m "feat(web): add PanelConsole for tabbed right panel"
```

---

### Task 12: Create PanelPreview (preview tab content)

**Files:**
- Create: `web/src/components/panel/PanelPreview.tsx`

**Step 1: Create PanelPreview**

Create `web/src/components/panel/PanelPreview.tsx`:

```typescript
import { useWorkflowStore } from '@/stores/workflowStore'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Eye } from 'lucide-react'

export function PanelPreview() {
  const runEvents = useWorkflowStore((s) => s.runEvents)

  // Extract the final "done" event result
  const doneEvent = runEvents.find((e) => e.type === 'done')
  const outputEvents = runEvents.filter(
    (e) => e.type === 'node.completed' && e.data.output,
  )

  if (!doneEvent && outputEvents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-muted-foreground p-6">
        <Eye className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm text-center">
          Run a workflow to see results here.
        </p>
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-4 space-y-4">
        {outputEvents.map((event, i) => (
          <div key={i} className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground">
              {(event.data.node_id as string) || `Step ${i + 1}`}
            </p>
            <div className="rounded-lg border border-border bg-card p-3 text-sm whitespace-pre-wrap">
              {typeof event.data.output === 'string'
                ? event.data.output
                : JSON.stringify(event.data.output, null, 2)}
            </div>
          </div>
        ))}
        {doneEvent && (
          <div className="space-y-1">
            <p className="text-xs font-medium text-node-output">Final Result</p>
            <div className="rounded-lg border border-node-output/30 bg-node-output/5 p-3 text-sm whitespace-pre-wrap">
              {typeof doneEvent.data.result === 'string'
                ? doneEvent.data.result
                : JSON.stringify(doneEvent.data, null, 2)}
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  )
}
```

**Step 2: Commit**

```bash
git add web/src/components/panel/PanelPreview.tsx
git commit -m "feat(web): add PanelPreview for workflow result display"
```

---

### Task 13: Update NodeEditor for embedded mode

**Files:**
- Modify: `web/src/components/editor/nodes/NodeEditor.tsx`

**Step 1: Add `embedded` prop to NodeEditor**

The NodeEditor currently wraps itself in an `<aside>`. When embedded inside RightPanel, it should only render the form content without the aside wrapper.

Update the props:
```typescript
type NodeEditorProps = {
  nodeId: string
  data: NodeData
  onClose: () => void
  embedded?: boolean
}
```

Update the component:
```typescript
export function NodeEditor({ nodeId, data, onClose, embedded }: NodeEditorProps) {
```

When `embedded` is true, skip the outer `<aside>` wrapper and the header (since RightPanel already has tabs + close button):

```typescript
const formContent = (
  <div className="p-4 space-y-4">
    {/* Label field */}
    <div className="space-y-2">
      <Label htmlFor="node-label">Label</Label>
      <Input
        id="node-label"
        value={data.label}
        onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
      />
    </div>

    {/* Description field */}
    <div className="space-y-2">
      <Label htmlFor="node-description">Description</Label>
      <Textarea
        id="node-description"
        className="min-h-[60px] resize-y"
        value={data.description ?? ''}
        placeholder="Describe what this node does..."
        onChange={(e) => updateNodeDescription(nodeId, e.target.value)}
      />
    </div>

    {/* ...rest of type-specific fields unchanged... */}
  </div>
)

if (embedded) {
  return formContent
}

return (
  <aside className="w-80 border-l border-border bg-background p-0 overflow-y-auto">
    {/* Panel header */}
    <div className="flex items-center justify-between p-4 border-b border-border">
      <div className="flex items-center gap-2">
        {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
        <h3 className="text-sm font-semibold">Properties</h3>
      </div>
      <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
        <X className="h-4 w-4" />
      </Button>
    </div>
    {formContent}
  </aside>
)
```

**Step 2: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add web/src/components/editor/nodes/NodeEditor.tsx web/src/components/panel/RightPanel.tsx web/src/App.tsx
git commit -m "feat(web): integrate tabbed right panel replacing standalone console"
```

---

### Task 14: Remove old Console component

**Files:**
- Delete: `web/src/components/console/Console.tsx`
- Verify: no remaining imports of the old Console

**Step 1: Delete the old Console**

```bash
rm web/src/components/console/Console.tsx
rmdir web/src/components/console 2>/dev/null || true
```

**Step 2: Verify no broken imports**

Run from `web/`: `npm run build`
Expected: Build succeeds with no errors. If the old Console is still imported anywhere, fix those imports.

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor(web): remove old standalone Console component"
```

---

### Task 15: Always show RightPanel (never hide)

**Files:**
- Modify: `web/src/App.tsx`

**Step 1: Make RightPanel always visible**

In `web/src/App.tsx`, the RightPanel should always render (not conditional on selectedNode). It already does based on Task 10, but verify the JSX looks like:

```tsx
<div className="flex flex-1 overflow-hidden">
  <NodePalette onAddNode={handleAddNode} />

  <main className="flex-1">
    <Canvas
      onAddFirstNode={() => handleAddNode('input')}
      onGenerate={() => setShowGenerateDialog(true)}
    />
  </main>

  <RightPanel
    selectedNode={selectedNode ? { id: selectedNode.id, data: selectedNode.data } : null}
    onCloseNode={() => selectNode(null)}
  />
</div>
```

No conditional rendering. The panel shows "Select a node to edit" placeholder when no node is selected.

**Step 2: Verify build and visual**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 3: Commit (if any changes were needed)**

```bash
git add web/src/App.tsx
git commit -m "feat(web): always show right panel with tab navigation"
```

---

### Task 16: Visual verification of Phase 3

**Step 1: Verify in browser**

Open http://localhost:5173 and verify:
1. Right panel is always visible with 3 tabs: Properties, Console, Preview
2. Click Properties tab — shows "Select a node to edit" placeholder
3. Click a node — Properties tab shows the edit form
4. Click Console tab — shows console events
5. Click Preview tab — shows "Run a workflow to see results"
6. Run a workflow — console auto-switches, events stream in
7. After run completes — Preview tab shows output results
8. Light/dark themes both work with the tabbed panel

---

## Phase 4: Polish & Micro-Interactions

### Task 17: Refine dark mode colors (Deep Navy)

**Files:**
- Modify: `web/src/index.css`

**Step 1: Change dark mode background from pure black to deep navy**

In `web/src/index.css`, update `.dark` section:

```css
.dark {
  --background: oklch(0.16 0.015 260);
  --foreground: oklch(0.985 0 0);
  --card: oklch(0.21 0.015 260);
  --card-foreground: oklch(0.985 0 0);
  --popover: oklch(0.25 0.012 260);
  --popover-foreground: oklch(0.985 0 0);
  --primary: oklch(0.922 0 0);
  --primary-foreground: oklch(0.205 0 0);
  --secondary: oklch(0.25 0.012 260);
  --secondary-foreground: oklch(0.985 0 0);
  --muted: oklch(0.25 0.012 260);
  --muted-foreground: oklch(0.708 0 0);
  --accent: oklch(0.33 0.012 260);
  --accent-foreground: oklch(0.985 0 0);
  --destructive: oklch(0.704 0.191 22.216);
  --destructive-foreground: oklch(0.985 0 0);
  --border: oklch(1 0 0 / 10%);
  --input: oklch(1 0 0 / 15%);
  --ring: oklch(0.556 0 0);

  --sidebar: oklch(0.19 0.015 260);
  --sidebar-foreground: oklch(0.985 0 0);
  --sidebar-primary: oklch(0.488 0.243 264.376);
  --sidebar-primary-foreground: oklch(0.985 0 0);
  --sidebar-accent: oklch(0.25 0.012 260);
  --sidebar-accent-foreground: oklch(0.985 0 0);
  --sidebar-border: oklch(1 0 0 / 10%);
  --sidebar-ring: oklch(0.439 0 0);

  /* Node-type semantic colors — Dark (same as before) */
  --node-input: oklch(0.795 0.184 86.047);
  --node-input-foreground: oklch(0.985 0 0);
  --node-agent: oklch(0.588 0.158 241.966);
  --node-agent-foreground: oklch(0.985 0 0);
  --node-tool: oklch(0.637 0.237 25.331);
  --node-tool-foreground: oklch(0.985 0 0);
  --node-output: oklch(0.648 0.2 142.495);
  --node-output-foreground: oklch(0.985 0 0);
}
```

Key changes: `background` 0.145→0.16 with chroma 0.015 and hue 260 (navy), `card` 0.205→0.21, `popover` 0.269→0.25, `secondary/muted` 0.269→0.25, `accent` 0.371→0.33, `sidebar` 0.205→0.19. All with subtle blue-navy tint.

**Step 2: Verify build and visuals**

Run from `web/`: `npm run build`
Toggle to dark mode and verify the background has a subtle navy tint instead of pure black.

**Step 3: Commit**

```bash
git add web/src/index.css
git commit -m "style(web): refine dark mode to deep navy tone for premium feel"
```

---

### Task 18: Drag-and-drop from sidebar to canvas

**Files:**
- Modify: `web/src/components/sidebar/NodePalette.tsx`
- Modify: `web/src/components/editor/Canvas.tsx`
- Modify: `web/src/stores/workflowStore.ts`

**Step 1: Make palette items draggable**

In `web/src/components/sidebar/NodePalette.tsx`, add `draggable` and `onDragStart` to each button:

```typescript
<button
  draggable
  onDragStart={(e) => {
    e.dataTransfer.setData('application/upal-node-type', item.type)
    e.dataTransfer.effectAllowed = 'move'
  }}
  onClick={() => onAddNode(item.type)}
  className={cn(
    'flex items-center gap-3 rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors cursor-grab active:cursor-grabbing',
    item.colorClass
  )}
>
```

**Step 2: Handle drop on Canvas**

In `web/src/components/editor/Canvas.tsx`, add `onDrop` and `onDragOver` to the ReactFlow wrapper:

Add import:
```typescript
import { useCallback, type DragEvent } from 'react'
```

Update props:
```typescript
type CanvasProps = {
  onAddFirstNode: () => void
  onGenerate: () => void
  onDropNode: (type: string, position: { x: number; y: number }) => void
}
```

Add handlers inside the component:
```typescript
const onDragOver = useCallback((e: DragEvent) => {
  e.preventDefault()
  e.dataTransfer.dropEffect = 'move'
}, [])

const onDrop = useCallback(
  (e: DragEvent) => {
    e.preventDefault()
    const type = e.dataTransfer.getData('application/upal-node-type')
    if (!type) return

    // Get canvas bounds to calculate position
    const bounds = e.currentTarget.getBoundingClientRect()
    const position = {
      x: e.clientX - bounds.left,
      y: e.clientY - bounds.top,
    }
    onDropNode(type, position)
  },
  [onDropNode],
)
```

Add to ReactFlow:
```tsx
<ReactFlow
  nodes={nodes}
  edges={edges}
  onNodesChange={onNodesChange}
  onEdgesChange={onEdgesChange}
  onConnect={onConnect}
  nodeTypes={nodeTypes}
  onDrop={onDrop}
  onDragOver={onDragOver}
  fitView
  className="bg-background"
>
```

**Step 3: Wire up in App.tsx**

In `web/src/App.tsx`, add the `handleDropNode` handler:

```typescript
const handleDropNode = (type: string, position: { x: number; y: number }) => {
  addNode(type as 'input' | 'agent' | 'tool' | 'output', position)
}
```

Update Canvas usage:
```tsx
<Canvas
  onAddFirstNode={() => handleAddNode('input')}
  onGenerate={() => setShowGenerateDialog(true)}
  onDropNode={handleDropNode}
/>
```

**Step 4: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 5: Commit**

```bash
git add web/src/components/sidebar/NodePalette.tsx web/src/components/editor/Canvas.tsx web/src/App.tsx
git commit -m "feat(web): add drag-and-drop from sidebar palette to canvas"
```

---

### Task 19: Keyboard shortcuts

**Files:**
- Create: `web/src/hooks/useKeyboardShortcuts.ts`
- Modify: `web/src/App.tsx`

**Step 1: Create keyboard shortcuts hook**

Create `web/src/hooks/useKeyboardShortcuts.ts`:

```typescript
import { useEffect } from 'react'
import { useWorkflowStore } from '@/stores/workflowStore'

type ShortcutHandlers = {
  onSave: () => void
  onRun: () => void
  onGenerate: () => void
}

export function useKeyboardShortcuts({ onSave, onRun, onGenerate }: ShortcutHandlers) {
  const selectNode = useWorkflowStore((s) => s.selectNode)
  const selectedNodeId = useWorkflowStore((s) => s.selectedNodeId)
  const nodes = useWorkflowStore((s) => s.nodes)
  const onNodesChange = useWorkflowStore((s) => s.onNodesChange)

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      const isInput =
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement ||
        e.target instanceof HTMLSelectElement

      // Ctrl/Cmd + S = Save
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        onSave()
        return
      }

      // Ctrl/Cmd + Enter = Run
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault()
        onRun()
        return
      }

      // Ctrl/Cmd + G = Generate
      if ((e.ctrlKey || e.metaKey) && e.key === 'g') {
        e.preventDefault()
        onGenerate()
        return
      }

      // Skip the rest if user is in an input field
      if (isInput) return

      // Escape = Deselect node
      if (e.key === 'Escape') {
        selectNode(null)
        return
      }

      // Delete/Backspace = Delete selected node
      if ((e.key === 'Delete' || e.key === 'Backspace') && selectedNodeId) {
        onNodesChange([{ id: selectedNodeId, type: 'remove' }])
        selectNode(null)
        return
      }
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [onSave, onRun, onGenerate, selectNode, selectedNodeId, nodes, onNodesChange])
}
```

**Step 2: Use in App.tsx**

In `web/src/App.tsx`, add:

```typescript
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
```

Inside the component, after the handler declarations:
```typescript
useKeyboardShortcuts({
  onSave: handleSave,
  onRun: () => !isRunning && setShowRunDialog(true),
  onGenerate: () => setShowGenerateDialog(true),
})
```

**Step 3: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add web/src/hooks/useKeyboardShortcuts.ts web/src/App.tsx
git commit -m "feat(web): add keyboard shortcuts (Ctrl+S, Ctrl+Enter, Del, Esc)"
```

---

### Task 20: Transition animations

**Files:**
- Modify: `web/src/index.css`

**Step 1: Add subtle CSS transitions**

In `web/src/index.css`, add after the body rule:

```css
/* Smooth panel transitions */
aside {
  transition: width 200ms ease, opacity 200ms ease;
}

/* Node enter animation */
.react-flow__node {
  animation: nodeEnter 200ms ease-out;
}

@keyframes nodeEnter {
  from {
    opacity: 0;
    transform: scale(0.9);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
```

**Step 2: Verify build**

Run from `web/`: `npm run build`
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add web/src/index.css
git commit -m "style(web): add subtle transition animations for nodes and panels"
```

---

### Task 21: Final build verification and visual check

**Step 1: Full build**

Run from `web/`:
```bash
npm run build
```
Expected: Build succeeds with zero errors.

**Step 2: Visual verification checklist**

Open http://localhost:5173 and verify ALL of these:

**Phase 2 checks:**
- [ ] Nodes flow left-to-right (horizontal)
- [ ] Node cards are ~280px wide with description text
- [ ] Edges are dashed with animation
- [ ] Handles are on left/right sides
- [ ] Selected node has type-colored glow effect
- [ ] Description field appears in properties

**Phase 3 checks:**
- [ ] Right panel always visible with 3 tabs
- [ ] Properties tab shows form when node selected
- [ ] Console tab shows events
- [ ] Preview tab shows results after run
- [ ] Tab auto-switches to Console when workflow runs

**Phase 4 checks:**
- [ ] Dark mode has subtle navy tint (not pure black)
- [ ] Drag nodes from sidebar to canvas works
- [ ] Ctrl+S triggers save
- [ ] Ctrl+Enter opens run dialog
- [ ] Delete key removes selected node
- [ ] Escape deselects node
- [ ] Nodes appear with fade-in animation

**Step 3: Final commit**

If everything passes, no additional commit needed.

---

## Summary

| Task | Phase | Description |
|------|-------|-------------|
| 1 | 2 | Install dagre |
| 2 | 2 | Auto-layout utility |
| 3 | 2 | Store + horizontal flow integration |
| 4 | 2 | Left/Right handles |
| 5 | 2 | Larger cards + description field |
| 6 | 2 | Dashed animated edges |
| 7 | 2 | Selection glow effect |
| 8 | 2 | Phase 2 visual verification |
| 9 | 3 | Install shadcn Tabs |
| 10 | 3 | RightPanel container |
| 11 | 3 | PanelConsole component |
| 12 | 3 | PanelPreview component |
| 13 | 3 | NodeEditor embedded mode |
| 14 | 3 | Remove old Console |
| 15 | 3 | Always-visible panel |
| 16 | 3 | Phase 3 visual verification |
| 17 | 4 | Deep navy dark mode |
| 18 | 4 | Drag-and-drop |
| 19 | 4 | Keyboard shortcuts |
| 20 | 4 | Transition animations |
| 21 | 4 | Final verification |
