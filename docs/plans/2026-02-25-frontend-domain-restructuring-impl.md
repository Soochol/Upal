# Frontend Domain Restructuring — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure frontend entities so each domain owns its types/api/store/exports, eliminate duplication, fix FSD violations.

**Architecture:** Vertical domain slices — complete one domain (types → API → store → imports → verify) before moving to next. Backward compatibility via re-exports in shared/types during transition.

**Tech Stack:** React 19, TypeScript, Zustand, TanStack React Query, Vite, FSD architecture

**Design doc:** `docs/plans/2026-02-25-frontend-domain-restructuring.md`

---

## Task 1: Baseline — verify current build passes

**Files:** None (read-only)

**Step 1: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS (or note existing errors to exclude from our scope)

**Step 2: Run lint**

Run: `cd /home/dev/code/Upal/web && npm run lint`
Expected: PASS

---

## Task 2: Pipeline — create entity types

**Files:**
- Create: `web/src/entities/pipeline/types.ts`
- Modify: `web/src/shared/types/index.ts` (remove pipeline types, add re-exports)

**Step 1: Create `web/src/entities/pipeline/types.ts`**

Move these types from `shared/types/index.ts` lines 220-355:

```typescript
// Pipeline domain types

export type PipelineSourceType =
  | 'rss'
  | 'hn'
  | 'reddit'
  | 'google_trends'
  | 'social'
  | 'http'

export type PipelineSource = {
  id: string
  type: PipelineSourceType
  source_type: 'static' | 'signal'
  label: string
  url?: string
  subreddit?: string
  min_score?: number
  keywords?: string[]
  accounts?: string[]
  geo?: string
  limit?: number
}

export type PipelineContext = {
  purpose: string
  target_audience: string
  tone_style: string
  focus_keywords: string[]
  exclude_keywords: string[]
  language: string
}

export type PipelineWorkflow = {
  workflow_name: string
  label?: string
  auto_select?: boolean
  channel_id?: string
}

export type Pipeline = {
  id: string
  name: string
  description?: string
  stages: Stage[]
  thumbnail_svg?: string
  sources?: PipelineSource[]
  workflows?: PipelineWorkflow[]
  context?: PipelineContext
  schedule?: string
  model?: string
  last_collected_at?: string
  pending_session_count?: number
  created_at: string
  updated_at: string
}

export type Stage = {
  id: string
  name: string
  type: 'workflow' | 'approval' | 'notification' | 'schedule' | 'trigger' | 'transform' | 'collect'
  config: StageConfig
  depends_on?: string[]
}

export type CollectSource = {
  id: string
  type: 'rss' | 'http' | 'scrape'
  url: string
  limit?: number
  method?: string
  headers?: Record<string, string>
  body?: string
  selector?: string
  attribute?: string
  scrape_limit?: number
}

export type StageConfig = {
  workflow_name?: string
  input_mapping?: Record<string, string>
  message?: string
  connection_id?: string
  subject?: string
  timeout?: number
  cron?: string
  timezone?: string
  schedule_id?: string
  trigger_id?: string
  expression?: string
  sources?: CollectSource[]
}

export type PipelineRun = {
  id: string
  pipeline_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'failed'
  current_stage?: string
  stage_results?: Record<string, StageResult>
  started_at: string
  completed_at?: string
}

export type StageResult = {
  stage_id: string
  status: 'pending' | 'running' | 'waiting' | 'completed' | 'skipped' | 'failed'
  output?: Record<string, unknown>
  error?: string
  started_at: string
  completed_at?: string
}

// --- Multi-Workflow Session ---

export type SessionStage = 'collect' | 'analyze' | 'workflow' | 'publish'
export type SessionStatus = 'pending' | 'running' | 'pending_approval' | 'completed' | 'failed' | 'cancelled'

export type PipelineSession = {
  id: string
  pipeline_id: string
  status: SessionStatus
  current_stage: SessionStage
  source_data?: Record<string, unknown>[]
  analyzed_data?: Record<string, unknown>
  injected_context?: Record<string, unknown>
  workflow_runs?: Record<string, string>
  publish_results?: Record<string, unknown>
  error?: string
  created_at: string
  updated_at: string
}
```

**Step 2: Replace pipeline types in `shared/types/index.ts` with re-exports**

Remove lines 220-355 (the `// --- Pipeline ---` through `PipelineSession` sections) and replace with:

