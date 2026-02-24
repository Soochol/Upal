# Pipeline Error Tracking & Result Viewing Design

## Problem

Pipeline-triggered workflows fail silently:
- **Failed sessions**: No error message, no info about which node failed — just "Failed" status
- **Successful sessions**: No way to view the actual output — only `output_url` if present

Root cause: `ContentCollector.ProduceWorkflows()` calls `workflowSvc.Run()` directly, bypassing `RunPublisher` and `RunHistoryService`. Errors are logged to console and discarded.

## Solution

### Backend

1. **Add error fields to `WorkflowResult`**: `ErrorMessage`, `FailedNodeID`
2. **Integrate ContentCollector with RunHistoryService**: Create RunRecords for pipeline-triggered workflows, enabling full node-level tracking
3. **Capture errors from event stream and RunRecord**: After execution, pull error/node data from RunRecord into WorkflowResult

### Frontend

1. **Inline error display**: Failed workflow cards show error message and failed node in a collapsible section
2. **Run link**: All WorkflowResult cards link to the existing Runs detail page via RunRecord ID
3. **Inline output preview**: Success cards show final output text from RunRecord.outputs

## Data Flow

```
ContentCollector.ProduceWorkflows()
  → RunHistoryService.StartRun() → RunRecord created
  → workflowSvc.Run() → eventCh, resultCh
  → Drain events, capture errors (existing logic)
  → On success: RunHistoryService.CompleteRun() + read outputs
  → On failure: RunHistoryService.FailRun() + read RunRecord for failed node
  → Update WorkflowResult with RunID (now real RunRecord ID), ErrorMessage, FailedNodeID
```
