# Frontend FSD Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure the React frontend from a flat `components/stores/hooks/lib` layout to Feature-Sliced Design (FSD) with strict layer boundaries and 4 design patterns applied.

**Architecture:** FSD layers `app → pages → widgets → features → entities → shared`. Each slice exposes only a public API via `index.ts`. Cross-store orchestration moves from stores to feature hooks. Node type registration becomes a registry pattern; SSE dispatch becomes an event bus.

**Tech Stack:** React 19, TypeScript, Zustand 5, React Flow (`@xyflow/react`), Tailwind v4, Vite 7

**Design doc:** `docs/plans/2026-02-22-frontend-fsd-architecture-design.md`

---

## Verification commands (run after each task)

```bash
cd web && npx tsc -b --noEmit   # type-check only (fast)
cd web && npm run build          # full build (slow — run after major milestones)
```

---

## Task 1: Scaffold directory structure

**Files:**
- Create directories and empty `index.ts` placeholders

**Step 1: Create all FSD directories**

```bash
mkdir -p web/src/{app,pages/{editor,landing,runs,pipelines,connections},widgets/{workflow-canvas/ui,right-panel/ui,node-palette/ui,bottom-console/ui,workflow-header/ui,run-detail/ui,pipeline-editor/ui},features/{execute-workflow/model,edit-node/{ui,model},generate-workflow/{ui,model,api},manage-canvas/model,upload-asset/{model,api},generate-pipeline/{ui,model,api}},entities/{workflow/{model,api,lib},run/{model,api},node/model,ui/model,pipeline/{model,api}},shared/{api,ui,lib,types}}
```

**Step 2: Verify structure exists**

```bash
ls web/src/entities web/src/features web/src/widgets web/src/shared
```
Expected: all 4 directories listed.

**Step 3: Install eslint-plugin-boundaries**

```bash
cd web && npm install --save-dev eslint-plugin-boundaries
```

**Step 4: Commit**

```bash
git add web/src/entities web/src/features web/src/widgets web/src/shared web/src/app web/src/pages web/package.json web/package-lock.json
git commit -m "chore(frontend): scaffold FSD directory structure"
```

---

## Task 2: shared/ — API client, types, utilities

Move shared infrastructure with no behavioral changes.

**Files:**
- Create: `web/src/shared/api/client.ts`
- Create: `web/src/shared/api/index.ts`
- Create: `web/src/shared/types/index.ts`
- Create: `web/src/shared/lib/utils.ts`
- Create: `web/src/shared/lib/promptSerialization.ts`
- Create: `web/src/shared/lib/nodeConfigs.ts`

**Step 1: Copy API client to shared/api/client.ts**

Copy content of `web/src/lib/api/client.ts` → `web/src/shared/api/client.ts` unchanged.

**Step 2: Create shared/api/index.ts — re-export everything**

```typescript
// web/src/shared/api/index.ts
export { ApiError, apiFetch, API_BASE } from './client'
```

**Step 3: Copy shared types to shared/types/index.ts**

Copy content of `web/src/lib/api/types.ts` → `web/src/shared/types/index.ts` unchanged.
All `RunEvent` subtypes, `ModelInfo`, `ToolInfo`, `Pipeline`, `PipelineRun`, etc. live here.

**Step 4: Copy utilities**

- `web/src/lib/utils.ts` → `web/src/shared/lib/utils.ts` (no changes)
- `web/src/lib/promptSerialization.ts` → `web/src/shared/lib/promptSerialization.ts` (no changes)
- `web/src/lib/nodeConfigs.ts` → `web/src/shared/lib/nodeConfigs.ts` (no changes)

**Step 5: Type-check**

```bash
cd web && npx tsc -b --noEmit
```
Expected: passes (new files not yet imported).

**Step 6: Commit**

```bash
git add web/src/shared
git commit -m "feat(frontend/shared): add shared api client, types, utilities"
```

---

## Task 3: entities/node — Node Registry (Factory + Strategy pattern)

Replace `NODE_TYPES` dict + `nodeEditors` switch with an extensible registry.

**Files:**
- Create: `web/src/entities/node/model/registry.ts`
- Create: `web/src/entities/node/types.ts`
- Create: `web/src/entities/node/index.ts`

**Step 1: Create node types**

```typescript
// web/src/entities/node/types.ts
import type { ComponentType } from 'react'

export type NodeType = 'input' | 'agent' | 'output' | 'asset'

export type NodeEditorProps<C = Record<string, unknown>> = {
  nodeId: string
  config: C
  setConfig: (key: string, value: unknown) => void
}

export type NodeTypeDefinition = {
  type: NodeType
  label: string
  description: string
  icon: ComponentType<{ className?: string }>
  // Styling
  border: string
  borderSelected: string
  headerBg: string
  accent: string
  glow: string
  paletteBg: string
  cssVar: string
  // Editor component (registered separately to avoid circular deps)
  EditorComponent?: ComponentType<NodeEditorProps>
  defaultConfig?: Record<string, unknown>
}
```

**Step 2: Create registry**

```typescript
// web/src/entities/node/model/registry.ts
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
```

**Step 3: Create public API**

```typescript
// web/src/entities/node/index.ts
export { registerNode, registerNodeEditor, getNodeDefinition, getAllNodeDefinitions } from './model/registry'
export type { NodeType, NodeTypeDefinition, NodeEditorProps } from './types'
```

**Step 4: Type-check**

```bash
cd web && npx tsc -b --noEmit
```

**Step 5: Commit**

```bash
git add web/src/entities/node
git commit -m "feat(entities/node): add node type registry (Factory+Strategy pattern)"
```

---

## Task 4: entities/workflow — Purified store + API + serializer

Remove cross-store coupling from workflowStore. All calls to `useUIStore` are deleted; orchestration moves to features.

**Files:**
- Create: `web/src/entities/workflow/model/store.ts`
- Create: `web/src/entities/workflow/api/index.ts`
- Create: `web/src/entities/workflow/lib/serializer.ts`
- Create: `web/src/entities/workflow/lib/layout.ts`
- Create: `web/src/entities/workflow/types.ts`
- Create: `web/src/entities/workflow/index.ts`

