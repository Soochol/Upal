// internal/services/stage_notification.go
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
// Unlike ApprovalStageExecutor, it does not pause the pipeline — the run
// continues to the next stage after the message is delivered.
type NotificationStageExecutor struct {
	senderReg    *notify.SenderRegistry
	connResolver agents.ConnectionResolver
}

func NewNotificationStageExecutor(senderReg *notify.SenderRegistry, connResolver agents.ConnectionResolver) *NotificationStageExecutor {
	return &NotificationStageExecutor{senderReg: senderReg, connResolver: connResolver}
}

func (e *NotificationStageExecutor) Type() string { return "notification" }

func (e *NotificationStageExecutor) Execute(ctx context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	now := time.Now()
	completedAt := now

	if stage.Config.ConnectionID == "" {
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      "failed",
			Error:       "notification stage requires a connection_id",
			StartedAt:   now,
			CompletedAt: &completedAt,
		}, fmt.Errorf("notification stage %q: connection_id is required", stage.ID)
	}

	if e.senderReg == nil || e.connResolver == nil {
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      "failed",
			Error:       "notification service not configured",
			StartedAt:   now,
			CompletedAt: &completedAt,
		}, fmt.Errorf("notification stage %q: sender registry or connection resolver is nil", stage.ID)
	}

	conn, err := e.connResolver.Resolve(ctx, stage.Config.ConnectionID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to resolve connection %q: %v", stage.Config.ConnectionID, err)
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      "failed",
			Error:       errMsg,
			StartedAt:   now,
			CompletedAt: &completedAt,
		}, fmt.Errorf("notification stage %q: %s", stage.ID, errMsg)
	}

	sender, err := e.senderReg.Get(conn.Type)
	if err != nil {
		errMsg := fmt.Sprintf("no sender for connection type %q: %v", conn.Type, err)
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      "failed",
			Error:       errMsg,
			StartedAt:   now,
			CompletedAt: &completedAt,
		}, fmt.Errorf("notification stage %q: %s", stage.ID, errMsg)
	}

	// For SMTP connections, allow a per-stage subject override by temporarily
	// injecting it into conn.Extras before sending.
	if stage.Config.Subject != "" && conn.Extras == nil {
		conn.Extras = map[string]any{}
	}
	if stage.Config.Subject != "" {
		conn.Extras["subject"] = stage.Config.Subject
	}

	msg := stage.Config.Message
	if msg == "" {
		msg = stage.Name
	}

	if err := sender.Send(ctx, conn, msg); err != nil {
		errMsg := fmt.Sprintf("send failed: %v", err)
		return &upal.StageResult{
			StageID:     stage.ID,
			Status:      "failed",
			Error:       errMsg,
			StartedAt:   now,
			CompletedAt: &completedAt,
		}, fmt.Errorf("notification stage %q: %s", stage.ID, errMsg)
	}

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output: map[string]any{
			"sent":    true,
			"channel": conn.Name,
			"type":    string(conn.Type),
		},
		StartedAt:   now,
		CompletedAt: &completedAt,
	}, nil
}
