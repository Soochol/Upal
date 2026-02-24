# Run Canvas Viewer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the `/runs/:id` page with a read-only workflow canvas view that shows the workflow snapshot at execution time, per-node status visualization, and a right panel for inspecting outputs/errors — with live SSE updates for running runs.

**Architecture:** Add `WorkflowDef` field to `RunRecord` (Go + TS), store it at `StartRun()` time, then build a new `RunViewer` page that reuses `<Canvas readOnly>` and a new `<RunRightPanel>` to display the run. The existing `RunDetail` widget is replaced entirely.

**Tech Stack:** Go (Chi), React 19, TypeScript, React Flow, Zustand, TanStack Query, SSE

---

### Task 1: Add `WorkflowDef` to RunRecord (Backend)

Add the workflow definition snapshot field to the domain type, port interface, service, and database layer.

**Files:**
- Modify: `internal/upal/scheduler.go:36-53`
- Modify: `internal/upal/ports/runhistory.go:12`
- Modify: `internal/services/runhistory.go:26-43`
- Modify: `internal/db/run.go:13-30, 33-57, 61-74, 150-172`
- Modify: `internal/db/db.go:88-108` (migration SQL)
- Modify: `internal/repository/run_persistent.go` (no changes needed — passes through)

**Step 1: Add field to RunRecord**

In `internal/upal/scheduler.go`, add `WorkflowDef` to `RunRecord`:

