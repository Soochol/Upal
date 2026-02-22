# Console Tab Step View Redesign

**Date:** 2026-02-23
**Status:** Approved

## Goal

Redesign the `PanelConsole` (Console tab in RightPanel) from a terminal-style log view into a visual step card view that shows each node as a colored card, displays run status in real time, and renders node output inline after completion.

## Scope

- **File changed:** `web/src/widgets/right-panel/ui/PanelConsole.tsx` (full rewrite)
- **New helper:** `web/src/widgets/right-panel/ui/console/detectOutputKind.ts`
- **New component:** `web/src/widgets/right-panel/ui/console/NodeOutputViewer.tsx`

## UI Layout

```
┌──────────────────────────────────────┐
│ toolbar: mode label + [trash]        │
├──────────────────────────────────────┤
│ ▋ [Icon] Kpop Query      ✓  [▶]     │  ← input, yellow border, completed
│   "K-pop query text..."              │  ← auto-expanded output
├──────────────────────────────────────┤
│ ▋ [Icon] Generate Fanchants  ⠿  [—] │  ← agent, blue, running, ▶ disabled
├──────────────────────────────────────┤
│ ▋ [Icon] Generate Banner Image   [—] │  ← idle, disabled
│ ▋ [Icon] Generate Digital Banner [—] │
│ ▋ [Icon] Cheering Webpage        [—] │
└──────────────────────────────────────┘
```

## Node Card Design

| Element | Detail |
|---------|--------|
| Left border | 3px solid `cssVar` of node type |
| Background | `bg-[cssVar]/8` tint |
| Icon | From `NodeTypeDefinition.icon` |
| Label | `node.data.label` |
| Status badge | ✓ (completed), × (error), spinner (running), none (idle) |
| ▶ button | Enabled only when `inDegree === 0` (no incoming edges) |
| Chevron | Expand/collapse output area |

## Play Button Behavior

- **Enabled condition:** Node has zero incoming edges in the workflow (root node).
  In practice this is always `input` type nodes.
- **On click:** Collect all input nodes' `config.value` → call `executeRun(inputs)` immediately. No extra prompt.
- **While running:** ▶ button disabled for all nodes.

## Node Ordering

DAG topological sort of all non-group nodes using Kahn's algorithm (same as `getInputNodesInOrder` but returns all node types).

## Status → Visual Mapping

| `nodeStatuses[id]` | Visual |
|--------------------|--------|
| `idle` | No badge, muted card |
| `running` | Left border pulse animation, spinner icon |
| `completed` | ✓ badge in node color, auto-expand output |
| `error` | × badge in destructive color |
| `skipped` | `—` badge, muted |
| `waiting` | Pause icon |

## Output Rendering

`NodeCompletedEvent.output` string → detect type → render:

| Detection | Render |
|-----------|--------|
| `data:image/...` or image URL (`.png`, `.jpg`, `.gif`, `.webp`, `.svg`) | `<img>` |
| `/api/files/` URL | `<img>` (file serve endpoint = image) |
| `data:audio/...` or audio URL (`.mp3`, `.wav`, `.ogg`) | `<audio controls>` |
| Starts with `<!doctype` or `<html` | `<iframe srcDoc>` + Render/Source toggle |
| Valid JSON | `<pre>` formatted |
| Else | `<pre>` plain text, max-height scrollable |

Output is auto-expanded when node status transitions to `completed`. User can re-collapse with chevron.

## Data Sources

- `useWorkflowStore` → `nodes`, `edges` (for card list + in-degree calc)
- `useExecutionStore` → `nodeStatuses`, `runEvents`, `isRunning`
- `NodeCompletedEvent.output` → extracted per nodeId from `runEvents`
- `getNodeDefinition(nodeType)` → icon, cssVar, label
- `useExecuteRun()` → `executeRun`, `isRunning`

## Component Structure

```
PanelConsole
├── toolbar (clear button)
├── scroll container
│   └── NodeStepCard[] (one per non-group node, topo order)
│       ├── card header (chevron, icon, label, status, play btn)
│       └── output area (NodeOutputViewer, shown when expanded)
│           └── detectOutputKind() → renders img/audio/iframe/pre
```

## Files

| File | Action |
|------|--------|
| `PanelConsole.tsx` | Full rewrite |
| `console/detectOutputKind.ts` | New — output type detection util |
| `console/NodeOutputViewer.tsx` | New — renders output by kind |

No backend changes required. No new API endpoints.