**Step 1: Create workflow types**

```typescript
// web/src/entities/workflow/types.ts
export type NodeData = {
  label: string
  nodeType: 'input' | 'agent' | 'output' | 'asset' | 'group'
  description: string
  config: Record<string, unknown>
}

export type WorkflowDefinition = {
  name: string
  version: number
  nodes: { id: string; type: string; config: Record<string, unknown>; group?: string }[]
  edges: { from: string; to: string; condition?: string; trigger_rule?: string; loop?: boolean }[]
  groups?: { id: string; label: string; color: string }[]
  thumbnail_svg?: string
}
```

**Step 2: Create purified store**

Key change: remove ALL `useUIStore` imports. Remove `deleteFile` call.
The `onNodesChange` no longer calls `useUIStore.getState().selectNode` — that's `features/manage-canvas` responsibility.

```typescript
// web/src/entities/workflow/model/store.ts
import { create } from 'zustand'
import {
  type Node, type Edge, addEdge,
  applyNodeChanges, applyEdgeChanges,
  type OnNodesChange, type OnEdgesChange, type OnConnect,
} from '@xyflow/react'
import { getLayoutedElements } from '../lib/layout'
import { getNodeDefinition } from '@/entities/node'
import type { NodeData } from '../types'

// Node types whose prompt field receives auto-inserted template references on connect
const AUTO_PROMPT_TYPES = new Set(['agent', 'output'])

let nodeId = 0
const getId = () => `node_${++nodeId}`

type WorkflowState = {
  nodes: Node<NodeData>[]
  edges: Edge[]
  workflowName: string
  originalName: string

  onNodesChange: OnNodesChange<Node<NodeData>>
  onEdgesChange: OnEdgesChange
  onConnect: OnConnect

  addNode: (type: NodeData['nodeType'], position: { x: number; y: number }, initialConfig?: Record<string, unknown>) => void
  removeNode: (id: string) => void
  updateNodeConfig: (nodeId: string, config: Record<string, unknown>) => void
  updateNodeLabel: (nodeId: string, label: string) => void
  updateNodeDescription: (nodeId: string, description: string) => void
  applyAutoLayout: () => void
  setWorkflowName: (name: string) => void
  setOriginalName: (name: string) => void

  createGroup: (nodeIds: string[]) => void
  removeGroup: (groupId: string) => void
  updateGroupLabel: (groupId: string, label: string) => void
  updateGroupColor: (groupId: string, color: string) => void

  // Accessor used by features
  getNode: (id: string) => Node<NodeData> | undefined
}

export const useWorkflowStore = create<WorkflowState>((set, get) => ({
  nodes: [],
  edges: [],
  workflowName: '',
  originalName: '',

  getNode: (id) => get().nodes.find(n => n.id === id),

  // NOTE: side-effects (file deletion, UI deselection) are handled by features/manage-canvas
  onNodesChange: (changes) => {
    set({ nodes: applyNodeChanges(changes, get().nodes) })
  },
  onEdgesChange: (changes) => {
    const removals = changes.filter((c) => c.type === 'remove')
    if (removals.length > 0) {
      const { edges, nodes } = get()
      const removedEdges = removals
        .map((c) => edges.find((e) => e.id === c.id))
        .filter(Boolean) as Edge[]
      let updatedNodes = nodes
      for (const edge of removedEdges) {
        const target = updatedNodes.find((n) => n.id === edge.target)
        if (!target || !AUTO_PROMPT_TYPES.has(target.data.nodeType)) continue
        const prompt = (target.data.config.prompt as string) ?? ''
        if (!prompt) continue
        const ref = `{{${edge.source}}}`
        if (!prompt.includes(ref)) continue
        const newPrompt = prompt.split('\n').filter((line) => line.trim() !== ref).join('\n')
        updatedNodes = updatedNodes.map((n) =>
          n.id === edge.target
            ? { ...n, data: { ...n.data, config: { ...n.data.config, prompt: newPrompt } } }
            : n,
        )
      }
      if (updatedNodes !== nodes) set({ nodes: updatedNodes })
    }
    set({ edges: applyEdgeChanges(changes, get().edges) })
  },
  onConnect: (connection) => {
    set({ edges: addEdge({ ...connection, type: 'default' }, get().edges) })
    if (!connection.source || !connection.target) return
    const targetNode = get().nodes.find((n) => n.id === connection.target)
    if (!targetNode || !AUTO_PROMPT_TYPES.has(targetNode.data.nodeType)) return
    const currentPrompt = (targetNode.data.config.prompt as string) ?? ''
    const ref = `{{${connection.source}}}`
    if (currentPrompt.includes(ref)) return
    const newPrompt = currentPrompt ? `${currentPrompt}\n${ref}` : ref
    set({
      nodes: get().nodes.map((n) =>
        n.id === connection.target
          ? { ...n, data: { ...n.data, config: { ...n.data.config, prompt: newPrompt } } }
          : n,
      ),
    })
  },

  addNode: (type, position, initialConfig) => {
    const id = getId()
    const def = getNodeDefinition(type as import('@/entities/node').NodeType)
    const newNode: Node<NodeData> = {
      id,
      type: 'upalNode',
      position,
      data: {
        label: def.label,
        nodeType: type,
        description: def.description,
        config: initialConfig ?? {},
      },
    }
    set({ nodes: [...get().nodes, newNode] })
    return id
  },

  removeNode: (id) => {
    set({ nodes: get().nodes.filter(n => n.id !== id) })
  },

  updateNodeConfig: (nodeId, config) => {
    set({
      nodes: get().nodes.map((n) =>
        n.id === nodeId
          ? { ...n, data: { ...n.data, config: { ...n.data.config, ...config } } }
          : n,
      ),
    })
  },
  updateNodeLabel: (nodeId, label) => {
    set({ nodes: get().nodes.map((n) => n.id === nodeId ? { ...n, data: { ...n.data, label } } : n) })
  },
  updateNodeDescription: (nodeId, description) => {
    set({ nodes: get().nodes.map((n) => n.id === nodeId ? { ...n, data: { ...n.data, description } } : n) })
  },
  applyAutoLayout: () => {
    const { nodes, edges } = get()
    if (nodes.length === 0) return
    const { nodes: layouted } = getLayoutedElements<Node<NodeData>>(nodes, edges, 'LR')
    set({ nodes: layouted })
  },
  setWorkflowName: (name) => set({ workflowName: name }),
  setOriginalName: (name) => set({ originalName: name }),

  createGroup: (nodeIds) => {
    if (nodeIds.length === 0) return
    const groupId = getId()
    const { nodes } = get()
    const selectedNodes = nodes.filter((n) => nodeIds.includes(n.id))
    if (selectedNodes.length === 0) return
    const padding = 40
    const xs = selectedNodes.map((n) => n.position.x)
    const ys = selectedNodes.map((n) => n.position.y)
    const minX = Math.min(...xs) - padding
    const minY = Math.min(...ys) - padding
    const maxX = Math.max(...xs) + 320 + padding
    const maxY = Math.max(...ys) + 100 + padding
    const groupNode: Node<NodeData> = {
      id: groupId,
      type: 'groupNode',
      position: { x: minX, y: minY },
      style: { width: maxX - minX, height: maxY - minY },
      data: { label: 'Group', nodeType: 'group', description: '', config: { color: 'purple' } },
    }
    const updatedNodes = nodes.map((n) => {
      if (nodeIds.includes(n.id)) {
        return { ...n, parentId: groupId, position: { x: n.position.x - minX, y: n.position.y - minY }, extent: 'parent' as const }
      }
      return n
    })
    set({ nodes: [groupNode, ...updatedNodes] })
    return groupId
  },

  removeGroup: (groupId) => {
    const { nodes } = get()
    const groupNode = nodes.find((n) => n.id === groupId)
    if (!groupNode || groupNode.type !== 'groupNode') return
    const updatedNodes = nodes.filter((n) => n.id !== groupId).map((n) => {
      if (n.parentId === groupId) {
        return {
          ...n, parentId: undefined, extent: undefined,
          position: { x: n.position.x + groupNode.position.x, y: n.position.y + groupNode.position.y },
        }
      }
      return n
    })
    set({ nodes: updatedNodes })
  },

  updateGroupLabel: (groupId, label) => {
    set({ nodes: get().nodes.map((n) => n.id === groupId ? { ...n, data: { ...n.data, label } } : n) })
  },
  updateGroupColor: (groupId, color) => {
    set({ nodes: get().nodes.map((n) => n.id === groupId ? { ...n, data: { ...n.data, config: { ...n.data.config, color } } } : n) })
  },
}))
```

