# Console Step View Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign the Console tab in RightPanel from a terminal log view into a visual node step-card view that shows run status in real time and renders output inline after each node completes.

**Architecture:** Replace `PanelConsole.tsx` entirely. Add two small helpers in a `console/` subdirectory: `detectOutputKind.ts` (pure function, detects image/audio/html/json/text from a string) and `NodeOutputViewer.tsx` (renders by kind). The main component reads from `useWorkflowStore` (nodes + edges) and `useExecutionStore` (statuses + events), sorts nodes by DAG topological order, and renders one `NodeStepCard` per node.

**Tech Stack:** React 19, TypeScript, Tailwind CSS v4, Zustand, `@xyflow/react` (Node/Edge types), Lucide icons. No new deps.

---

### Task 1: Create `detectOutputKind` utility

**Files:**
- Create: `web/src/widgets/right-panel/ui/console/detectOutputKind.ts`

**Step 1: Create the file**

```typescript
// web/src/widgets/right-panel/ui/console/detectOutputKind.ts

export type OutputKind = 'image' | 'audio' | 'html' | 'json' | 'text'

export function detectOutputKind(output: string): OutputKind {
  const t = output.trim()
  if (!t) return 'text'

  // Image: data URI, image-extension URL, or /api/files/ serve path
  if (t.startsWith('data:image/')) return 'image'
  if (/^https?:\/\/.+\.(png|jpg|jpeg|gif|webp|svg)(\?.*)?$/i.test(t)) return 'image'
  if (/^\/api\/files\/[^/]+\/serve/.test(t)) return 'image'

  // Audio: data URI or audio-extension URL
  if (t.startsWith('data:audio/')) return 'audio'
  if (/^https?:\/\/.+\.(mp3|wav|ogg|m4a)(\?.*)?$/i.test(t)) return 'audio'

  // HTML
  const tl = t.toLowerCase()
  if (tl.startsWith('<!doctype') || tl.startsWith('<html')) return 'html'

  // JSON (only attempt if it starts with { or [)
  if (t.startsWith('{') || t.startsWith('[')) {
    try { JSON.parse(t); return 'json' } catch { /* not json */ }
  }

  return 'text'
}
```

**Step 2: Type-check**

```bash
cd web && npx tsc -b --noEmit 2>&1 | head -40
```
Expected: no errors about the new file.

**Step 3: Commit**

```bash
git add web/src/widgets/right-panel/ui/console/detectOutputKind.ts
git commit -m "feat(console): add detectOutputKind utility"
```

---

### Task 2: Create `NodeOutputViewer` component

**Files:**
- Create: `web/src/widgets/right-panel/ui/console/NodeOutputViewer.tsx`

**Step 1: Create the file**

```tsx
// web/src/widgets/right-panel/ui/console/NodeOutputViewer.tsx
import { useState } from 'react'
import { Code } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { detectOutputKind } from './detectOutputKind'

type Props = { output: string }

export function NodeOutputViewer({ output }: Props) {
  const [showSource, setShowSource] = useState(false)
  const kind = detectOutputKind(output)

  if (kind === 'image') {
    return (
      <div className="p-2">
        <img
          src={output.trim()}
          alt="node output"
          className="max-w-full rounded-md border border-border"
          onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
        />
      </div>
    )
  }

  if (kind === 'audio') {
    return (
      <div className="p-2">
        <audio controls src={output.trim()} className="w-full h-8" />
      </div>
    )
  }

  if (kind === 'html') {
    return (
      <div className="p-2">
        <div className="flex items-center gap-1 mb-1.5">
          {(['render', 'source'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => setShowSource(mode === 'source')}
              className={cn(
                'flex items-center gap-1 text-[10px] px-2 py-0.5 rounded-md transition-colors cursor-pointer',
                (mode === 'source') === showSource
                  ? 'bg-foreground text-background'
                  : 'text-muted-foreground hover:text-foreground hover:bg-muted',
              )}
            >
              {mode === 'source' && <Code className="h-2.5 w-2.5" />}
              {mode === 'render' ? 'Render' : 'Source'}
            </button>
          ))}
        </div>
        {showSource ? (
          <pre className="text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-64 overflow-auto whitespace-pre-wrap break-all">
            {output}
          </pre>
        ) : (
          <iframe
            srcDoc={output}
            sandbox="allow-same-origin"
            className="w-full rounded-lg border border-border bg-white dark:bg-card"
            style={{ height: '280px' }}
            title="Rendered output"
          />
        )}
      </div>
    )
  }

  if (kind === 'json') {
    let pretty = output
    try { pretty = JSON.stringify(JSON.parse(output), null, 2) } catch { /* keep raw */ }
    return (
      <pre className="m-2 text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-64 overflow-auto whitespace-pre-wrap break-all">
        {pretty}
      </pre>
    )
  }

  // text
  return (
    <pre className="m-2 text-[11px] font-mono bg-muted/30 rounded-lg p-3 max-h-64 overflow-auto whitespace-pre-wrap break-words">
      {output}
    </pre>
  )
}
```

