# Run Canvas Viewer Design

## Goal

Replace the current Run detail page (`/runs/:id`) with a **read-only workflow canvas view** that shows the workflow as it was at execution time, with per-node status visualization and a right panel for inspecting outputs/errors.

## Problem

Current Run detail page shows a linear timeline — users can't visually see *where* in the workflow graph something failed or what each node produced. The workflow editor canvas is a much more intuitive way to understand execution flow.

Additionally, pipeline-triggered runs don't store the workflow definition, so there's no way to reconstruct the canvas for past runs if the workflow is later modified or deleted.

## Design

### Backend: Workflow Snapshot in RunRecord

**RunRecord gets a `WorkflowDef` field** (`*WorkflowDefinition`):
- Stored as JSONB in the `runs` table (`workflow_definition` column)
- Populated at `StartRun()` time — caller passes the definition
- Each Run is a fully independent snapshot
- Typical size: 2-10 KB per workflow definition

**Changes:**
- `RunRecord` struct: add `WorkflowDef *WorkflowDefinition` field
- `StartRun()`: accept `WorkflowDefinition` parameter, store it
- DB `CreateRun` / `GetRun` / `UpdateRun`: handle new JSONB column
- API `GET /api/runs/:id`: include `workflow_definition` in response
- All callers of `StartRun()`: pass workflow definition (RunPublisher, ContentCollector)

### Frontend: Read-Only Canvas Page

**Reuse existing components:**
- `<Canvas readOnly={true}>` — already supports read-only mode (no drag, connect, delete, palette, prompt bar)
- Right panel — new read-only variant with Output/Error/Info tabs instead of Properties

**Page layout** (same 3-column structure as editor):
- Header: Run ID, status badge, workflow name, "Open Workflow" link, retry button
- Center: Read-only canvas with node status visualization
- Right: Run-specific panel (output, error, info per node)

**Node status visualization:**
- Overlay execution status on each node using `NodeRunRecord` data
- Colors: success=green border, failed=red border, running=blue+spinner, pending=default
- Status badge on each node showing the state

**Data flow:**
1. Fetch `RunRecord` from `GET /api/runs/:id` (includes `workflow_definition`)
2. Deserialize `workflow_definition` → React Flow nodes/edges
3. Map `node_runs` status onto deserialized nodes
4. For running runs: connect to SSE stream for live updates

### Right Panel (Run Mode)

When a node is clicked, show:
- **Output tab**: Node execution output (from `NodeRunRecord` or SSE events)
- **Error tab**: Error message if node failed
- **Info tab**: Started/completed timestamps, duration, token usage

### Live Run Support

- Running/pending runs: use existing `connectToRunEvents()` SSE infrastructure
- Node statuses update in real-time as events arrive
- Completed runs: load all status from `RunRecord.node_runs` in one shot

### Header

- Run ID (truncated), status badge with color
- Workflow name
- "Open Workflow" button → navigate to `/workflows?w={name}`
- "Retry" button (visible only for failed runs)
- Timestamps: started, completed, duration

## Non-Goals

- Editing the workflow from the run view
- Replaying execution step-by-step
- Comparing two runs side-by-side