**Step 3: Move serializer and layout**

- Copy `web/src/lib/serializer.ts` → `web/src/entities/workflow/lib/serializer.ts`
  - Update import: `@/lib/nodeTypes` → `@/entities/node` (use `getNodeDefinition`)
- Copy `web/src/lib/layout.ts` → `web/src/entities/workflow/lib/layout.ts` (no import changes)

**Step 4: Move workflow API**

Copy `web/src/lib/api/workflows.ts` → `web/src/entities/workflow/api/index.ts`
Update imports: `./client` → `@/shared/api/client`, `./types` → `@/shared/types`

**Step 5: Create public API**

```typescript
// web/src/entities/workflow/index.ts
export { useWorkflowStore } from './model/store'
export { serializeWorkflow, deserializeWorkflow } from './lib/serializer'
export { getLayoutedElements } from './lib/layout'
export * from './api/index'
export type { NodeData, WorkflowDefinition } from './types'
```

**Step 6: Type-check**

```bash
cd web && npx tsc -b --noEmit
```

**Step 7: Commit**

```bash
git add web/src/entities/workflow
git commit -m "feat(entities/workflow): add purified store, serializer, workflow API"
```

---

## Task 5: entities/run — Purified store + run API

**Files:**
- Create: `web/src/entities/run/model/store.ts`
- Create: `web/src/entities/run/api/index.ts`
- Create: `web/src/entities/run/index.ts`

**Step 1: Copy executionStore to entities/run/model/store.ts**

Copy `web/src/stores/executionStore.ts` → `web/src/entities/run/model/store.ts`
- Rename export: `useExecutionStore` stays as-is
- Update import: `@/lib/api` → `@/shared/types`
- Remove: `export type { RunEvent }` (already in shared/types)

**Step 2: Move run API**

Copy `web/src/lib/api/runs.ts` → `web/src/entities/run/api/index.ts`
Update imports to use `@/shared/api/client` and `@/shared/types`.

**Step 3: Create public API**

```typescript
// web/src/entities/run/index.ts
export { useExecutionStore } from './model/store'
export type { NodeRunStatus } from './model/store'
export * from './api/index'
```

**Step 4: Type-check + commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/entities/run
git commit -m "feat(entities/run): add purified execution store and run API"
```

---

## Task 6: entities/ui + entities/pipeline

**Files:**
- Create: `web/src/entities/ui/model/store.ts`
- Create: `web/src/entities/ui/index.ts`
- Create: `web/src/entities/pipeline/api/index.ts`
- Create: `web/src/entities/pipeline/index.ts`

**Step 1: Copy uiStore**

Copy `web/src/stores/uiStore.ts` → `web/src/entities/ui/model/store.ts` (no import changes needed).

```typescript
// web/src/entities/ui/index.ts
export { useUIStore } from './model/store'
export type { Toast } from './model/store'
```

**Step 2: Move pipeline API**

Copy `web/src/lib/api/pipelines.ts` → `web/src/entities/pipeline/api/index.ts`
Update imports to use `@/shared/api/client` and `@/shared/types`.

```typescript
// web/src/entities/pipeline/index.ts
export * from './api/index'
```

**Step 3: Type-check + commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/entities/ui web/src/entities/pipeline
git commit -m "feat(entities): add ui store and pipeline API"
```

---

## Task 7: features/manage-canvas — Cross-store orchestration

This is the key fix for cross-store coupling. `useCanvasActions` owns all operations that touch multiple stores.

**Files:**
- Create: `web/src/features/manage-canvas/model/useCanvasActions.ts`
- Create: `web/src/features/manage-canvas/index.ts`

**Step 1: Create useCanvasActions**

