// internal/services/stage_workflow.go
package services

import (
	"context"
	"fmt"

	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

type WorkflowStageExecutor struct {
	workflowSvc ports.WorkflowExecutor
}

func NewWorkflowStageExecutor(workflowSvc ports.WorkflowExecutor) *WorkflowStageExecutor {
	return &WorkflowStageExecutor{workflowSvc: workflowSvc}
}

func (e *WorkflowStageExecutor) Type() string { return "workflow" }

func (e *WorkflowStageExecutor) Execute(ctx context.Context, _ *upal.Pipeline, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	wfName := stage.Config.WorkflowName
	if wfName == "" {
		return nil, fmt.Errorf("workflow_name is required for workflow stage")
	}

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
		inputs[destKey] = srcExpr
	}

	wf, err := e.workflowSvc.Lookup(ctx, wfName)
	if err != nil {
		return nil, fmt.Errorf("workflow %q not found: %w", wfName, err)
	}

	eventCh, resultCh, err := e.workflowSvc.Run(ctx, wf, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow %q: %w", wfName, err)
	}

	for range eventCh {
	}

	result := <-resultCh

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  upal.StageStatusCompleted,
		Output:  result.State,
	}, nil
}
