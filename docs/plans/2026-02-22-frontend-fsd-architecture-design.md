# Frontend FSD Architecture Design

**Date**: 2026-02-22
**Scope**: Complete frontend restructure — Feature-Sliced Design (FSD)
**Motivation**: Maintainability improvement, dependency removal (cross-store coupling, ad-hoc API calls from components)

---

## Goals

- Remove hidden dependencies between Zustand stores
- Enforce clear layer boundaries with strict import direction
- Group code by domain concern, not by file type
- Enable each slice to be changed independently without cascading edits
- Apply 4 design patterns: Node Registry, Event Bus, Store Purification, Public API

---

## Architecture: Feature-Sliced Design

FSD organizes code into horizontal layers, each with vertical slices (domains).
**Import rule**: upper layers may import from lower layers only. Same-layer imports are forbidden.

```
app → pages → widgets → features → entities → shared
```

---

## Directory Structure

```
web/src/
  app/
    router.tsx
    providers.tsx
    main.tsx

  pages/
    editor/
      index.tsx
    landing/
      index.tsx
    runs/
      index.tsx
    pipelines/
      index.tsx
    connections/
      index.tsx

  widgets/
    workflow-canvas/
      ui/Canvas.tsx
      ui/UpalNode.tsx
      ui/GroupNode.tsx
      index.ts
    right-panel/
      ui/RightPanel.tsx
      index.ts
    node-palette/
      ui/NodePalette.tsx
      index.ts
    bottom-console/
      ui/Console.tsx
      index.ts
    workflow-header/
      ui/WorkflowHeader.tsx
      index.ts
    run-detail/
      ui/RunDetail.tsx
      index.ts
    pipeline-editor/
      ui/PipelineEditor.tsx
      index.ts

  features/
    execute-workflow/
      model/useExecuteRun.ts
      model/useReconnectRun.ts
      model/eventDispatcher.ts     ← typed event bus (Observer pattern)
      index.ts
    edit-node/
      ui/NodeEditor.tsx            ← generic editor shell
      ui/AgentNodeEditor.tsx
      ui/InputNodeEditor.tsx
      ui/OutputNodeEditor.tsx
      ui/AssetNodeEditor.tsx
      ui/AIChatEditor.tsx
      model/useNodeEditor.ts
      index.ts
    generate-workflow/
      ui/GenerateDialog.tsx
      model/useGenerateWorkflow.ts
      api/index.ts
      index.ts
    manage-canvas/
      model/useCanvasActions.ts    ← orchestrates workflow + ui stores
      model/useKeyboardShortcuts.ts
      model/useAutoSave.ts
      index.ts
    upload-asset/
      model/useUploadAsset.ts
      api/index.ts
      index.ts
    generate-pipeline/
      ui/GeneratePipelineDialog.tsx
      model/useGeneratePipeline.ts
      api/index.ts
      index.ts

  entities/
    workflow/
      model/store.ts               ← pure canvas state (nodes, edges, name)
      api/index.ts                 ← workflow CRUD
      lib/serializer.ts
      lib/layout.ts
      types.ts
      index.ts
    run/
      model/store.ts               ← pure execution state (events, statuses, timings)
      api/index.ts                 ← run CRUD + SSE connection
      types.ts
      index.ts
    node/
      model/registry.ts            ← Node Registry (Factory pattern)
      types.ts
      index.ts
    ui/
      model/store.ts               ← pure UI state (selectedNodeId, toasts)
      index.ts
    pipeline/
      model/store.ts
      api/index.ts
      types.ts
      index.ts

  shared/
    api/
      client.ts                    ← apiFetch, ApiError (unchanged)
      index.ts
    ui/                            ← Shadcn/ui primitives (unchanged)
    lib/
      utils.ts
      promptSerialization.ts
      nodeConfigs.ts
    types/
      index.ts                     ← shared TypeScript types (RunEvent union, etc.)
    config/
      constants.ts
```

---

## Design Patterns

### 1. Node Registry (Factory + Strategy)

Eliminates `switch(nodeType)` scattered across the codebase. New node types are added by registering, not by editing switch statements.

```typescript
// entities/node/model/registry.ts
interface NodeTypeDefinition {
  type: NodeType
  icon: ComponentType
  label: string
  color: string
  EditorComponent: ComponentType<NodeEditorProps>
  defaultConfig: Record<string, unknown>
}

const registry = new Map<NodeType, NodeTypeDefinition>()

export function registerNode(def: NodeTypeDefinition) {
  registry.set(def.type, def)
}

export function getNodeDefinition(type: NodeType): NodeTypeDefinition {
  const def = registry.get(type)
  if (!def) throw new Error(`Unknown node type: ${type}`)
  return def
}

// Registrations (e.g., in entities/node/model/registry.ts bottom section)
registerNode({ type: 'agent', icon: Bot, EditorComponent: AgentNodeEditor, ... })
registerNode({ type: 'input', icon: FileInput, EditorComponent: InputNodeEditor, ... })
```

```typescript
// features/edit-node/ui/NodeEditor.tsx  — no switch needed
const { EditorComponent } = getNodeDefinition(data.nodeType)
return <EditorComponent config={config} onChange={setConfig} />
```

### 2. Event Bus (Observer pattern)

Decouples SSE event reception from store updates. The executor emits typed events; stores subscribe independently.