```typescript
// --- Pipeline (re-exported from entity) ---
export type {
  PipelineSourceType, PipelineSource, PipelineContext, PipelineWorkflow,
  Pipeline, Stage, CollectSource, StageConfig, PipelineRun, StageResult,
  SessionStage, SessionStatus, PipelineSession,
} from '@/entities/pipeline/types'
```

**Step 3: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS — re-exports maintain backward compatibility

**Step 4: Commit**

```bash
git add web/src/entities/pipeline/types.ts web/src/shared/types/index.ts
git commit -m "refactor(pipeline): move domain types to entity"
```

---

## Task 3: Pipeline — deduplicate API

**Files:**
- Delete: `web/src/shared/api/pipelines.ts`
- Modify: `web/src/shared/api/index.ts` (remove pipelines re-export)
- Modify: `web/src/entities/pipeline/api/index.ts` (update imports to use entity types)

**Step 1: Update `entities/pipeline/api/index.ts` imports**

Change line 2 from:
```typescript
import type { Pipeline, PipelineRun, PipelineSource, PipelineContext } from '@/shared/types'
```
to:
```typescript
import type { Pipeline, PipelineRun, PipelineSource, PipelineContext } from '../types'
```

**Step 2: Delete `web/src/shared/api/pipelines.ts`**

Remove the file entirely.

**Step 3: Remove pipelines from `web/src/shared/api/index.ts`**

Remove the line:
```typescript
export * from './pipelines'
```

**Step 4: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS — no files import directly from `@/shared/api/pipelines` (verified by grep: all consumers use entity API or barrel)

**Step 5: Commit**

```bash
git add -A web/src/shared/api/pipelines.ts web/src/shared/api/index.ts web/src/entities/pipeline/api/index.ts
git commit -m "refactor(pipeline): remove duplicate API, use entity as canonical source"
```

---

## Task 4: Pipeline — update barrel export

**Files:**
- Modify: `web/src/entities/pipeline/index.ts`

**Step 1: Update `entities/pipeline/index.ts`**

Replace the current single-line content with:

```typescript
export * from './types'
export * from './api/index'
```

**Step 2: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/entities/pipeline/index.ts
git commit -m "refactor(pipeline): export types from entity barrel"
```

---

## Task 5: Pipeline — migrate consumer imports (pages)

**Files:**
- Modify: `web/src/pages/Pipelines.tsx:19`
- Modify: `web/src/pages/pipelines/PipelineSettingsPanel.tsx:12`
- Modify: `web/src/pages/pipelines/WorkflowPicker.tsx:5`
- Modify: `web/src/pages/pipelines/PipelineNew.tsx:14`
- Modify: `web/src/pages/pipelines/PipelineSidebar.tsx:8`
- Modify: `web/src/pages/pipelines/session/SessionDetailPreview.tsx:8`
- Modify: `web/src/pages/pipelines/session/SessionSetupView.tsx:21`

**Step 1: Update all page imports**

For each file, change:
```typescript
import type { PipelineSource, PipelineContext, PipelineWorkflow, Pipeline, ... } from '@/shared/types'
```
to:
```typescript
import type { PipelineSource, PipelineContext, PipelineWorkflow, Pipeline, ... } from '@/entities/pipeline'
```

Keep non-pipeline types (e.g. `ContentSessionStatus`) importing from their current source.

**Step 2: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/
git commit -m "refactor(pipeline): migrate page imports to entity"
```

---

## Task 6: Pipeline — migrate consumer imports (features + widgets)

**Files:**
- Modify: `web/src/features/configure-pipeline-sources/AddSourceModal.tsx:4`
- Modify: `web/src/features/configure-pipeline/ui/PipelineChatEditor.tsx:4`
- Modify: `web/src/features/define-editorial-brief/EditorialBriefForm.tsx:5`
- Modify: `web/src/widgets/floating-chat/FloatingChat.tsx:7`
- Modify: `web/src/widgets/pipeline-editor/ui/PipelineRunHistory.tsx:4`
- Modify: `web/src/widgets/pipeline-editor/ui/PipelineEditor.tsx:9`
- Modify: `web/src/widgets/pipeline-editor/ui/StageCard.tsx:4`
- Modify: `web/src/widgets/pipeline-editor/ui/PipelineCard.tsx:2`
- Modify: `web/src/widgets/pipeline-editor/ui/PipelineRunDetail.tsx:6`