```go
type RunRecord struct {
	ID           string                `json:"id"`
	WorkflowName string               `json:"workflow_name"`
	WorkflowDef  *WorkflowDefinition  `json:"workflow_definition,omitempty"` // ← NEW
	TriggerType  string               `json:"trigger_type"`
	TriggerRef   string               `json:"trigger_ref"`
	Status       RunStatus            `json:"status"`
	Inputs       map[string]any       `json:"inputs"`
	Outputs      map[string]any       `json:"outputs,omitempty"`
	Error        *string              `json:"error,omitempty"`
	RetryOf      *string              `json:"retry_of,omitempty"`
	RetryCount   int                  `json:"retry_count"`
	SessionID    *string              `json:"session_id,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	StartedAt    *time.Time           `json:"started_at,omitempty"`
	CompletedAt  *time.Time           `json:"completed_at,omitempty"`
	NodeRuns     []NodeRunRecord      `json:"node_runs,omitempty"`
	Usage        *TokenUsage          `json:"usage,omitempty"`
}
```

**Step 2: Update port interface**

In `internal/upal/ports/runhistory.go`, update `StartRun` signature:

```go
StartRun(ctx context.Context, workflowName string, triggerType, triggerRef string, inputs map[string]any, wfDef *upal.WorkflowDefinition) (*upal.RunRecord, error)
```

**Step 3: Update RunHistoryService.StartRun**

In `internal/services/runhistory.go:26-43`, accept and store `wfDef`:

```go
func (s *RunHistoryService) StartRun(ctx context.Context, workflowName, triggerType, triggerRef string, inputs map[string]any, wfDef *upal.WorkflowDefinition) (*upal.RunRecord, error) {
	now := time.Now()
	record := &upal.RunRecord{
		ID:           upal.GenerateID("run"),
		WorkflowName: workflowName,
		WorkflowDef:  wfDef,
		TriggerType:  triggerType,
		TriggerRef:   triggerRef,
		Status:       upal.RunStatusRunning,
		Inputs:       inputs,
		CreatedAt:    now,
		StartedAt:    &now,
	}
	if err := s.runRepo.Create(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}
```

**Step 4: Update DB migration**

In `internal/db/db.go`, add after the `ALTER TABLE runs ADD COLUMN IF NOT EXISTS session_id TEXT;` line (line 173):

```sql
ALTER TABLE runs ADD COLUMN IF NOT EXISTS workflow_definition JSONB;
```

**Step 5: Update DB CreateRun**

In `internal/db/run.go` `CreateRun`, add `workflow_definition` to the INSERT:

```go
func (d *DB) CreateRun(ctx context.Context, r *upal.RunRecord) error {
	inputsJSON, _ := json.Marshal(r.Inputs)
	outputsJSON, _ := json.Marshal(r.Outputs)
	nodeRunsJSON, _ := json.Marshal(r.NodeRuns)
	wfDefJSON, _ := json.Marshal(r.WorkflowDef) // ← NEW

	_, err := d.Pool.ExecContext(ctx,
		`INSERT INTO runs (id, workflow_name, trigger_type, trigger_ref, status, inputs, outputs, error, retry_of, retry_count, node_runs, session_id, workflow_definition, created_at, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		r.ID, r.WorkflowName, r.TriggerType, r.TriggerRef,
		string(r.Status), inputsJSON, outputsJSON, r.Error,
		r.RetryOf, r.RetryCount, nodeRunsJSON,
		r.SessionID, wfDefJSON, r.CreatedAt, r.StartedAt, r.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}
```

**Step 6: Update DB GetRun + scanRuns**

In `GetRun`, add `workflow_definition` to SELECT and Scan:

```go
func (d *DB) GetRun(ctx context.Context, id string) (*upal.RunRecord, error) {
	r := &upal.RunRecord{}
	var status string
	var inputsJSON, outputsJSON, nodeRunsJSON, wfDefJSON []byte

	err := d.Pool.QueryRowContext(ctx,
		`SELECT id, workflow_name, trigger_type, trigger_ref, status, inputs, outputs, error, retry_of, retry_count, node_runs, session_id, workflow_definition, created_at, started_at, completed_at
		 FROM runs WHERE id = $1`, id,
	).Scan(&r.ID, &r.WorkflowName, &r.TriggerType, &r.TriggerRef,
		&status, &inputsJSON, &outputsJSON, &r.Error,
		&r.RetryOf, &r.RetryCount, &nodeRunsJSON,
		&r.SessionID, &wfDefJSON, &r.CreatedAt, &r.StartedAt, &r.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("run not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	r.Status = upal.RunStatus(status)
	json.Unmarshal(inputsJSON, &r.Inputs)
	json.Unmarshal(outputsJSON, &r.Outputs)
	json.Unmarshal(nodeRunsJSON, &r.NodeRuns)
	if len(wfDefJSON) > 0 {
		r.WorkflowDef = &upal.WorkflowDefinition{}
		json.Unmarshal(wfDefJSON, r.WorkflowDef)
	}
	return r, nil
}
```

Update `scanRuns` similarly — add `wfDefJSON` to the SELECT queries in `ListRunsByWorkflow` and `ListAllRuns`, and unmarshal in `scanRuns`.

**Important:** All `SELECT ... FROM runs` queries in `run.go` must now include `workflow_definition` in the column list. The `scanRuns` function must add `&wfDefJSON` to the Scan call and unmarshal it.

**Step 7: Update all callers of StartRun**

Two callers need updating:

1. `internal/api/run.go:62` — pass `wf`:
```go
record, err := s.runHistorySvc.StartRun(r.Context(), name, "manual", "", req.Inputs, wf)
```

2. `internal/services/content_collector.go:529` — pass the workflow definition (already available as `wfDef` from Lookup):
```go
rec, startErr := c.runHistorySvc.StartRun(gCtx, req.Name, "pipeline", sessionID, inputs, wfDef)
```
Note: In `content_collector.go`, the workflow is looked up as `wfDef` before execution. Pass it to `StartRun`.

**Step 8: Build and verify**

Run: `go build ./cmd/upal`
Expected: SUCCESS

**Step 9: Commit**

```bash
git add internal/upal/scheduler.go internal/upal/ports/runhistory.go internal/services/runhistory.go internal/db/run.go internal/db/db.go internal/api/run.go internal/services/content_collector.go
git commit -m "feat: store workflow definition snapshot in RunRecord"
```

---

### Task 2: Add `workflow_definition` to frontend RunRecord type

**Files:**
- Modify: `web/src/shared/types/index.ts:90-105`

**Step 1: Update RunRecord type**

Add `workflow_definition` field:

```typescript
export type RunRecord = {
  id: string
  workflow_name: string
  workflow_definition?: {
    name: string
    version: number
    nodes: { id: string; type: string; config: Record<string, unknown>; group?: string }[]
    edges: { from: string; to: string; loop?: { max_iterations: number; exit_when: string }; condition?: string; trigger_rule?: string }[]
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
```

**Step 2: Type-check**

Run: `cd web && npx tsc --noEmit`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add web/src/shared/types/index.ts
git commit -m "feat: add workflow_definition to frontend RunRecord type"
```

---

### Task 3: Build the RunViewer page component

Create a new `RunViewer` page that shows a read-only canvas with the run's workflow snapshot, node status overlays, and a right panel for inspecting node execution details.

**Files:**
- Create: `web/src/pages/runs/RunViewer.tsx`
- Modify: `web/src/app/router.tsx:33` (replace `RunDetail` with `RunViewer`)

**Step 1: Create RunViewer component**

Create `web/src/pages/runs/RunViewer.tsx`:

```tsx
import { useEffect, useState, useCallback, useRef } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { ReactFlowProvider } from '@xyflow/react'
import { fetchRun, connectToRunEvents } from '@/entities/run'
import { deserializeWorkflow, useWorkflowStore } from '@/entities/workflow'
import { useExecutionStore } from '@/entities/run'
import { Canvas } from '@/widgets/workflow-canvas'
import { useUIStore } from '@/entities/ui'
import { useResizeDrag } from '@/shared/lib/useResizeDrag'
import type { RunRecord, RunEvent } from '@/shared/types'
import { Header } from '@/shared/ui/Header'
import {
  ArrowLeft, CheckCircle2, XCircle, Clock, Loader2, Timer,
  ExternalLink, RotateCcw, PanelRightClose, PanelRightOpen,
} from 'lucide-react'
import { RunNodePanel } from './RunNodePanel'

const statusConfig: Record<string, { icon: typeof Clock; color: string; label: string }> = {
  pending:   { icon: Clock,        color: 'text-warning',          label: 'Pending' },
  running:   { icon: Loader2,      color: 'text-info',             label: 'Running' },
  success:   { icon: CheckCircle2, color: 'text-success',          label: 'Success' },
  failed:    { icon: XCircle,      color: 'text-destructive',      label: 'Failed' },
  cancelled: { icon: XCircle,      color: 'text-muted-foreground', label: 'Cancelled' },
  retrying:  { icon: Timer,        color: 'text-warning',          label: 'Retrying' },
}

export function RunViewer() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [run, setRun] = useState<RunRecord | null>(null)
  const [loading, setLoading] = useState(true)
  const [isPanelOpen, setIsPanelOpen] = useState(true)
  const abortRef = useRef<AbortController | null>(null)

  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const selectNode = useUIStore((s) => s.selectNode)
  const setNodeStatus = useExecutionStore((s) => s.setNodeStatus)
  const clearNodeStatuses = useExecutionStore((s) => s.clearNodeStatuses)
  const nodes = useWorkflowStore((s) => s.nodes)

  const selectedNode = selectedNodeId
    ? nodes.find((n) => n.id === selectedNodeId) ?? null
    : null

  const { size: panelWidth, handleMouseDown: onPanelDrag } = useResizeDrag({
    direction: 'horizontal', min: 260, max: 700, initial: 320,
  })

  // Fetch run + load workflow into canvas store
  useEffect(() => {
    if (!id) return
    setLoading(true)
    clearNodeStatuses()

    fetchRun(id)
      .then((r) => {
        setRun(r)

        // Deserialize workflow definition into canvas
        if (r.workflow_definition) {
          const { nodes: n, edges: e } = deserializeWorkflow(r.workflow_definition)
          useWorkflowStore.setState({ nodes: n, edges: e })
        } else {
          useWorkflowStore.setState({ nodes: [], edges: [] })
        }

        // Apply node statuses from completed run
        if (r.node_runs) {
          for (const nr of r.node_runs) {
            const status = nr.status === 'completed' ? 'completed'
              : nr.status === 'error' ? 'error'
              : nr.status === 'running' ? 'running'
              : 'idle'
            setNodeStatus(nr.node_id, status)
          }
        }
      })
      .catch(() => setRun(null))
      .finally(() => setLoading(false))

    return () => selectNode(null)
  }, [id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Live SSE for running runs
  useEffect(() => {
    if (!run || !id) return
    if (run.status !== 'running' && run.status !== 'pending') return

    const abort = new AbortController()
    abortRef.current = abort

    connectToRunEvents(
      id,
      (event: RunEvent) => {
        if (event.type === 'node_started' || event.type === 'tool_call' || event.type === 'tool_result') {
          const nodeId = 'nodeId' in event ? event.nodeId : undefined
          if (nodeId) setNodeStatus(nodeId, 'running')
        } else if (event.type === 'node_completed') {
          const nodeId = 'nodeId' in event ? event.nodeId : undefined
          if (nodeId) setNodeStatus(nodeId, 'completed')
        } else if (event.type === 'node_skipped') {
          const nodeId = 'nodeId' in event ? event.nodeId : undefined
          if (nodeId) setNodeStatus(nodeId, 'skipped')
        }
      },
      (result) => {
        // Run completed — re-fetch to get final state
        fetchRun(id).then(setRun).catch(() => {})
      },
      () => {},
      { signal: abort.signal },
    )

    return () => abort.abort()
  }, [run?.status, id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-open panel on node select
  useEffect(() => {
    if (selectedNodeId) setIsPanelOpen(true)
  }, [selectedNodeId])

  if (loading) {
    return (
      <div className="h-screen flex flex-col bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="animate-spin text-muted-foreground" size={32} />
        </div>
      </div>
    )
  }

  if (!run) {
    return (
      <div className="h-screen flex flex-col bg-background">
        <Header />
        <div className="flex-1 flex items-center justify-center">
          <p className="text-muted-foreground">Run not found</p>
        </div>
      </div>
    )
  }

  const cfg = statusConfig[run.status] ?? statusConfig.pending
  const Icon = cfg.icon
  const isLive = run.status === 'running' || run.status === 'pending'

  function formatDuration(): string {
    if (!run?.started_at) return '-'
    const end = run.completed_at ? new Date(run.completed_at) : new Date()
    const ms = end.getTime() - new Date(run.started_at).getTime()
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header />

      {/* Run header bar */}
      <div className="px-4 md:px-6 py-3 border-b border-border/50 bg-background/80 backdrop-blur-sm shrink-0 shadow-sm z-10 flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/runs')}
            className="text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft size={18} />
          </button>
          <code className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded font-mono">
            {run.id.length > 12 ? `${run.id.slice(0, 12)}...` : run.id}
          </code>
          <div className="flex items-center gap-1.5">
            <Icon size={18} className={`${cfg.color} ${run.status === 'running' ? 'animate-spin' : ''}`} />
            <span className={`text-sm font-semibold ${cfg.color}`}>{cfg.label}</span>
          </div>
          {isLive && (
            <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-info/10 text-info animate-pulse">live</span>
          )}
          <span className="text-xs text-muted-foreground hidden md:inline">
            {run.workflow_name} · {formatDuration()}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {run.status === 'failed' && (
            <button className="flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors cursor-pointer">
              <RotateCcw className="h-3 w-3" />
              Retry
            </button>
          )}
          <Link
            to={`/workflows?w=${encodeURIComponent(run.workflow_name)}`}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-md text-xs border border-border text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          >
            <ExternalLink className="h-3 w-3" />
            Open Workflow
          </Link>
          <button
            onClick={() => setIsPanelOpen((v) => !v)}
            className="hidden md:flex p-2 rounded-xl border border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
          >
            {isPanelOpen ? <PanelRightClose className="w-4 h-4" /> : <PanelRightOpen className="w-4 h-4" />}
          </button>
        </div>
      </div>

      {/* Error banner */}
      {run.error && (
        <div className="mx-4 md:mx-6 mt-2 bg-destructive/10 border border-destructive/20 rounded-lg p-3">
          <p className="text-sm text-destructive font-mono">{run.error}</p>
        </div>
      )}

      {/* Canvas + Right Panel */}
      <div className="flex-1 min-h-0 overflow-hidden flex">
        <div className="flex-1 min-w-0 h-full">
          <ReactFlowProvider>
            <Canvas
              readOnly
              onAddFirstNode={() => {}}
              onDropNode={() => {}}
              onPromptSubmit={() => {}}
              isGenerating={false}
              onAddNode={() => {}}
            />
          </ReactFlowProvider>
        </div>

        {isPanelOpen && (
          <div className="hidden md:contents">
            <div
              onMouseDown={onPanelDrag}
              className="w-1 shrink-0 cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-30 relative
                after:absolute after:inset-y-0 after:-left-1 after:-right-1"
            />
            <aside
              style={{ width: panelWidth }}
              className="border-l border-border bg-sidebar/95 backdrop-blur-md shadow-2xl z-30 flex flex-col shrink-0"
            >
              <RunNodePanel
                selectedNode={selectedNode}
                run={run}
                onClose={() => selectNode(null)}
                onCollapse={() => setIsPanelOpen(false)}
              />
            </aside>
          </div>
        )}
      </div>
    </div>
  )
}
```

**Step 2: Update router**

In `web/src/app/router.tsx`, replace the `RunDetail` import and route:

```tsx
// Remove: import { RunDetail } from '@/widgets/run-detail'
// Add:
import { RunViewer } from '@/pages/runs/RunViewer'

// Change line 33:
// From: <Route path="/runs/:id" element={<RunDetail />} />
// To:
<Route path="/runs/:id" element={<RunViewer />} />
```

**Step 3: Type-check**

Run: `cd web && npx tsc --noEmit`
Expected: May fail on `RunNodePanel` import — that's the next task.

**Step 4: Commit** (after Task 4 completes)

---

### Task 4: Build the RunNodePanel (right panel for run viewer)

A read-only right panel that shows Output, Error, and Info tabs for the selected node.

**Files:**
- Create: `web/src/pages/runs/RunNodePanel.tsx`

**Step 1: Create RunNodePanel**

```tsx
import { useState, useRef, useEffect } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import type { NodeData } from '@/entities/workflow'
import type { RunRecord } from '@/shared/types'
import type { Node } from '@xyflow/react'
import { useExecutionStore, type NodeRunStatus } from '@/entities/run'
import { FileText, AlertTriangle, Info, CheckCircle2, XCircle, Clock, Loader2 } from 'lucide-react'

type Props = {
  selectedNode: Node<NodeData> | null
  run: RunRecord
  onClose: () => void
  onCollapse: () => void
}

const statusDisplay: Record<string, { icon: typeof Clock; color: string; label: string }> = {
  idle:      { icon: Clock,        color: 'text-muted-foreground', label: 'Not Executed' },
  running:   { icon: Loader2,      color: 'text-info',             label: 'Running' },
  completed: { icon: CheckCircle2, color: 'text-success',          label: 'Completed' },
  error:     { icon: XCircle,      color: 'text-destructive',      label: 'Failed' },
  skipped:   { icon: Clock,        color: 'text-muted-foreground', label: 'Skipped' },
  waiting:   { icon: Clock,        color: 'text-warning',          label: 'Waiting' },
}

export function RunNodePanel({ selectedNode, run, onClose, onCollapse }: Props) {
  const [activeTab, setActiveTab] = useState('output')
  const activeTabRef = useRef(activeTab)
  activeTabRef.current = activeTab

  const nodeStatuses = useExecutionStore((s) => s.nodeStatuses)
  const prevNodeIdRef = useRef<string | null>(selectedNode?.id ?? null)

  useEffect(() => {
    if (selectedNode) {
      setActiveTab('output')
    } else if (prevNodeIdRef.current !== null && activeTabRef.current === 'output') {
      onCollapse()
    }
    prevNodeIdRef.current = selectedNode?.id ?? null
  }, [selectedNode?.id, onCollapse])

  const nodeRun = selectedNode
    ? run.node_runs?.find((nr) => nr.node_id === selectedNode.id)
    : null

  const nodeStatus: NodeRunStatus = selectedNode
    ? (nodeStatuses[selectedNode.id] ?? 'idle')
    : 'idle'

  const statusCfg = statusDisplay[nodeStatus] ?? statusDisplay.idle
  const StatusIcon = statusCfg.icon

  const tabs = [
    { value: 'output', label: 'Output', icon: FileText },
    ...(nodeRun?.error ? [{ value: 'error', label: 'Error', icon: AlertTriangle }] : []),
    { value: 'info', label: 'Info', icon: Info },
  ]

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1 min-h-0 gap-0">
      <div className="flex items-center justify-between border-b border-border px-1">
        <TabsList className="h-10 bg-transparent p-0 gap-0">
          {tabs.map((tab) => {
            const TabIcon = tab.icon
            return (
              <TabsTrigger
                key={tab.value}
                value={tab.value}
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:shadow-none px-3 py-2 flex items-center gap-1.5 text-xs font-medium"
              >
                <TabIcon className="h-3.5 w-3.5 shrink-0" />
                {tab.label}
              </TabsTrigger>
            )
          })}
        </TabsList>
      </div>

      {/* Node header */}
      {selectedNode && (
        <div className="px-4 py-3 border-b border-border/50">
          <div className="flex items-center gap-2">
            <StatusIcon className={`h-4 w-4 ${statusCfg.color} ${nodeStatus === 'running' ? 'animate-spin' : ''}`} />
            <span className="text-sm font-semibold">{selectedNode.data.label}</span>
          </div>
          <span className={`text-xs ${statusCfg.color}`}>{statusCfg.label}</span>
        </div>
      )}

      {/* Output tab */}
      <TabsContent value="output" className="flex-1 min-h-0 overflow-auto mt-0 p-4">
        {!selectedNode ? (
          <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
            Select a node to view its output.
          </div>
        ) : run.outputs && run.outputs[selectedNode.id] ? (
          <pre className="text-xs text-foreground bg-muted p-3 rounded overflow-auto font-mono whitespace-pre-wrap break-words">
            {typeof run.outputs[selectedNode.id] === 'string'
              ? run.outputs[selectedNode.id] as string
              : JSON.stringify(run.outputs[selectedNode.id], null, 2)}
          </pre>
        ) : (
          <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
            {nodeStatus === 'idle' ? 'Node was not executed.' : 'No output available.'}
          </div>
        )}
      </TabsContent>

      {/* Error tab */}
      {nodeRun?.error && (
        <TabsContent value="error" className="flex-1 min-h-0 overflow-auto mt-0 p-4">
          <div className="rounded-lg bg-destructive/5 border border-destructive/20 p-3">
            <p className="text-xs text-destructive font-mono whitespace-pre-wrap break-all">
              {nodeRun.error}
            </p>
          </div>
        </TabsContent>
      )}

      {/* Info tab */}
      <TabsContent value="info" className="flex-1 min-h-0 overflow-auto mt-0 p-4">
        {!selectedNode ? (
          <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
            Select a node to view execution info.
          </div>
        ) : nodeRun ? (
          <div className="space-y-3 text-xs">
            <InfoRow label="Node ID" value={selectedNode.id} mono />
            <InfoRow label="Type" value={selectedNode.data.nodeType} />
            <InfoRow label="Status" value={statusCfg.label} />
            <InfoRow label="Started" value={nodeRun.started_at ? new Date(nodeRun.started_at).toLocaleString() : '-'} />
            <InfoRow label="Completed" value={nodeRun.completed_at ? new Date(nodeRun.completed_at).toLocaleString() : '-'} />
            {nodeRun.started_at && nodeRun.completed_at && (
              <InfoRow
                label="Duration"
                value={`${((new Date(nodeRun.completed_at).getTime() - new Date(nodeRun.started_at).getTime()) / 1000).toFixed(1)}s`}
                mono
              />
            )}
            {nodeRun.retry_count > 0 && (
              <InfoRow label="Retries" value={String(nodeRun.retry_count)} />
            )}
          </div>
        ) : (
          <div className="flex items-center justify-center h-32 text-xs text-muted-foreground">
            Node was not executed in this run.
          </div>
        )}
      </TabsContent>
    </Tabs>
  )
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-start justify-between">
      <span className="text-muted-foreground">{label}</span>
      <span className={`text-foreground text-right ${mono ? 'font-mono' : ''}`}>{value}</span>
    </div>
  )
}
```

**Step 2: Type-check**

Run: `cd web && npx tsc --noEmit`
Expected: SUCCESS

**Step 3: Commit both Task 3 and Task 4**

```bash
git add web/src/pages/runs/RunViewer.tsx web/src/pages/runs/RunNodePanel.tsx web/src/app/router.tsx
git commit -m "feat: replace RunDetail with canvas-based RunViewer page"
```

---

### Task 5: Wire RunViewer exports and clean up old RunDetail references

**Files:**
- Modify: `web/src/app/router.tsx` (done in Task 3)
- Check: any other imports of `RunDetail` widget

**Step 1: Search for RunDetail references**

Run: `grep -r "RunDetail\|run-detail" web/src/ --include="*.ts" --include="*.tsx" -l`

Remove or update any remaining imports. The `web/src/widgets/run-detail/` directory can remain for now (no deletion needed since it might be referenced from other places).

**Step 2: Verify the full build**

Run: `cd web && npm run build`
Expected: SUCCESS

**Step 3: Backend build**

Run: `go build ./cmd/upal`
Expected: SUCCESS

**Step 4: Commit cleanup if needed**

```bash
git add -A
git commit -m "chore: wire RunViewer and clean up RunDetail references"
```

---

### Task 6: Update PublishInboxPreview run link

Now that RunRecords contain `workflow_definition`, the `/runs/:id` page will show the canvas. Update the link in PublishInboxPreview to always link (for `run-` prefixed IDs) since the new RunViewer page will properly display the workflow canvas.

**Files:**
- Verify: `web/src/pages/publish-inbox/PublishInboxPreview.tsx` — the conditional link logic from the earlier fix is still correct (`run-` prefix check).

**Step 1: Verify**

The link should already show correctly: `run-` prefixed IDs get a clickable link to `/runs/{id}`, old `session-` IDs show as plain text. No changes needed.

**Step 2: Manual test plan**

1. Start the dev server: `make dev-backend` + `make dev-frontend`
2. Navigate to `/runs` — click a run → should show the canvas-based RunViewer
3. If the run has `workflow_definition`, the canvas shows nodes with status colors
4. Click a node → right panel opens with Output/Error/Info tabs
5. Navigate to `/publish-inbox` → click a `run-` prefixed link → should open RunViewer
6. For a running workflow, node statuses should update live via SSE

**Step 3: Commit if any fixes needed**

---
