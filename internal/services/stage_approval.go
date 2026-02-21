// internal/services/stage_approval.go
package services

import (
	"context"
	"time"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/notify"
	"github.com/soochol/upal/internal/upal"
)

// ApprovalStageExecutor pauses the pipeline and waits for user approval.
// If connection_id is configured, sends an external notification before pausing.
// Resume logic (approve/reject) is handled by the API layer — not stored here.
type ApprovalStageExecutor struct {
	senderReg    *notify.SenderRegistry
	connResolver agents.ConnectionResolver
}

func NewApprovalStageExecutor(senderReg *notify.SenderRegistry, connResolver agents.ConnectionResolver) *ApprovalStageExecutor {
	return &ApprovalStageExecutor{senderReg: senderReg, connResolver: connResolver}
}

func (e *ApprovalStageExecutor) Type() string { return "approval" }

func (e *ApprovalStageExecutor) Execute(ctx context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	// Send external notification if connection is configured (best-effort).
	if stage.Config.ConnectionID != "" && e.senderReg != nil && e.connResolver != nil {
		if conn, err := e.connResolver.Resolve(ctx, stage.Config.ConnectionID); err == nil {
			if sender, err := e.senderReg.Get(conn.Type); err == nil {
				_ = sender.Send(ctx, conn, stage.Config.Message)
			}
		}
	}

	// Return a "waiting" result — the PipelineRunner will persist this
	// and the API layer handles resume via Approve/Reject endpoints.
	return &upal.StageResult{
		StageID:   stage.ID,
		Status:    "waiting",
		Output:    map[string]any{"message": stage.Config.Message},
		StartedAt: time.Now(),
	}, nil
}