**Step 1: Update all feature/widget imports**

Same pattern: change `from '@/shared/types'` to `from '@/entities/pipeline'` for pipeline-related types only.

Note: `StageCard.tsx` also imports `Connection` from `@/shared/types` — keep that import separate until Phase 3.

Note: `FloatingChat.tsx` imports `CreatedWorkflowInfo` from `@/shared/types` — this type doesn't exist in the current shared/types (likely stale). Investigate and either remove or define it.

**Step 2: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/features/ web/src/widgets/
git commit -m "refactor(pipeline): migrate feature/widget imports to entity"
```

---

## Task 7: Run — create entity types

**Files:**
- Create: `web/src/entities/run/types.ts`
- Modify: `web/src/shared/types/index.ts` (remove run types, add re-exports)

**Step 1: Create `web/src/entities/run/types.ts`**

Move from `shared/types/index.ts` lines 88-204:

```typescript
// Run domain types

export type RunRecord = {
  id: string
  workflow_name: string
  workflow_definition?: {
    name: string
    version: number
    nodes: { id: string; type: string; config: Record<string, unknown>; group?: string }[]
    edges: { from: string; to: string; loop?: { max_iterations: number; exit_when: string }; condition?: string; trigger_rule?: 'on_success' | 'on_failure' | 'always' }[]
    groups?: { id: string; label: string; color?: string }[]
  }
  trigger_type: string
  trigger_ref: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'cancelled' | 'retrying'
  inputs: Record<string, unknown>
  outputs?: Record<string, unknown>
  error?: string
  retry_of?: string
  retry_count: number
  created_at: string
  started_at?: string
  completed_at?: string
  node_runs?: NodeRunRecord[]
}

export type NodeRunRecord = {
  node_id: string
  status: 'idle' | 'running' | 'completed' | 'error' | 'waiting' | 'skipped'
  started_at: string
  completed_at?: string
  error?: string
  retry_count: number
}

export type RunListResponse = {
  runs: RunRecord[]
  total: number
}

// --- Workflow Execution Events ---

export type ToolCall = { name: string; args?: Record<string, unknown> }
export type ToolResult = { name: string; response?: Record<string, unknown> }

export type NodeStartedEvent = { type: 'node_started'; nodeId: string; startedAt?: number }
export type ToolCallEvent = { type: 'tool_call'; nodeId: string; calls: ToolCall[] }
export type ToolResultEvent = { type: 'tool_result'; nodeId: string; results: ToolResult[] }
export type TokenUsage = {
  input: number
  output: number
  total: number
}

export type NodeCompletedEvent = {
  type: 'node_completed'
  nodeId: string
  output: string
  stateDelta: Record<string, unknown>
  tokens?: TokenUsage
  finishReason?: string
  completedAt?: number
}
export type NodeSkippedEvent = { type: 'node_skipped'; nodeId: string }
export type NodeWaitingEvent = { type: 'node_waiting'; nodeId: string }
export type NodeResumedEvent = { type: 'node_resumed'; nodeId: string }
export type WorkflowDoneEvent = { type: 'done'; status: string; sessionId: string; state: Record<string, unknown>; error?: string }
export type WorkflowErrorEvent = { type: 'error'; message: string }
export type InfoEvent = { type: 'info'; message: string }
export type LogEvent = { type: 'log'; nodeId: string; message: string }

export type RunEvent =
  | NodeStartedEvent | ToolCallEvent | ToolResultEvent
  | NodeCompletedEvent | NodeSkippedEvent | NodeWaitingEvent | NodeResumedEvent
  | WorkflowDoneEvent | WorkflowErrorEvent
  | InfoEvent | LogEvent
```

**Step 2: Replace run types in `shared/types/index.ts` with re-exports**

Remove lines 88-204 and replace with:

```typescript
// --- Run (re-exported from entity) ---
export type {
  RunRecord, NodeRunRecord, RunListResponse,
  ToolCall, ToolResult, TokenUsage,
  NodeStartedEvent, ToolCallEvent, ToolResultEvent, NodeCompletedEvent,
  NodeSkippedEvent, NodeWaitingEvent, NodeResumedEvent,
  WorkflowDoneEvent, WorkflowErrorEvent, InfoEvent, LogEvent, RunEvent,
} from '@/entities/run/types'
```

**Step 3: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/entities/run/types.ts web/src/shared/types/index.ts
git commit -m "refactor(run): move domain types to entity"
```

