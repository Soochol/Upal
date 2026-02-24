package run

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
)

// RunPublisher bridges WorkflowExecutor.Run() into RunManager + RunHistoryService.
// It owns the background execution loop and event publishing logic.
type RunPublisher struct {
	workflowExec  ports.WorkflowExecutor
	runManager    *services.RunManager
	runHistorySvc ports.RunHistoryPort
	executionReg  *services.ExecutionRegistry
}

// NewRunPublisher creates a RunPublisher that drives background workflow executions.
func NewRunPublisher(
	workflowExec ports.WorkflowExecutor,
	runManager *services.RunManager,
	runHistorySvc ports.RunHistoryPort,
	executionReg *services.ExecutionRegistry,
) *RunPublisher {
	return &RunPublisher{
		workflowExec:  workflowExec,
		runManager:    runManager,
		runHistorySvc: runHistorySvc,
		executionReg:  executionReg,
	}
}

// Launch starts background execution and publishes events to RunManager.
// Caller MUST call runManager.Register(runID) before calling Launch.
func (p *RunPublisher) Launch(ctx context.Context, runID string, wf *upal.WorkflowDefinition, inputs map[string]any) {
	if p.executionReg != nil {
		p.executionReg.Register(runID)
		defer p.executionReg.Unregister(runID)
	}

	events, result, err := p.workflowExec.Run(ctx, wf, inputs)
	if err != nil {
		slog.Error("background run failed to start", "run_id", runID, "err", err)
		if p.runHistorySvc != nil {
			p.runHistorySvc.FailRun(ctx, runID, err.Error())
		}
		p.runManager.Fail(runID, err.Error())
		return
	}

	// Stream events into the RunManager buffer.
	var totalUsage upal.TokenUsage
	for ev := range events {
		if ev.Type == upal.EventError {
			errMsg := fmt.Sprintf("%v", ev.Payload["error"])
			slog.Error("background run error", "run_id", runID, "err", errMsg)
			p.runManager.Append(runID, services.EventRecord{
				WorkflowEvent: ev,
			})
			if p.runHistorySvc != nil {
				p.runHistorySvc.FailRun(ctx, runID, errMsg)
			}
			p.runManager.Fail(runID, errMsg)
			return
		}

		// Inject server timestamp into node_started events so reconnecting
		// clients can restore accurate elapsed timers.
		if ev.Type == upal.EventNodeStarted {
			ev.Payload["started_at"] = time.Now().UnixMilli()
		}

		if ev.Type == upal.EventNodeCompleted {
			ev.Payload["completed_at"] = time.Now().UnixMilli()
		}

		p.runManager.Append(runID, services.EventRecord{
			WorkflowEvent: ev,
		})

		if p.runHistorySvc != nil {
			nodeUsage := p.trackNodeRun(ctx, runID, ev)
			if nodeUsage != nil {
				totalUsage.PromptTokens += nodeUsage.PromptTokens
				totalUsage.CompletionTokens += nodeUsage.CompletionTokens
				totalUsage.TotalTokens += nodeUsage.TotalTokens
			}
		}
	}

	// Collect final result.
	res := <-result

	donePayload := map[string]any{
		"status":     "completed",
		"session_id": res.SessionID,
		"state":      res.State,
		"run_id":     runID,
	}

	if p.runHistorySvc != nil {
		p.runHistorySvc.CompleteRun(ctx, runID, res.State)
	}
	p.runManager.Complete(runID, donePayload)
}

// trackNodeRun updates the run record with node-level execution status.
// Returns token usage if the event is a completed node with usage data.
func (p *RunPublisher) trackNodeRun(ctx context.Context, runID string, ev upal.WorkflowEvent) *upal.TokenUsage {
	if p.runHistorySvc == nil || ev.NodeID == "" {
		return nil
	}

	now := time.Now()

	switch ev.Type {
	case upal.EventNodeStarted:
		p.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:    ev.NodeID,
			Status:    upal.NodeRunRunning,
			StartedAt: now,
		})
	case upal.EventNodeCompleted:
		var usage *upal.TokenUsage
		if tokens, ok := ev.Payload["tokens"].(map[string]any); ok {
			usage = &upal.TokenUsage{
				PromptTokens:     int32(toInt(tokens["prompt_token_count"])),
				CompletionTokens: int32(toInt(tokens["candidates_token_count"])),
				TotalTokens:      int32(toInt(tokens["total_token_count"])),
			}
		}
		p.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      upal.NodeRunCompleted,
			StartedAt:   now,
			CompletedAt: &now,
			Usage:       usage,
		})
		return usage
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	default:
		return 0
	}
}