```typescript
// web/src/features/manage-canvas/model/useCanvasActions.ts
import { useCallback } from 'react'
import { useWorkflowStore } from '@/entities/workflow'
import { useUIStore } from '@/entities/ui'
import { deleteFile } from '@/shared/api'
import type { NodeData } from '@/entities/workflow'
import type { Node } from '@xyflow/react'

/**
 * Orchestrates canvas mutations that touch multiple stores.
 * Components call these instead of calling stores directly.
 */
export function useCanvasActions() {
  const workflowStore = useWorkflowStore()
  const selectNode = useUIStore((s) => s.selectNode)

  /**
   * Delete a node + its backing file (if asset) + deselect.
   * Cross-store: workflow store removes node, ui store clears selection.
   */
  const deleteNode = useCallback((id: string) => {
    const node = useWorkflowStore.getState().getNode(id) as Node<NodeData> | undefined
    if (node?.data.nodeType === 'asset') {
      const fileId = node.data.config.file_id as string | undefined
      if (fileId) {
        deleteFile(fileId).catch((err) =>
          console.error(`Failed to delete file ${fileId}:`, err),
        )
      }
    }
    useWorkflowStore.getState().removeNode(id)
    // Deselect if this was the selected node
    if (useUIStore.getState().selectedNodeId === id) {
      selectNode(null)
    }
  }, [selectNode])

  /**
   * Create a group from selected node IDs, then select the group node.
   */
  const createGroup = useCallback((nodeIds: string[]) => {
    const groupId = useWorkflowStore.getState().createGroup(nodeIds)
    if (groupId) selectNode(groupId)
  }, [selectNode])

  /**
   * Remove a group node and release its children, then deselect.
   */
  const removeGroup = useCallback((groupId: string) => {
    useWorkflowStore.getState().removeGroup(groupId)
    selectNode(null)
  }, [selectNode])

  return {
    deleteNode,
    createGroup,
    removeGroup,
    // Convenience pass-throughs to workflow store (no cross-store concern)
    addNode: workflowStore.addNode,
    updateNodeConfig: workflowStore.updateNodeConfig,
    updateNodeLabel: workflowStore.updateNodeLabel,
    applyAutoLayout: workflowStore.applyAutoLayout,
    updateGroupLabel: workflowStore.updateGroupLabel,
    updateGroupColor: workflowStore.updateGroupColor,
  }
}
```

**Step 2: Create public API**

```typescript
// web/src/features/manage-canvas/index.ts
export { useCanvasActions } from './model/useCanvasActions'
```

**Step 3: Type-check + commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/features/manage-canvas
git commit -m "feat(features/manage-canvas): add useCanvasActions (cross-store orchestration)"
```

---

## Task 8: features/execute-workflow — Event bus + useExecuteRun

The key new pattern: a typed event bus replaces direct store calls in the SSE dispatcher.

**Files:**
- Create: `web/src/features/execute-workflow/model/eventBus.ts`
- Create: `web/src/features/execute-workflow/model/useExecuteRun.ts`
- Create: `web/src/features/execute-workflow/model/useReconnectRun.ts`
- Create: `web/src/features/execute-workflow/index.ts`

**Step 1: Create typed event bus utility in shared**

```typescript
// web/src/shared/lib/eventBus.ts
type EventHandler<T> = (data: T) => void

export function createEventBus<TMap extends Record<string, unknown>>() {
  const listeners = new Map<keyof TMap, Set<EventHandler<unknown>>>()

  return {
    on<K extends keyof TMap>(event: K, handler: EventHandler<TMap[K]>) {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(handler as EventHandler<unknown>)
      // Return off function for easy cleanup
      return () => listeners.get(event)?.delete(handler as EventHandler<unknown>)
    },
    emit<K extends keyof TMap>(event: K, data: TMap[K]) {
      listeners.get(event)?.forEach((h) => h(data))
    },
    clear() {
      listeners.clear()
    },
  }
}
```

**Step 2: Create run event bus**

```typescript
// web/src/features/execute-workflow/model/eventBus.ts
import { createEventBus } from '@/shared/lib/eventBus'
import type { NodeRunStatus } from '@/entities/run'

export type RunEventBusMap = {
  node_started:   { nodeId: string; startedAt?: number }
  node_completed: { nodeId: string }
  node_error:     { nodeId: string }
  node_skipped:   { nodeId: string }
  node_waiting:   { nodeId: string }
  node_resumed:   { nodeId: string }
  run_done:       { sessionState: Record<string, unknown>; status: string; sessionId: string }
  run_error:      { message: string }
}

export const runEventBus = createEventBus<RunEventBusMap>()

// Wire bus events → executionStore (one-time setup, runs at module load)
import { useExecutionStore } from '@/entities/run'

runEventBus.on('node_started',   ({ nodeId, startedAt }) => useExecutionStore.getState().setNodeStatus(nodeId, 'running', startedAt))
runEventBus.on('node_completed', ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'completed'))
runEventBus.on('node_error',     ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'error'))
runEventBus.on('node_skipped',   ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'skipped'))
runEventBus.on('node_waiting',   ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'waiting'))
runEventBus.on('node_resumed',   ({ nodeId }) => useExecutionStore.getState().setNodeStatus(nodeId, 'running'))
runEventBus.on('run_done', ({ sessionState }) => useExecutionStore.getState().setSessionState(sessionState))
```

**Step 3: Create useExecuteRun (refactored)**

```typescript
// web/src/features/execute-workflow/model/useExecuteRun.ts
import { useCallback, useRef } from 'react'
import { useWorkflowStore, serializeWorkflow } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { startRun, connectToRunEvents } from '@/entities/run'
import type { RunEvent } from '@/shared/types'
import { runEventBus } from './eventBus'

// sessionStorage helpers
export function setActiveRun(runId: string, workflowName: string) {
  sessionStorage.setItem('active_run_id', runId)
  sessionStorage.setItem('active_run_workflow', workflowName)
}
export function getActiveRun(): { runId: string; workflowName: string } | null {
  const runId = sessionStorage.getItem('active_run_id')
  const workflowName = sessionStorage.getItem('active_run_workflow')
  if (!runId) return null
  return { runId, workflowName: workflowName || '' }
}
export function clearActiveRun() {
  sessionStorage.removeItem('active_run_id')
  sessionStorage.removeItem('active_run_workflow')
}

