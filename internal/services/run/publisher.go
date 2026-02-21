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
	runHistorySvc *services.RunHistoryService
}

// NewRunPublisher creates a RunPublisher that drives background workflow executions.
func NewRunPublisher(
	workflowExec ports.WorkflowExecutor,
	runManager *services.RunManager,
	runHistorySvc *services.RunHistoryService,
) *RunPublisher {
	return &RunPublisher{
		workflowExec:  workflowExec,
		runManager:    runManager,
		runHistorySvc: runHistorySvc,
	}
}

// Launch starts background execution and publishes events to RunManager.
// Caller MUST call runManager.Register(runID) before calling Launch.
func (p *RunPublisher) Launch(ctx context.Context, runID string, wf *upal.WorkflowDefinition, inputs map[string]any) {
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
	for ev := range events {
		if ev.Type == upal.EventError {
			errMsg := fmt.Sprintf("%v", ev.Payload["error"])
			slog.Error("background run error", "run_id", runID, "err", errMsg)
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

		p.runManager.Append(runID, services.EventRecord{
			Type:    ev.Type,
			NodeID:  ev.NodeID,
			Payload: ev.Payload,
		})

		if p.runHistorySvc != nil {
			p.trackNodeRun(ctx, runID, ev)
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
func (p *RunPublisher) trackNodeRun(ctx context.Context, runID string, ev upal.WorkflowEvent) {
	if p.runHistorySvc == nil || ev.NodeID == "" {
		return
	}

	now := time.Now()

	switch ev.Type {
	case upal.EventNodeStarted:
		p.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:    ev.NodeID,
			Status:    "running",
			StartedAt: now,
		})
	case upal.EventNodeCompleted:
		p.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      "completed",
			StartedAt:   now,
			CompletedAt: &now,
		})
	}
}
