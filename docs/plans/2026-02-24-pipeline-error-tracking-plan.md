# Pipeline Error Tracking Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Surface pipeline workflow failure details and results in Publish Inbox — both inline and via Run detail page links.

**Architecture:** Add `ErrorMessage` + `FailedNodeID` to `WorkflowResult`. Integrate `ContentCollector` with `RunHistoryService` to create real `RunRecord`s for pipeline workflows. Frontend shows errors inline and links to Run detail page.

**Tech Stack:** Go backend (domain types, services, DB), React/TypeScript frontend (Publish Inbox components)

---

### Task 1: Add error fields to WorkflowResult (Backend domain type)

**Files:**
- Modify: `internal/upal/content.go:126-133`

**Step 1: Add ErrorMessage and FailedNodeID fields**

```go
// WorkflowResult tracks workflow execution results for the produce stage.
type WorkflowResult struct {
	WorkflowName string               `json:"workflow_name"`
	RunID        string               `json:"run_id"`
	Status       WorkflowResultStatus `json:"status"`
	ChannelID    string               `json:"channel_id,omitempty"`
	OutputURL    string               `json:"output_url,omitempty"`
	CompletedAt  *time.Time           `json:"completed_at,omitempty"`
	ErrorMessage string               `json:"error_message,omitempty"`
	FailedNodeID string               `json:"failed_node_id,omitempty"`
}
```

**Step 2: Run backend tests**

Run: `go test ./internal/... -v -race -count=1 -run TestContent`
Expected: PASS (JSON serialization is automatic, no schema migration needed — `workflow_results` is stored as JSONB)

**Step 3: Commit**

```bash
git add internal/upal/content.go
git commit -m "feat: add error tracking fields to WorkflowResult"
```

---

### Task 2: Inject RunHistoryService into ContentCollector

**Files:**
- Modify: `internal/services/content_collector.go:27-57`
- Modify: `cmd/upal/main.go` (wire new dependency)

**Step 1: Add runHistorySvc field to ContentCollector**

In `content_collector.go`, add `runHistorySvc ports.RunHistoryPort` to the struct and constructor:

```go
type ContentCollector struct {
	contentSvc   *ContentSessionService
	collectExec  *CollectStageExecutor
	workflowSvc  *WorkflowService
	workflowRepo repository.WorkflowRepository
	pipelineRepo repository.PipelineRepository
	resolver     ports.LLMResolver
	generator    ports.WorkflowGenerator
	skills       skills.Provider
	runHistorySvc ports.RunHistoryPort
}
```

Update `NewContentCollector` to accept and store `runHistorySvc ports.RunHistoryPort`.

**Step 2: Wire in main.go**

Find where `NewContentCollector` is called in `cmd/upal/main.go` and pass the existing `runHistorySvc` instance.

**Step 3: Run tests**

Run: `go build ./cmd/upal && go test ./internal/services/... -v -race -count=1`
Expected: PASS (compile + existing tests still pass)

**Step 4: Commit**

```bash
git add internal/services/content_collector.go cmd/upal/main.go
git commit -m "feat: inject RunHistoryService into ContentCollector"
```

---

### Task 3: Create RunRecords for pipeline-triggered workflows

**Files:**
- Modify: `internal/services/content_collector.go:468-579` (ProduceWorkflows method)

**Step 1: Modify ProduceWorkflows to create and manage RunRecords**

Replace the workflow execution section inside the `g.Go(func() error { ... })` block. For each workflow:

1. Call `c.runHistorySvc.StartRun()` before execution to create a RunRecord
2. Store the real RunRecord ID in `WorkflowResult.RunID`
3. Drain events and track node runs (similar to RunPublisher logic)
4. On success: call `CompleteRun()` with outputs from result state
5. On failure: call `FailRun()` with error message, then read the RunRecord to find the failed node
6. Set `ErrorMessage` and `FailedNodeID` on the WorkflowResult