/** Translate a raw RunEvent from SSE into runEventBus emissions + raw event storage */
export function dispatchRunEvent(event: RunEvent, addRunEvent: (e: RunEvent) => void) {
  addRunEvent(event)
  switch (event.type) {
    case 'node_started':   runEventBus.emit('node_started',   { nodeId: event.nodeId, startedAt: event.startedAt }); break
    case 'node_completed': runEventBus.emit('node_completed', { nodeId: event.nodeId }); break
    case 'node_skipped':   runEventBus.emit('node_skipped',   { nodeId: event.nodeId }); break
    case 'node_waiting':   runEventBus.emit('node_waiting',   { nodeId: event.nodeId }); break
    case 'node_resumed':   runEventBus.emit('node_resumed',   { nodeId: event.nodeId }); break
  }
}

export function useExecuteRun() {
  const workflowName = useWorkflowStore((s) => s.workflowName)
  const isRunning = useExecutionStore((s) => s.isRunning)
  const setIsRunning = useExecutionStore((s) => s.setIsRunning)
  const addRunEvent = useExecutionStore((s) => s.addRunEvent)
  const clearRunEvents = useExecutionStore((s) => s.clearRunEvents)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const setSessionState = useExecutionStore((s) => s.setSessionState)
  const abortRef = useRef<AbortController | null>(null)

  const executeRun = useCallback(async (inputs: Record<string, string>) => {
    const name = workflowName
    if (!name) return
    abortRef.current?.abort()
    const { nodes, edges } = useWorkflowStore.getState()
    const workflow = serializeWorkflow(name, nodes, edges)
    clearRunEvents()
    clearNodeStatuses()
    setIsRunning(true)
    addRunEvent({ type: 'info', message: `Running workflow "${name}"...` })
    try {
      const { run_id } = await startRun(name, inputs, workflow)
      setActiveRun(run_id, name)
      const abort = new AbortController()
      abortRef.current = abort
      await connectToRunEvents(
        run_id,
        (event) => dispatchRunEvent(event, addRunEvent),
        (result) => {
          addRunEvent({ type: 'done', status: result.status as string, sessionId: result.session_id as string, state: (result.state ?? {}) as Record<string, unknown> })
          const statuses = useExecutionStore.getState().nodeStatuses
          for (const [id, status] of Object.entries(statuses)) {
            if (status === 'running') runEventBus.emit('node_completed', { nodeId: id })
          }
          if (result.status === 'failed') {
            const match = String(result.error || '').match(/node "([^"]+)"/)
            if (match) runEventBus.emit('node_error', { nodeId: match[1] })
          }
          if (result.state && typeof result.state === 'object') {
            runEventBus.emit('run_done', { sessionState: result.state as Record<string, unknown>, status: result.status as string, sessionId: result.session_id as string })
          }
          clearActiveRun()
          setIsRunning(false)
        },
        (error) => {
          addRunEvent({ type: 'error', message: error.message })
          clearActiveRun()
          setIsRunning(false)
        },
        { signal: abort.signal },
      )
    } catch (err) {
      addRunEvent({ type: 'error', message: err instanceof Error ? err.message : String(err) })
      clearActiveRun()
      setIsRunning(false)
    }
  }, [workflowName, clearRunEvents, clearNodeStatuses, setIsRunning, addRunEvent, setSessionState])

  return { executeRun, isRunning }
}
```

**Step 4: Move useReconnectRun**

Copy `web/src/hooks/useReconnectRun.ts` → `web/src/features/execute-workflow/model/useReconnectRun.ts`
Update imports:
- `@/stores/workflowStore` → `@/entities/workflow`
- `@/stores/executionStore` → `@/entities/run`
- `@/lib/api` → `@/entities/run` (for `connectToRunEvents`)
- `./useExecuteRun` references → use `dispatchRunEvent`, `getActiveRun`, `clearActiveRun` from same `model/` folder

**Step 5: Create public API**

```typescript
// web/src/features/execute-workflow/index.ts
export { useExecuteRun, getActiveRun, setActiveRun, clearActiveRun, dispatchRunEvent } from './model/useExecuteRun'
export { runEventBus } from './model/eventBus'
```

**Step 6: Type-check + commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/features/execute-workflow web/src/shared/lib/eventBus.ts
git commit -m "feat(features/execute-workflow): add typed event bus and refactored useExecuteRun"
```

---

## Task 9: features/edit-node — NodeEditor with registry

Replace static `nodeEditors` dict with `getNodeDefinition` registry lookup.

**Files:**
- Create: `web/src/features/edit-node/ui/NodeEditor.tsx`
- Create: `web/src/features/edit-node/index.ts`
- Move existing node editor components to `features/edit-node/ui/`

**Step 1: Move node editor components**

Move (rename path only, no content changes):
- `web/src/components/editor/nodes/InputNodeEditor.tsx` → `web/src/features/edit-node/ui/InputNodeEditor.tsx`
- `web/src/components/editor/nodes/AgentNodeEditor.tsx` → `web/src/features/edit-node/ui/AgentNodeEditor.tsx`
- `web/src/components/editor/nodes/OutputNodeEditor.tsx` → `web/src/features/edit-node/ui/OutputNodeEditor.tsx`
- `web/src/components/editor/nodes/AssetNodeEditor.tsx` → `web/src/features/edit-node/ui/AssetNodeEditor.tsx`
- `web/src/components/panel/AIChatEditor.tsx` → `web/src/features/edit-node/ui/AIChatEditor.tsx`

Update imports in each moved file:
- `@/stores/workflowStore` → `@/entities/workflow`
- `@/lib/api` → `@/entities/run` / `@/shared/types` / `@/shared/api`
- `@/lib/nodeTypes` → `@/entities/node`

**Step 2: Register EditorComponents in entities/node**

After moving editors, register them in `entities/node/model/registry.ts`. Add at the end of the file:

```typescript
// web/src/entities/node/model/registry.ts — add import and call at bottom
// NOTE: lazy import to break circular dep (entities -> features is forbidden, so we use a registration hook)
// Instead: features/edit-node/model/registerEditors.ts does the registration
```

Create registration file in features:

```typescript
// web/src/features/edit-node/model/registerEditors.ts
import { registerNodeEditor } from '@/entities/node'
import { InputNodeEditor } from '../ui/InputNodeEditor'
import { AgentNodeEditor } from '../ui/AgentNodeEditor'
import { OutputNodeEditor } from '../ui/OutputNodeEditor'
import { AssetNodeEditor } from '../ui/AssetNodeEditor'

// Called once at app startup (in app/providers.tsx)
export function registerAllEditors() {
  registerNodeEditor('input', InputNodeEditor)
  registerNodeEditor('agent', AgentNodeEditor)
  registerNodeEditor('output', OutputNodeEditor)
  registerNodeEditor('asset', AssetNodeEditor)
}
```

**Step 3: Create registry-based NodeEditor**

```typescript
// web/src/features/edit-node/ui/NodeEditor.tsx
import { getNodeDefinition } from '@/entities/node'
import { useWorkflowStore } from '@/entities/workflow'
import type { NodeData } from '@/entities/workflow'
import { Button } from '@/shared/ui/button'
import { X } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

export const fieldBox = "text-xs overflow-y-auto cursor-default select-text rounded-md border border-input bg-transparent px-3 py-2 whitespace-pre-wrap break-words max-h-32"
export const fieldBoxExpand = "text-xs overflow-y-auto cursor-default select-text rounded-md border border-input bg-transparent px-3 py-2 whitespace-pre-wrap break-words flex-1 min-h-24"

type NodeEditorProps = {
  nodeId: string
  data: NodeData
  onClose: () => void
  embedded?: boolean
}

export function NodeEditor({ nodeId, data, onClose, embedded }: NodeEditorProps) {
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig)
  const updateNodeLabel = useWorkflowStore((s) => s.updateNodeLabel)

  const setConfig = (key: string, value: unknown) => updateNodeConfig(nodeId, { [key]: value })

  // Registry lookup — no switch statement
  const def = getNodeDefinition(data.nodeType as import('@/entities/node').NodeType)
  const Icon = def.icon
  const EditorComponent = def.EditorComponent

  const header = (
    <div className="flex items-center gap-2 px-3 py-2.5 border-b border-border">
      {Icon && (
        <div className={cn('h-6 w-6 rounded-md flex items-center justify-center shrink-0', def.accent)}>
          <Icon className="h-3.5 w-3.5" />
        </div>
      )}
      <input
        className="flex-1 min-w-0 text-sm font-semibold bg-transparent border-none outline-none focus:ring-1 focus:ring-ring rounded px-1"
        value={data.label}
        onChange={(e) => updateNodeLabel(nodeId, e.target.value)}
      />
      <Button variant="ghost" size="icon" className="h-6 w-6 shrink-0" onClick={onClose}>
        <X className="h-3.5 w-3.5" />
      </Button>
    </div>
  )

  if (embedded) {
    return (
      <div className="flex flex-col flex-1 min-h-0">
        {header}
        <div className="p-2 flex-1 flex flex-col gap-3 min-h-0 overflow-y-auto">
          {EditorComponent && <EditorComponent nodeId={nodeId} config={data.config} setConfig={setConfig} />}
        </div>
      </div>
    )
  }

  return (
    <aside className="w-80 border-l border-border bg-background p-0 overflow-y-auto">
      <div className="flex items-center justify-between p-4 border-b border-border">
        <div className="flex items-center gap-2">
          {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
          <h3 className="text-sm font-semibold">Properties</h3>
        </div>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="p-2 space-y-3">
        {EditorComponent && <EditorComponent nodeId={nodeId} config={data.config} setConfig={setConfig} />}
      </div>
    </aside>
  )
}
```

**Step 4: Create public API**

```typescript
// web/src/features/edit-node/index.ts
export { NodeEditor, fieldBox, fieldBoxExpand } from './ui/NodeEditor'
export { registerAllEditors } from './model/registerEditors'
```

**Step 5: Type-check + commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/features/edit-node
git commit -m "feat(features/edit-node): registry-based NodeEditor, move editor components"
```

---

## Task 10: features/generate-workflow + features/upload-asset

**Files:**
- Move: `web/src/components/dialogs/GenerateDialog.tsx` → `web/src/features/generate-workflow/ui/GenerateDialog.tsx`
- Move: `web/src/hooks/useUpstreamNodes.ts` → `web/src/features/edit-node/model/useUpstreamNodes.ts`
- Move: `web/src/hooks/useUploadAsset.ts` (if exists, else extract from Canvas) → `web/src/features/upload-asset/model/useUploadAsset.ts`
- Move: `web/src/lib/api/configure.ts` → `web/src/features/edit-node/api/index.ts`
- Move: `web/src/lib/api/models.ts` → `web/src/shared/api/models.ts`
- Move: `web/src/lib/api/tools.ts` → `web/src/shared/api/tools.ts`
- Move: `web/src/lib/api/upload.ts` → `web/src/features/upload-asset/api/index.ts`

For each moved file, update imports:
- `@/lib/api` → `@/shared/api` / `@/shared/types`
- `@/stores/workflowStore` → `@/entities/workflow`
- `@/stores/executionStore` → `@/entities/run`

**Step 1: Move and update each file**

Follow the mapping above. After each file move, verify with `npx tsc -b --noEmit`.

**Step 2: Create public APIs**

```typescript
// web/src/features/generate-workflow/index.ts
export { GenerateDialog } from './ui/GenerateDialog'

// web/src/features/upload-asset/index.ts
export { useUploadAsset } from './model/useUploadAsset'
```

**Step 3: Commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/features/generate-workflow web/src/features/upload-asset web/src/shared/api
git commit -m "feat(features): add generate-workflow, upload-asset; move API modules to shared"
```

---

## Task 11: features/manage-canvas — Keyboard shortcuts + auto-save