---

## Task 8: Run — deduplicate API + fix internal imports

**Files:**
- Delete: `web/src/shared/api/runs.ts`
- Modify: `web/src/shared/api/index.ts` (remove runs re-export)
- Modify: `web/src/entities/run/api/index.ts` (import from entity types)
- Modify: `web/src/entities/run/model/store.ts` (import from entity, not shared)
- Modify: `web/src/entities/run/index.ts` (export types)

**Step 1: Update `entities/run/api/index.ts` line 2**

Change:
```typescript
import type { RunRecord, RunListResponse, RunEvent, ToolCall, TokenUsage } from '@/shared/types'
```
to:
```typescript
import type { RunRecord, RunListResponse, RunEvent, ToolCall, TokenUsage } from '../types'
```

**Step 2: Update `entities/run/model/store.ts` lines 2-3**

Change:
```typescript
import type { RunEvent } from '@/shared/types'
import { startRun as apiStartRun, connectToRunEvents } from '@/shared/api'
```
to:
```typescript
import type { RunEvent } from '../types'
import { startRun as apiStartRun, connectToRunEvents } from '../api/index'
```

**Step 3: Update `entities/run/index.ts`**

Change to:
```typescript
export * from './types'
export { useExecutionStore } from './model/store'
export type { NodeRunStatus } from './model/store'
export * from './api/index'
```

**Step 4: Delete `web/src/shared/api/runs.ts`**

Remove the file.

**Step 5: Remove runs from `web/src/shared/api/index.ts`**

Remove the line:
```typescript
export * from './runs'
```

**Step 6: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 7: Commit**

```bash
git add -A web/src/entities/run/ web/src/shared/api/runs.ts web/src/shared/api/index.ts
git commit -m "refactor(run): remove duplicate API, use entity types internally"
```

---

## Task 9: Run — migrate consumer imports

**Files:**
- Modify: `web/src/pages/Runs.tsx:4`
- Modify: `web/src/pages/runs/RunNodePanel.tsx:6`
- Modify: `web/src/pages/runs/RunViewer.tsx:10`
- Modify: `web/src/features/execute-workflow/model/useExecuteRun.ts:4`
- Modify: `web/src/widgets/run-detail/ui/NodeDetail.tsx:2`
- Modify: `web/src/widgets/run-detail/ui/NodeTimeline.tsx:1`
- Modify: `web/src/widgets/run-detail/ui/TimelineNode.tsx:2`
- Modify: `web/src/widgets/run-detail/ui/RunDetail.tsx:4`
- Modify: `web/src/widgets/right-panel/ui/PanelConsole.tsx:16`
- Modify: `web/src/widgets/bottom-console/ui/Console.tsx:17`
- Modify: `web/src/widgets/right-panel/ui/preview/ResultsDisplay.tsx:2`

**Step 1: Update all consumer imports**

For each file, change:
```typescript
import type { RunRecord, RunEvent, NodeRunRecord, NodeCompletedEvent, ... } from '@/shared/types'
```
to:
```typescript
import type { RunRecord, RunEvent, NodeRunRecord, NodeCompletedEvent, ... } from '@/entities/run'
```

**Step 2: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/ web/src/features/ web/src/widgets/
git commit -m "refactor(run): migrate consumer imports to entity"
```

---

## Task 10: Connection — create new entity

**Files:**
- Create: `web/src/entities/connection/types.ts`
- Create: `web/src/entities/connection/api.ts`
- Create: `web/src/entities/connection/index.ts`
- Modify: `web/src/shared/types/index.ts` (remove connection types, add re-exports)
- Delete: `web/src/shared/api/connections.ts`
- Modify: `web/src/shared/api/index.ts` (remove connections re-export)

**Step 1: Create `web/src/entities/connection/types.ts`**

Move from `shared/types/index.ts` lines 358-381:

```typescript
export type ConnectionType = 'telegram' | 'slack' | 'http' | 'smtp'

export type Connection = {
  id: string
  name: string
  type: ConnectionType
  host?: string
  port?: number
  login?: string
  extras?: Record<string, string>
}