```go
g.Go(func() error {
	updateResult(i, func(r *upal.WorkflowResult) { r.Status = upal.WFResultRunning })

	// Look up workflow definition.
	wf, err := c.workflowRepo.Get(gCtx, req.Name)
	if err != nil {
		log.Printf("content_collector: workflow %q not found: %v", req.Name, err)
		updateResult(i, func(r *upal.WorkflowResult) {
			r.Status = upal.WFResultFailed
			r.ErrorMessage = fmt.Sprintf("Workflow %q not found", req.Name)
		})
		return nil
	}

	// Build inputs mapped to actual input node IDs.
	inputs := buildProductionInputs(detail, wf)

	// Create RunRecord for tracking.
	var runID string
	if c.runHistorySvc != nil {
		record, err := c.runHistorySvc.StartRun(gCtx, req.Name, "pipeline", sessionID, inputs)
		if err != nil {
			log.Printf("content_collector: failed to create run record for %q: %v", req.Name, err)
		} else {
			runID = record.ID
			updateResult(i, func(r *upal.WorkflowResult) { r.RunID = runID })
		}
	}

	// Run the workflow.
	eventCh, resultCh, err := c.workflowSvc.Run(gCtx, wf, inputs)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to start: %s", err.Error())
		log.Printf("content_collector: failed to run workflow %q: %v", req.Name, err)
		if runID != "" && c.runHistorySvc != nil {
			c.runHistorySvc.FailRun(gCtx, runID, errMsg)
		}
		updateResult(i, func(r *upal.WorkflowResult) {
			r.Status = upal.WFResultFailed
			r.ErrorMessage = errMsg
		})
		return nil
	}

	// Drain event channel, capturing errors and tracking node runs.
	var runErr string
	for evt := range eventCh {
		if evt.Type == "error" {
			if errMsg, ok := evt.Payload["error"].(string); ok {
				runErr = errMsg
			}
		}
		// Track node-level runs in RunRecord.
		if runID != "" && c.runHistorySvc != nil {
			trackNodeRunFromEvent(gCtx, c.runHistorySvc, runID, evt)
		}
	}

	if runErr != "" {
		log.Printf("content_collector: workflow %q execution error: %s", req.Name, runErr)
		if runID != "" && c.runHistorySvc != nil {
			c.runHistorySvc.FailRun(gCtx, runID, runErr)
		}
		// Find failed node from RunRecord.
		failedNode := ""
		if runID != "" && c.runHistorySvc != nil {
			if rec, err := c.runHistorySvc.GetRun(gCtx, runID); err == nil {
				for _, nr := range rec.NodeRuns {
					if nr.Status == upal.NodeRunError {
						failedNode = nr.NodeID
						break
					}
				}
			}
		}
		updateResult(i, func(r *upal.WorkflowResult) {
			r.Status = upal.WFResultFailed
			r.ErrorMessage = runErr
			r.FailedNodeID = failedNode
		})
		return nil
	}

	// Wait for result.
	runResult, ok := <-resultCh
	if !ok {
		errMsg := "Result channel closed unexpectedly"
		log.Printf("content_collector: workflow %q %s", req.Name, errMsg)
		if runID != "" && c.runHistorySvc != nil {
			c.runHistorySvc.FailRun(gCtx, runID, errMsg)
		}
		updateResult(i, func(r *upal.WorkflowResult) {
			r.Status = upal.WFResultFailed
			r.ErrorMessage = errMsg
		})
		return nil
	}

	// Success path.
	if runID != "" && c.runHistorySvc != nil {
		c.runHistorySvc.CompleteRun(gCtx, runID, runResult.State)
	}

	now := time.Now()
	updateResult(i, func(r *upal.WorkflowResult) {
		r.Status = upal.WFResultSuccess
		r.RunID = runID
		r.CompletedAt = &now
	})
	return nil
})
```

**Step 2: Add trackNodeRunFromEvent helper**

Add a package-level helper function (similar to RunPublisher.trackNodeRun but standalone):