**Step 2: Type-check**

```bash
cd web && npx tsc -b --noEmit 2>&1 | head -40
```
Expected: no errors.

**Step 3: Commit**

```bash
git add web/src/widgets/right-panel/ui/console/NodeOutputViewer.tsx
git commit -m "feat(console): add NodeOutputViewer component"
```

---

### Task 3: Rewrite `PanelConsole.tsx`

**Files:**
- Modify: `web/src/widgets/right-panel/ui/PanelConsole.tsx`

**Step 1: Replace file contents**

```tsx
// web/src/widgets/right-panel/ui/PanelConsole.tsx
import { useMemo, useState, useCallback, useEffect } from 'react'
import type { Node, Edge } from '@xyflow/react'
import { useWorkflowStore } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { useExecuteRun } from '@/features/execute-workflow'
import { getNodeDefinition } from '@/entities/node'
import { Button } from '@/shared/ui/button'
import {
  Trash2, ChevronDown, Play, Loader2,
  Check, X, Pause, SkipForward,
} from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { NodeOutputViewer } from './console/NodeOutputViewer'
import type { NodeCompletedEvent, NodeRunStatus } from '@/entities/run'

// ── Topological sort of ALL non-group nodes ──────────────────────────────────
// Returns each node paired with its in-degree (0 = root node, can be run directly).

type SortedEntry = { node: Node<NodeData>; inDegree: number }

function sortAllNodesTopologically(nodes: Node<NodeData>[], edges: Edge[]): SortedEntry[] {
  const regularNodes = nodes.filter((n) => n.type !== 'groupNode')
  const inDegreeMap = new Map<string, number>()
  const adj = new Map<string, string[]>()

  for (const node of regularNodes) {
    inDegreeMap.set(node.id, 0)
    adj.set(node.id, [])
  }
  for (const edge of edges) {
    if (adj.has(edge.source)) {
      adj.get(edge.source)!.push(edge.target)
      inDegreeMap.set(edge.target, (inDegreeMap.get(edge.target) ?? 0) + 1)
    }
  }

  // Kahn's algorithm
  const tempDeg = new Map(inDegreeMap)
  const queue: string[] = []
  for (const [id, deg] of tempDeg) {
    if (deg === 0) queue.push(id)
  }
  const sorted: string[] = []
  while (queue.length > 0) {
    const id = queue.shift()!
    sorted.push(id)
    for (const next of adj.get(id) ?? []) {
      const d = (tempDeg.get(next) ?? 1) - 1
      tempDeg.set(next, d)
      if (d === 0) queue.push(next)
    }
  }

  const nodeMap = new Map(regularNodes.map((n) => [n.id, n]))
  return sorted
    .filter((id) => nodeMap.has(id))
    .map((id) => ({ node: nodeMap.get(id)!, inDegree: inDegreeMap.get(id) ?? 0 }))
}

// ── NodeStepCard ──────────────────────────────────────────────────────────────

type NodeStepCardProps = {
  node: Node<NodeData>
  inDegree: number
  status: NodeRunStatus
  output: string | null
  isRunning: boolean
  onPlay: () => void
}

function StatusIcon({ status }: { status: NodeRunStatus; cssVar: string }) {
  if (status === 'running') return <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
  if (status === 'error')   return <X className="h-3 w-3 text-destructive" />
  if (status === 'skipped') return <SkipForward className="h-3 w-3 text-muted-foreground/40" />
  if (status === 'waiting') return <Pause className="h-3 w-3 text-warning" />
  return null
}

function NodeStepCard({ node, inDegree, status, output, isRunning, onPlay }: NodeStepCardProps) {
  const [expanded, setExpanded] = useState(false)

  const nodeType = node.data.nodeType as 'input' | 'agent' | 'output' | 'asset'
  let def
  try { def = getNodeDefinition(nodeType) } catch { return null }
  const Icon = def.icon
  const cssVar = def.cssVar

  // Auto-expand when a completed output arrives
  useEffect(() => {
    if (status === 'completed' && output) setExpanded(true)
  }, [status, output])

  const canPlay = inDegree === 0 && !isRunning
  const isCompleted = status === 'completed'

  return (
    <div
      className={cn(
        'rounded-lg border border-border/50 overflow-hidden transition-all',
        status === 'running' && 'shadow-sm',
      )}
      style={{
        borderLeftWidth: '3px',
        borderLeftColor: cssVar,
      }}
    >
      {/* Card header */}
      <div
        className="flex items-center gap-2 px-2.5 py-2 cursor-pointer select-none"
        style={{ background: `color-mix(in oklch, ${cssVar} 8%, transparent)` }}
        onClick={() => setExpanded((p) => !p)}
      >
        <ChevronDown
          className={cn(
            'h-3 w-3 text-muted-foreground/40 transition-transform shrink-0',
            expanded && 'rotate-180',
          )}
        />

        <Icon className="h-3.5 w-3.5 shrink-0" style={{ color: cssVar }} />

        <span className="text-xs font-medium flex-1 truncate leading-tight">
          {node.data.label}
        </span>

        {/* Status indicator */}
        {isCompleted ? (
          <Check className="h-3 w-3 shrink-0" style={{ color: cssVar }} />
        ) : (
          <StatusIcon status={status} cssVar={cssVar} />
        )}

        {/* Play button */}
        <button
          onClick={(e) => { e.stopPropagation(); if (canPlay) onPlay() }}
          disabled={!canPlay}
          className={cn(
            'h-5 w-5 rounded-full flex items-center justify-center transition-all shrink-0',
            canPlay
              ? 'hover:opacity-80 cursor-pointer'
              : 'opacity-20 cursor-not-allowed',
          )}
          style={{ background: canPlay ? cssVar : undefined }}
          title={canPlay ? 'Run workflow' : 'Has upstream dependencies'}
        >
          <Play
            className="h-2.5 w-2.5 ml-px"
            style={{ color: canPlay ? 'white' : undefined }}
          />
        </button>
      </div>

      {/* Expanded output or running indicator */}
      {expanded && (
        <div className="border-t border-border/30">
          {output ? (
            <NodeOutputViewer output={output} />
          ) : status === 'running' ? (
            <div className="px-3 py-2 flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="h-3 w-3 animate-spin" />
              Running…
            </div>
          ) : (
            <p className="px-3 py-2 text-xs text-muted-foreground/50">No output yet.</p>
          )}
        </div>
      )}
    </div>
  )
}

// ── PanelConsole ──────────────────────────────────────────────────────────────

export function PanelConsole() {
  const nodes    = useWorkflowStore((s) => s.nodes)
  const edges    = useWorkflowStore((s) => s.edges)
  const runEvents      = useExecutionStore((s) => s.runEvents)
  const nodeStatuses   = useExecutionStore((s) => s.nodeStatuses)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const { executeRun, isRunning } = useExecuteRun()

  const sortedNodes = useMemo(() => sortAllNodesTopologically(nodes, edges), [nodes, edges])

  // Map nodeId → completed output string
  const nodeOutputs = useMemo(() => {
    const map: Record<string, string> = {}
    for (const event of runEvents) {
      if (event.type === 'node_completed') {
        const c = event as NodeCompletedEvent
        if (c.output) map[c.nodeId] = c.output
      }
    }
    return map
  }, [runEvents])

  const handlePlay = useCallback(() => {
    if (isRunning) return
    const inputs: Record<string, string> = {}
    for (const n of nodes) {
      if (n.data.nodeType === 'input') {
        inputs[n.id] = (n.data.config.value as string) ?? ''
      }
    }
    clearNodeStatuses()
    executeRun(inputs)
  }, [nodes, isRunning, executeRun, clearNodeStatuses])

  const handleClear = useCallback(() => {
    clearRunEvents()
    clearNodeStatuses()
  }, [clearRunEvents, clearNodeStatuses])

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-border shrink-0">
        <span className="text-xs font-medium text-muted-foreground">Steps</span>
        {runEvents.length > 0 && (
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={handleClear} title="Clear">
            <Trash2 className="h-3 w-3" />
          </Button>
        )}
      </div>

      {/* Node card list */}
      <div className="flex-1 overflow-y-auto p-2 space-y-1.5">
        {sortedNodes.length === 0 ? (
          <p className="text-xs text-muted-foreground px-1 py-2">
            Add nodes to the canvas to see them here.
          </p>
        ) : (
          sortedNodes.map(({ node, inDegree }) => (
            <NodeStepCard
              key={node.id}
              node={node}
              inDegree={inDegree}
              status={(nodeStatuses[node.id] ?? 'idle') as NodeRunStatus}
              output={nodeOutputs[node.id] ?? null}
              isRunning={isRunning}
              onPlay={handlePlay}
            />
          ))
        )}
      </div>
    </div>
  )
}
```