**Files:**
- Move: `web/src/hooks/useKeyboardShortcuts.ts` → `web/src/features/manage-canvas/model/useKeyboardShortcuts.ts`
- Move: `web/src/hooks/useAutoSave.ts` → `web/src/features/manage-canvas/model/useAutoSave.ts`

Update imports in moved files:
- `@/stores/workflowStore` → `@/entities/workflow`
- `@/stores/uiStore` → `@/entities/ui`
- `@/lib/api` → appropriate entity/feature API

Update `web/src/features/manage-canvas/index.ts`:

```typescript
// web/src/features/manage-canvas/index.ts
export { useCanvasActions } from './model/useCanvasActions'
export { useKeyboardShortcuts } from './model/useKeyboardShortcuts'
export { useAutoSave } from './model/useAutoSave'
```

**Commit:**

```bash
cd web && npx tsc -b --noEmit
git add web/src/features/manage-canvas
git commit -m "feat(features/manage-canvas): add keyboard shortcuts and auto-save hooks"
```

---

## Task 12: widgets/ — Move canvas, panels, palette

Move visual components to `widgets/`. These are structural moves — copy content, update imports.

**Mapping:**

| Current path | New path |
|---|---|
| `components/editor/Canvas.tsx` | `widgets/workflow-canvas/ui/Canvas.tsx` |
| `components/editor/nodes/UpalNode.tsx` | `widgets/workflow-canvas/ui/UpalNode.tsx` |
| `components/editor/nodes/GroupNode.tsx` | `widgets/workflow-canvas/ui/GroupNode.tsx` |
| `components/panel/RightPanel.tsx` | `widgets/right-panel/ui/RightPanel.tsx` |
| `components/sidebar/NodePalette.tsx` | `widgets/node-palette/ui/NodePalette.tsx` |
| `components/console/Console.tsx` | `widgets/bottom-console/ui/Console.tsx` |
| `components/runs/RunDetail.tsx` | `widgets/run-detail/ui/RunDetail.tsx` |
| `components/pipelines/PipelineEditor.tsx` | `widgets/pipeline-editor/ui/PipelineEditor.tsx` |

**For each file:**
1. Copy content to new path
2. Update all imports (`@/stores/...` → `@/entities/...`, `@/hooks/...` → `@/features/...`, etc.)
3. Run `npx tsc -b --noEmit` after each

**Canvas.tsx key import updates:**
```typescript
// Before
import { useWorkflowStore } from '@/stores/workflowStore'
import { useUIStore } from '@/stores/uiStore'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useCanvasActions } from '@/hooks/...'  // doesn't exist yet
// After
import { useWorkflowStore } from '@/entities/workflow'
import { useUIStore } from '@/entities/ui'
import { useKeyboardShortcuts, useCanvasActions } from '@/features/manage-canvas'
```

**Create public APIs:**

```typescript
// web/src/widgets/workflow-canvas/index.ts
export { Canvas } from './ui/Canvas'
export { UpalNode } from './ui/UpalNode'

// web/src/widgets/right-panel/index.ts
export { RightPanel } from './ui/RightPanel'

// (etc. for each widget)
```

**Commit after all widgets:**

```bash
cd web && npx tsc -b --noEmit
git add web/src/widgets
git commit -m "feat(widgets): move canvas, right-panel, palette, console to widgets/"
```

---

## Task 13: pages/ — Slim page shells

Move and update page components.

**Mapping:**

| Current path | New path |
|---|---|
| `pages/Editor.tsx` | `pages/editor/index.tsx` |
| `pages/Landing.tsx` | `pages/landing/index.tsx` |
| `pages/Runs.tsx` | `pages/runs/index.tsx` |
| `pages/Pipelines.tsx` | `pages/pipelines/index.tsx` |
| `pages/Connections.tsx` | `pages/connections/index.tsx` |
| `pages/ProductLanding.tsx` | `pages/landing/ProductLanding.tsx` |

Pages should only import from `widgets/`, `features/`, `entities/`, `shared/`. No direct API calls in pages.

**Example Editor page (thin shell):**

```typescript
// web/src/pages/editor/index.tsx
import { Canvas } from '@/widgets/workflow-canvas'
import { RightPanel } from '@/widgets/right-panel'
import { NodePalette } from '@/widgets/node-palette'
import { BottomConsole } from '@/widgets/bottom-console'
import { WorkflowHeader } from '@/widgets/workflow-header'
import { useReconnectRun } from '@/features/execute-workflow'

export default function EditorPage() {
  useReconnectRun()
  return (
    <div className="h-screen flex flex-col">
      <WorkflowHeader />
      <div className="flex flex-1 min-h-0">
        <NodePalette />
        <Canvas />
        <RightPanel />
      </div>
      <BottomConsole />
    </div>
  )
}
```

**Commit:**

```bash
cd web && npx tsc -b --noEmit
git add web/src/pages
git commit -m "feat(pages): move pages to FSD structure, slim to thin shells"
```

---

## Task 14: app/ — Router + providers

**Files:**
- Create: `web/src/app/router.tsx`
- Create: `web/src/app/providers.tsx`
- Update: `web/src/main.tsx`

**Step 1: Create providers (register editors at startup)**

```typescript
// web/src/app/providers.tsx
import { registerAllEditors } from '@/features/edit-node'

// Call once at module load
registerAllEditors()

export function AppProviders({ children }: { children: React.ReactNode }) {
  return <>{children}</>
}
```

**Step 2: Move router from App.tsx**

```typescript
// web/src/app/router.tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import EditorPage from '@/pages/editor'
import LandingPage from '@/pages/landing'
import RunsPage from '@/pages/runs'
import PipelinesPage from '@/pages/pipelines'
import ConnectionsPage from '@/pages/connections'
import RunDetail from '@/widgets/run-detail'
import ProductLanding from '@/pages/landing/ProductLanding'

export function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<ProductLanding />} />
        <Route path="/workflows" element={<LandingPage />} />
        <Route path="/editor" element={<EditorPage />} />
        <Route path="/runs" element={<RunsPage />} />
        <Route path="/runs/:id" element={<RunDetail />} />
        <Route path="/pipelines" element={<PipelinesPage />} />
        <Route path="/pipelines/:id" element={<PipelinesPage />} />
        <Route path="/connections" element={<ConnectionsPage />} />
      </Routes>
    </BrowserRouter>
  )
}
```