export type ConnectionCreate = {
  name: string
  type: ConnectionType
  host?: string
  port?: number
  login?: string
  password?: string
  token?: string
  extras?: Record<string, string>
}
```

**Step 2: Create `web/src/entities/connection/api.ts`**

```typescript
import { apiFetch } from '@/shared/api/client'
import type { Connection, ConnectionCreate } from './types'

export async function listConnections(): Promise<Connection[]> {
  return apiFetch<Connection[]>('/api/connections')
}

export async function createConnection(conn: ConnectionCreate): Promise<Connection> {
  return apiFetch<Connection>('/api/connections', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(conn),
  })
}

export async function deleteConnection(id: string): Promise<void> {
  await apiFetch<void>(`/api/connections/${id}`, { method: 'DELETE' })
}
```

**Step 3: Create `web/src/entities/connection/index.ts`**

```typescript
export * from './types'
export * from './api'
```

**Step 4: Replace connection types in `shared/types/index.ts` with re-exports**

Remove the connection type definitions (lines 358-381) and replace with:

```typescript
// --- Connection (re-exported from entity) ---
export type { ConnectionType, Connection, ConnectionCreate } from '@/entities/connection'
```

**Step 5: Delete `web/src/shared/api/connections.ts`**

Remove the file.

**Step 6: Remove connections from `web/src/shared/api/index.ts`**

Remove the line:
```typescript
export * from './connections'
```

**Step 7: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 8: Commit**

```bash
git add -A web/src/entities/connection/ web/src/shared/types/index.ts web/src/shared/api/connections.ts web/src/shared/api/index.ts
git commit -m "refactor(connection): create entity with types + API"
```

---

## Task 11: Connection — migrate consumer imports

**Files:**
- Modify: `web/src/pages/Connections.tsx:6-7`
- Modify: `web/src/widgets/pipeline-editor/ui/StageCard.tsx:4-5`
- Modify: `web/src/widgets/pipeline-editor/ui/PipelineEditor.tsx:8`

**Step 1: Update Connections.tsx**

Change:
```typescript
import { listConnections, createConnection, deleteConnection } from '@/shared/api'
import type { ConnectionCreate, ConnectionType } from '@/shared/types'
```
to:
```typescript
import { listConnections, createConnection, deleteConnection } from '@/entities/connection'
import type { ConnectionCreate, ConnectionType } from '@/entities/connection'
```

**Step 2: Update StageCard.tsx**

Change `Connection` import from `@/shared/types` to `@/entities/connection`.

**Step 3: Update PipelineEditor.tsx**

Change `listConnections` import from `@/shared/api` to `@/entities/connection`.

**Step 4: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 5: Commit**

```bash
git add web/src/pages/Connections.tsx web/src/widgets/pipeline-editor/
git commit -m "refactor(connection): migrate consumer imports to entity"
```

---

## Task 12: Workflow — fix FSD violation

**Files:**
- Modify: `web/src/pages/pipelines/WorkflowPicker.tsx:4,38`

**Step 1: Update WorkflowPicker.tsx**

Change line 4 from:
```typescript
import { apiFetch } from '@/shared/api/client'
```
to:
```typescript
import { listWorkflows } from '@/entities/workflow'
```

Change line 38 query function from:
```typescript
queryFn: () => apiFetch<WorkflowListItem[]>('/api/workflows'),
```
to:
```typescript
queryFn: () => listWorkflows(),
```

Note: `listWorkflows()` returns `WorkflowDefinition[]` which has `name` and `description` fields, so `WorkflowListItem` type remains compatible. Verify the shape matches or adjust the local type.

**Step 2: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/pages/pipelines/WorkflowPicker.tsx
git commit -m "refactor(workflow): fix FSD violation in WorkflowPicker"
```

---

## Task 13: Content Session — verify and consolidate types

**Files:**
- Modify: `web/src/entities/content-session/types.ts` (potentially)
- Modify: `web/src/shared/types/index.ts` (potentially)

**Step 1: Evaluate ContentSessionStatus and SourceType location**

These types are currently in `shared/types` and re-exported from `entities/content-session/types.ts`. Check which non-content-session files import them:

Run grep to confirm consumers. If only content-session consumers use them, move to entity. If used by pipeline or other entities, keep in shared.

Current observation from grep: `ContentSessionStatus` and `SourceType` are only consumed by content-session entity itself — the entity types.ts imports them from shared and re-exports. **Decision: move to entity** since they are semantically content-session domain types, and content-session already re-exports them.

