// internal/services/stage_approval.go
package services

import (
	"context"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// ApprovalStageExecutor pauses the pipeline and waits for user approval.
type ApprovalStageExecutor struct {
	waitHandles map[string]chan map[string]any // runID+stageID → response channel
}

func NewApprovalStageExecutor() *ApprovalStageExecutor {
	return &ApprovalStageExecutor{
		waitHandles: make(map[string]chan map[string]any),
	}
}

func (e *ApprovalStageExecutor) Type() string { return "approval" }

func (e *ApprovalStageExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	// Return a "waiting" result — the PipelineRunner will persist this
	// and the API layer handles resume via Approve/Reject endpoints.
	return &upal.StageResult{
		StageID:   stage.ID,
		Status:    "waiting",
		Output:    map[string]any{"message": stage.Config.Message},
		StartedAt: time.Now(),
	}, nil
}
