package services

import (
	"context"
	"fmt"
	"time"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/notify"
	"github.com/soochol/upal/internal/upal"
)

// NotificationStageExecutor sends a notification and completes immediately.
// Unlike ApprovalStageExecutor, it does not pause the pipeline.
type NotificationStageExecutor struct {
	senderReg    *notify.SenderRegistry
	connResolver agents.ConnectionResolver
}

func NewNotificationStageExecutor(senderReg *notify.SenderRegistry, connResolver agents.ConnectionResolver) *NotificationStageExecutor {
	return &NotificationStageExecutor{senderReg: senderReg, connResolver: connResolver}
}

func (e *NotificationStageExecutor) Type() string { return "notification" }

func (e *NotificationStageExecutor) Execute(ctx context.Context, _ *upal.Pipeline, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	fail := func(errMsg string) (*upal.StageResult, error) {
		now := time.Now()
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      upal.StageStatusFailed,
			Error:       errMsg,
			StartedAt:   now,
			CompletedAt: &now,
		}, fmt.Errorf("notification stage %q: %s", stage.ID, errMsg)
	}

	if stage.Config.ConnectionID == "" {
		return fail("connection_id is required")
	}
	if e.senderReg == nil || e.connResolver == nil {
		return fail("notification service not configured")
	}

	conn, err := e.connResolver.Resolve(ctx, stage.Config.ConnectionID)
	if err != nil {
		return fail(fmt.Sprintf("failed to resolve connection %q: %v", stage.Config.ConnectionID, err))
	}

	sender, err := e.senderReg.Get(conn.Type)
	if err != nil {
		return fail(fmt.Sprintf("no sender for connection type %q: %v", conn.Type, err))
	}

	if stage.Config.Subject != "" {
		if conn.Extras == nil {
			conn.Extras = map[string]any{}
		}
		conn.Extras["subject"] = stage.Config.Subject
	}

	msg := stage.Config.Message
	if msg == "" {
		msg = stage.Name
	}

	if err := sender.Send(ctx, conn, msg); err != nil {
		return fail(fmt.Sprintf("send failed: %v", err))
	}

	now := time.Now()
	return &upal.StageResult{
		StageID: stage.ID,
		Status:  upal.StageStatusCompleted,
		Output: map[string]any{
			"sent":    true,
			"channel": conn.Name,
			"type":    string(conn.Type),
		},
		StartedAt:   now,
		CompletedAt: &now,
	}, nil
}
