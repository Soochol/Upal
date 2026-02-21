// internal/services/stage_workflow.go
package services

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

// WorkflowStageExecutor runs a workflow by name.
type WorkflowStageExecutor struct {
	workflowSvc *WorkflowService
}

func NewWorkflowStageExecutor(workflowSvc *WorkflowService) *WorkflowStageExecutor {
	return &WorkflowStageExecutor{workflowSvc: workflowSvc}
}

func (e *WorkflowStageExecutor) Type() string { return "workflow" }

func (e *WorkflowStageExecutor) Execute(ctx context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	wfName := stage.Config.WorkflowName
	if wfName == "" {
		return nil, fmt.Errorf("workflow_name is required for workflow stage")
	}

	// Build inputs from input mapping.
	// For each mapping entry: if srcExpr matches a key in the previous stage's
	// output, use that value (dynamic reference). Otherwise treat srcExpr as a
	// static literal value entered at pipeline design time.
	inputs := make(map[string]any)
	for destKey, srcExpr := range stage.Config.InputMapping {
		if srcExpr == "" {
			continue
		}
		if prevResult != nil {
			if val, ok := prevResult.Output[srcExpr]; ok {
				inputs[destKey] = val
				continue
			}
		}
		// Static value
		inputs[destKey] = srcExpr
	}

	// Look up and run the workflow
	wf, err := e.workflowSvc.Lookup(ctx, wfName)
	if err != nil {
		return nil, fmt.Errorf("workflow %q not found: %w", wfName, err)
	}

	eventCh, resultCh, err := e.workflowSvc.Run(ctx, wf, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow %q: %w", wfName, err)
	}

	// Drain events
	for range eventCh {
	}

	result := <-resultCh

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output:  result.State,
	}, nil
}