```go
// trackNodeRunFromEvent records node-level execution status from workflow events.
func trackNodeRunFromEvent(ctx context.Context, svc ports.RunHistoryPort, runID string, ev upal.WorkflowEvent) {
	if ev.NodeID == "" {
		return
	}
	now := time.Now()
	switch ev.Type {
	case upal.EventNodeStarted:
		svc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:    ev.NodeID,
			Status:    upal.NodeRunRunning,
			StartedAt: now,
		})
	case upal.EventNodeCompleted:
		svc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      upal.NodeRunCompleted,
			StartedAt:   now,
			CompletedAt: &now,
		})
	case upal.EventError:
		if ev.NodeID != "" {
			svc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
				NodeID:      ev.NodeID,
				Status:      upal.NodeRunError,
				StartedAt:   now,
				CompletedAt: &now,
			})
		}
	}
}
```

**Step 3: Run tests**

Run: `go build ./cmd/upal && go test ./internal/services/... -v -race -count=1`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/services/content_collector.go
git commit -m "feat: create RunRecords for pipeline-triggered workflows with error capture"
```

---

### Task 4: Add error fields to frontend WorkflowResult type

**Files:**
- Modify: `web/src/entities/content-session/types.ts:42-49`
- Modify: `web/src/shared/types/index.ts` (if WorkflowResult is also defined there — check)

**Step 1: Add error_message and failed_node_id to WorkflowResult type**

In `web/src/entities/content-session/types.ts`:

```typescript
export type WorkflowResult = {
  workflow_name: string
  run_id: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'published' | 'rejected'
  output_url?: string
  completed_at?: string
  channel_id?: string
  error_message?: string
  failed_node_id?: string
}
```

**Step 2: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 3: Commit**

```bash
git add web/src/entities/content-session/types.ts
git commit -m "feat: add error tracking fields to frontend WorkflowResult type"
```

---

### Task 5: Show error details inline in PublishInboxPreview

**Files:**
- Modify: `web/src/pages/publish-inbox/PublishInboxPreview.tsx:118-206` (WorkflowResultCard)

**Step 1: Add collapsible error section to failed cards**

Inside `WorkflowResultCard`, after the content area div and before the actions div, add an error section for failed workflows:

```tsx
{/* Error details for failed workflows */}
{result.status === 'failed' && result.error_message && (
    <div className="px-5 pb-4">
        <div className="rounded-lg bg-destructive/5 border border-destructive/20 p-3">
            <p className="text-xs text-destructive font-medium mb-1">
                {result.failed_node_id
                    ? `Failed at node: ${result.failed_node_id}`
                    : 'Execution failed'}
            </p>
            <p className="text-xs text-destructive/80 font-mono whitespace-pre-wrap break-all">
                {result.error_message}
            </p>
        </div>
    </div>
)}
```

**Step 2: Make Run ID a clickable link to Runs page**

In the content area, replace the static run ID text with a link:

```tsx
{result.run_id && (
    <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <a href={`/runs?run=${result.run_id}`}
            className="hover:text-foreground hover:underline transition-colors">
            Run: {result.run_id.slice(0, 12)}
        </a>
        {result.completed_at && (
            <span>Completed: {new Date(result.completed_at).toLocaleString()}</span>
        )}
        {result.output_url && (
            <a href={result.output_url} target="_blank" rel="noopener noreferrer"
                className="flex items-center gap-1 text-primary hover:underline">
                <ExternalLink className="h-3 w-3" /> Preview
            </a>
        )}
    </div>
)}
```

**Step 3: Run type check**

Run: `cd web && npx tsc -b --noEmit`
Expected: PASS

**Step 4: Commit**

```bash
git add web/src/pages/publish-inbox/PublishInboxPreview.tsx
git commit -m "feat: show error details and run links in Publish Inbox workflow cards"
```

---

### Task 6: Verify end-to-end

**Step 1: Start dev environment**

Run: `make dev-backend` (terminal 1) and `make dev-frontend` (terminal 2)

**Step 2: Test failed workflow**

Trigger a pipeline with a workflow that will fail. Verify:
- Error message shows inline in the Publish Inbox card
- Failed node ID shows if applicable
- Run ID links to the Runs page with full detail

**Step 3: Test successful workflow**

Trigger a pipeline with a working workflow. Verify:
- Run ID is clickable and navigates to Runs detail
- RunRecord shows in the Runs page with node timeline

**Step 4: Final commit if any fixes needed**