**Note on the `NodeRunStatus` import:** It is exported from `@/entities/run` (re-exported from `model/store.ts`). Check `web/src/entities/run/index.ts` to confirm the export is present; if not, add it.

**Step 2: Verify `NodeRunStatus` is exported from `@/entities/run`**

Open `web/src/entities/run/index.ts`. If `NodeRunStatus` is not listed, add:
```typescript
export type { NodeRunStatus } from './model/store'
```

**Step 3: Type-check**

```bash
cd web && npx tsc -b --noEmit 2>&1 | head -60
```
Expected: no errors. Fix any type errors before proceeding.

**Step 4: Lint check**

```bash
cd web && npm run lint 2>&1 | tail -20
```
Expected: no new errors.

**Step 5: Commit**

```bash
git add web/src/widgets/right-panel/ui/PanelConsole.tsx \
        web/src/entities/run/index.ts
git commit -m "feat(console): redesign Console tab as step card view with output rendering"
```

---

### Task 4: Smoke test in browser

**Step 1: Start dev server**

```bash
make dev-backend   # terminal 1
make dev-frontend  # terminal 2
```

**Step 2: Open the editor at http://localhost:5173**

**Step 3: Manual checks**

| Action | Expected |
|--------|----------|
| Open editor, click Console tab in right panel | Shows "Steps" header + one card per node in DAG order |
| Each node card: left border color | Matches node type color (yellow=input, blue=agent, green=output) |
| Input node ▶ button | Solid colored circle, clickable |
| Agent/output node ▶ button | Faded, not clickable |
| Click ▶ on input node | All cards show status (running spinner on active nodes) |
| After completion | Cards with output auto-expand; output renders correctly |
| Text output | `<pre>` with text |
| HTML output | iframe with Render/Source toggle |
| Image URL output | `<img>` element |
| Audio URL output | `<audio controls>` |
| Click trash icon | Clears all statuses and outputs |
| Click chevron | Collapses/expands output area |

---

### Task 5: Final commit

```bash
git add docs/plans/2026-02-23-console-step-view-plan.md \
        docs/plans/2026-02-23-console-step-view-design.md
git commit -m "docs: add console step view design and implementation plan"
```