**Step 3: Update main.tsx and App.tsx**

```typescript
// web/src/main.tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import { AppProviders } from './app/providers'
import { AppRouter } from './app/router'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppProviders>
      <AppRouter />
    </AppProviders>
  </StrictMode>,
)
```

**Step 4: Commit**

```bash
cd web && npx tsc -b --noEmit
git add web/src/app web/src/main.tsx
git commit -m "feat(app): consolidate router and providers into app/ layer"
```

---

## Task 15: ESLint boundaries rule

Enforce import direction at lint time.

**Files:**
- Modify: `web/eslint.config.js`

**Step 1: Add boundaries plugin config**

```javascript
// web/eslint.config.js — add to existing config
import boundaries from 'eslint-plugin-boundaries'

// Add to plugins section:
'boundaries': boundaries,

// Add to rules section:
'boundaries/element-types': ['error', {
  default: 'disallow',
  rules: [
    { from: 'app',      allow: ['pages', 'widgets', 'features', 'entities', 'shared', 'unknown'] },
    { from: 'pages',    allow: ['widgets', 'features', 'entities', 'shared', 'unknown'] },
    { from: 'widgets',  allow: ['features', 'entities', 'shared', 'unknown'] },
    { from: 'features', allow: ['entities', 'shared', 'unknown'] },
    { from: 'entities', allow: ['shared', 'unknown'] },
    { from: 'shared',   allow: ['unknown'] },
  ],
}],
```

Add `boundaries/element-types` settings:

```javascript
settings: {
  'boundaries/elements': [
    { type: 'app',      pattern: 'src/app/**' },
    { type: 'pages',    pattern: 'src/pages/**' },
    { type: 'widgets',  pattern: 'src/widgets/**' },
    { type: 'features', pattern: 'src/features/**' },
    { type: 'entities', pattern: 'src/entities/**' },
    { type: 'shared',   pattern: 'src/shared/**' },
  ],
},
```

**Step 2: Run lint to verify rules work**

```bash
cd web && npm run lint
```
Expected: no boundary errors (if all imports are correct), or specific violations to fix.

**Step 3: Fix any boundary violations**

For each lint error, move the import to the correct layer.

**Step 4: Commit**

```bash
git add web/eslint.config.js
git commit -m "chore(frontend): add ESLint boundaries to enforce FSD import rules"
```

---

## Task 16: Remove old directories + backward compat shims

**Step 1: Update old store barrel exports**

The `workflowStore.ts` re-exports were used for backward compatibility:
```typescript
// These re-exports from workflowStore.ts must be removed:
export type { RunEvent, NodeRunStatus } from './executionStore'
export { useExecutionStore } from './executionStore'
export { useUIStore } from './uiStore'
```

Search for any remaining imports of old paths and update them:
```bash
cd web && grep -r "@/stores/" src/ --include="*.ts" --include="*.tsx" -l
cd web && grep -r "@/hooks/" src/ --include="*.ts" --include="*.tsx" -l
cd web && grep -r "@/lib/api" src/ --include="*.ts" --include="*.tsx" -l
cd web && grep -r "@/lib/nodeTypes" src/ --include="*.ts" --include="*.tsx" -l
cd web && grep -r "@/lib/serializer" src/ --include="*.ts" --include="*.tsx" -l
```

Fix each found file to use the new FSD path.

**Step 2: Delete old directories once all imports updated**

```bash
# Only run after grep above shows 0 remaining imports
rm -rf web/src/stores
rm -rf web/src/hooks
rm -rf web/src/lib
rm -rf web/src/components/editor/nodes  # individual node editors moved
# Keep web/src/components/ if any components not yet moved (ErrorBoundary, ToastContainer, ui/)
```

**Step 3: Move remaining components**

- `web/src/components/ErrorBoundary.tsx` → `web/src/shared/ui/ErrorBoundary.tsx`
- `web/src/components/ToastContainer.tsx` → `web/src/shared/ui/ToastContainer.tsx`
- `web/src/components/ui/` → `web/src/shared/ui/` (Shadcn primitives)

**Step 4: Full build verification**

```bash
cd web && npm run build
```
Expected: Build succeeds with no errors.

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor(frontend): complete FSD migration, remove old directories"
```

---

## Task 17: Final verification

**Step 1: TypeScript strict check**

```bash
cd web && npx tsc -b --noEmit
```
Expected: 0 errors.

**Step 2: Lint check**

```bash
cd web && npm run lint
```
Expected: 0 boundary violations, 0 errors.

**Step 3: Production build**

```bash
cd web && npm run build
```
Expected: Build succeeds, outputs to `dist/`.

**Step 4: Smoke test (manual)**

Run `make dev-frontend` and verify:
- [ ] Workflow list loads at `/workflows`
- [ ] Editor opens at `/editor`
- [ ] Nodes can be added, connected, configured
- [ ] Run executes and SSE events display in console
- [ ] Node status rings update during execution

**Step 5: Final commit + tag**

```bash
git add -A
git commit -m "feat(frontend): complete FSD architecture refactoring

- Feature-Sliced Design with strict layer boundaries (app→pages→widgets→features→entities→shared)
- Node Registry pattern replaces hardcoded switch statements
- Typed Event Bus for SSE dispatch (Observer pattern)
- Store purification removes all cross-store coupling
- Public API (index.ts) per slice enforces encapsulation
- ESLint boundaries rule enforces import direction at lint time"
```

---

## Summary

| Phase | Tasks | Key change |
|-------|-------|-----------|
| Setup | 1 | Directory scaffold, eslint-plugin-boundaries |
| shared/ | 2 | API client, types, utils |
| entities/ | 3-6 | Node registry, purified stores, domain APIs |
| features/ | 7-11 | Cross-store orchestration, event bus, node editors, generate, upload |
| widgets/ | 12 | Canvas, RightPanel, NodePalette, Console |
| pages/ | 13 | Thin page shells |
| app/ | 14 | Router + providers |
| Tooling | 15 | ESLint boundaries |
| Cleanup | 16-17 | Remove old dirs, final verification |