```typescript
// features/execute-workflow/model/eventDispatcher.ts
type RunEventMap = {
  node_started: { nodeId: string; startedAt: number }
  node_completed: { nodeId: string; output: unknown; duration: number }
  node_error: { nodeId: string; error: string }
  node_skipped: { nodeId: string }
  run_done: { sessionState: Record<string, unknown> }
}

const runEventBus = createEventBus<RunEventMap>()
export { runEventBus }
```

```typescript
// entities/run/model/store.ts — subscribes during store initialization
runEventBus.on('node_started', ({ nodeId, startedAt }) => {
  useRunStore.getState().setNodeStatus(nodeId, 'running', startedAt)
})
```

### 3. Store Purification

Stores hold state only. All cross-entity orchestration is owned by feature-layer hooks.

```typescript
// ❌ Current (workflowStore.ts)
removeNode: (id) => {
  set(state => ({ nodes: state.nodes.filter(n => n.id !== id) }))
  useUIStore.getState().setSelectedNode(null)  // ← cross-store coupling
  deleteFile(fileId)                            // ← API call in store
}

// ✅ FSD (entities/workflow/model/store.ts)
removeNode: (id) => set(state => ({
  nodes: state.nodes.filter(n => n.id !== id)
}))

// ✅ FSD (features/manage-canvas/model/useCanvasActions.ts)
const deleteNode = useCallback((id: string) => {
  const node = workflowStore.getNode(id)
  workflowStore.removeNode(id)          // ← pure state update
  uiStore.clearSelection()              // ← orchestrated here
  if (node?.data.fileId) {
    deleteFile(node.data.fileId)        // ← side effect here
  }
}, [])
```

### 4. Public API via index.ts

Each slice controls its surface area. Internal modules are not directly importable from outside.

```typescript
// entities/workflow/index.ts
export { useWorkflowStore } from './model/store'
export { serializeWorkflow, deserializeWorkflow } from './lib/serializer'
export { applyAutoLayout } from './lib/layout'
export type { WorkflowDefinition, NodeData } from './types'
// ← api/, internal helpers are NOT exported
```

Enforced by ESLint `import/no-internal-modules` rule (see tooling section).

---

## State Management Changes

| Store | Current location | New location | Changes |
|-------|-----------------|--------------|---------|
| Workflow canvas | `stores/workflowStore.ts` | `entities/workflow/model/store.ts` | Remove `useUIStore` import, remove `deleteFile` call, remove API calls |
| Execution state | `stores/executionStore.ts` | `entities/run/model/store.ts` | Subscribe to eventDispatcher instead of being called directly |
| UI state | `stores/uiStore.ts` | `entities/ui/model/store.ts` | No structural change |

All cross-store coordination moves to `features/manage-canvas/model/useCanvasActions.ts` or the relevant feature hook.

---

## API Layer Changes

```
Current:                    FSD:
lib/api/client.ts     →     shared/api/client.ts        (unchanged)
lib/api/workflows.ts  →     entities/workflow/api/
lib/api/runs.ts       →     entities/run/api/
lib/api/pipelines.ts  →     entities/pipeline/api/
lib/api/models.ts     →     shared/api/models.ts         (no entity owner)
lib/api/tools.ts      →     shared/api/tools.ts
lib/api/upload.ts     →     features/upload-asset/api/
lib/api/configure.ts  →     features/edit-node/api/
lib/api/generate.ts   →     features/generate-workflow/api/
lib/api/triggers.ts   →     entities/pipeline/api/       (or shared)
lib/api/connections.ts →    pages/connections/api/       (or shared)
```

---

## Migration Order

Execute bottom-up (dependencies first):

| Phase | Target | Key task |
|-------|--------|----------|
| 1 | `shared/` | Move API client, utils, types; no logic change |
| 2 | `entities/` | Restructure stores (purify), move domain API modules |
| 3 | `features/` | Move hooks; introduce eventDispatcher; implement useCanvasActions |
| 4 | `widgets/` | Move Canvas, RightPanel, NodePalette; remove internal logic |
| 5 | `pages/` | Slim down to thin page shells |
| 6 | `app/` | Consolidate router and providers |

---

## Tooling

Add ESLint rule to enforce import direction:

```json
// .eslintrc or eslint.config.js addition
"boundaries/element-types": ["error", {
  "default": "disallow",
  "rules": [
    { "from": "app",      "allow": ["pages", "widgets", "features", "entities", "shared"] },
    { "from": "pages",    "allow": ["widgets", "features", "entities", "shared"] },
    { "from": "widgets",  "allow": ["features", "entities", "shared"] },
    { "from": "features", "allow": ["entities", "shared"] },
    { "from": "entities", "allow": ["shared"] },
    { "from": "shared",   "allow": [] }
  ]
}]
```

Plugin: `eslint-plugin-boundaries`

---

## What Does NOT Change

- `shared/ui/` — Shadcn/ui components, no restructuring
- CSS theme system (`index.css`) — Tailwind v4 + oklch, unchanged
- React Flow integration — moved inside `widgets/workflow-canvas/`, not restructured
- Zustand as state library — kept, slice structure changes
- `shared/api/client.ts` — `apiFetch` is already well-separated

---

## File Count Estimate

| Area | Current | After FSD |
|------|---------|-----------|
| Components | 48 | ~50 (split across widgets/ + features/ui/) |
| Stores | 3 | 5 (one per entity) |
| Hooks | 10 | ~12 (in features/*/model/) |
| API modules | 12 | ~14 (distributed to entities/ + features/) |
| index.ts (new) | 0 | ~20 (slice public APIs) |
| **Total** | **~110** | **~135** |