**Step 2: Move types to entity (if appropriate)**

In `entities/content-session/types.ts`, change:
```typescript
import type { ContentSessionStatus, SourceType } from '@/shared/types'
export type { ContentSessionStatus, SourceType }
```
to defining them directly:
```typescript
export type ContentSessionStatus =
  | 'collecting'
  | 'analyzing'
  | 'pending_review'
  | 'approved'
  | 'rejected'
  | 'producing'
  | 'published'
  | 'error'

export type SourceType = 'static' | 'signal'
```

In `shared/types/index.ts`, replace the definitions with re-exports:
```typescript
// --- Content Session (re-exported from entity) ---
export type { ContentSessionStatus, SourceType } from '@/entities/content-session'
```

**Step 3: Run type checker**

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/entities/content-session/types.ts web/src/shared/types/index.ts
git commit -m "refactor(content-session): own ContentSessionStatus and SourceType types"
```

---

## Task 14: Settings — add barrel export

**Files:**
- Create: `web/src/entities/settings/index.ts`

**Step 1: Create `web/src/entities/settings/index.ts`**

```typescript
export { useSettingsStore } from './store'
```

**Step 2: Verify** — check if any files import directly from `@/entities/settings/store` and update to use `@/entities/settings`.

Run: `cd /home/dev/code/Upal/web && npx tsc -b`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/entities/settings/index.ts
git commit -m "refactor(settings): add barrel export"
```

---

## Task 15: Shared layer — final cleanup

**Files:**
- Modify: `web/src/shared/types/index.ts` (verify only infra types + re-exports remain)
- Modify: `web/src/shared/api/index.ts` (verify cleaned up)

**Step 1: Audit `shared/types/index.ts`**

After all phases, this file should contain:

1. **Infrastructure types (defined here):** OptionChoice, OptionSchema, ModelInfo, ToolInfo, UploadResult, ConfigureNodeRequest/Response, ConfigurePipelineRequest/Response, Schedule, RetryPolicy, Trigger, SchedulerStats
2. **Re-exports:** Pipeline types from `@/entities/pipeline`, Run types from `@/entities/run`, Connection types from `@/entities/connection`, ContentSession types from `@/entities/content-session`

Verify no orphaned domain types remain.

**Step 2: Audit `shared/api/index.ts`**

After removing pipelines, runs, connections, it should contain:
```typescript
export { ApiError, apiFetch, API_BASE } from './client'
export * from './upload'
export * from './models'
export * from './tools'
export { useModels } from './useModels'
export * from './triggers'
export * from './configure'
export * from './stats'
```

**Step 3: Run full validation**

Run: `cd /home/dev/code/Upal/web && npx tsc -b && npm run lint`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/shared/
git commit -m "refactor(shared): clean up after domain type migration"
```

---

## Task 16: Final verification

**Files:** None (read-only)

**Step 1: Full build**

Run: `cd /home/dev/code/Upal/web && npm run build`
Expected: PASS

**Step 2: Verify no remaining FSD violations**

Run grep to check:
- No files import pipeline types from `@/shared/types` (only re-exports should reference entity)
- No files import run types from `@/shared/types` directly
- No files import connection types from `@/shared/types` directly
- No files use `apiFetch` directly in pages (except shared/api modules)
- `@/shared/api/pipelines`, `@/shared/api/runs`, `@/shared/api/connections` don't exist

**Step 3: Audit entity completeness**

| Entity | types.ts | api.ts | store.ts | index.ts |
|--------|----------|--------|----------|----------|
| pipeline | ✅ | ✅ | ❌ (React Query OK) | ✅ |
| run | ✅ | ✅ | ✅ | ✅ |
| connection | ✅ | ✅ | ❌ (not needed) | ✅ |
| workflow | ✅ | ✅ | ✅ | ✅ |
| content-session | ✅ | ✅ | ✅ | ✅ |
| settings | ❌ (not needed) | ❌ (not needed) | ✅ | ✅ |
| node | ✅ | ❌ (not needed) | ✅ (registry) | ✅ |
| ui | ❌ (not needed) | ❌ (not needed) | ✅ | ✅ |

**Step 4: Summary commit (if any remaining changes)**

```bash
git commit -m "refactor: complete frontend domain restructuring"
```
