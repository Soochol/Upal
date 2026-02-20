package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// RunRequest is the JSON body for workflow execution.
// When Workflow is provided, it is used directly instead of looking up by name.
type RunRequest struct {
	Inputs   map[string]any           `json:"inputs"`
	Workflow *upal.WorkflowDefinition `json:"workflow,omitempty"`
}

// sendSSEError writes an SSE "done" event with a failure status and flushes.
func sendSSEError(w http.ResponseWriter, flusher http.Flusher, err error) {
	data, _ := json.Marshal(map[string]string{"status": "failed", "error": err.Error()})
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
}

// sendSSEDone writes an SSE "done" event with arbitrary payload data and flushes.
func sendSSEDone(w http.ResponseWriter, flusher http.Flusher, payload any) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
	flusher.Flush()
}

func (s *Server) runWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// 1. Parse request body.
	var req RunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req.Inputs = nil
		}
	}

	// 2. Resolve workflow: inline or lookup via service.
	var wf *upal.WorkflowDefinition
	if req.Workflow != nil {
		wf = req.Workflow
		wf.Name = name
	} else {
		var err error
		wf, err = s.workflowSvc.Lookup(r.Context(), name)
		if err != nil {
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}
	}

	// 3. Set SSE headers.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 4. Validate workflow.
	if err := s.workflowSvc.Validate(wf); err != nil {
		sendSSEError(w, flusher, err)
		return
	}

	// 5. Create run record if history service is available.
	var runID string
	if s.runHistorySvc != nil {
		record, err := s.runHistorySvc.StartRun(r.Context(), name, "manual", "", req.Inputs)
		if err != nil {
			slog.Warn("failed to create run record", "err", err)
		} else {
			runID = record.ID
		}
	}

	// 6. Execute via WorkflowService.
	events, result, err := s.workflowSvc.Run(r.Context(), wf, req.Inputs)
	if err != nil {
		if runID != "" {
			s.runHistorySvc.FailRun(r.Context(), runID, err.Error())
		}
		sendSSEError(w, flusher, err)
		return
	}

	// 7. Stream events as SSE and track node runs.
	for ev := range events {
		if ev.Type == "error" {
			if runID != "" {
				errMsg := fmt.Sprintf("%v", ev.Payload["error"])
				s.runHistorySvc.FailRun(r.Context(), runID, errMsg)
			}
			sendSSEError(w, flusher, fmt.Errorf("%v", ev.Payload["error"]))
			return
		}

		// Track node-level execution.
		if runID != "" {
			s.trackNodeRun(r.Context(), runID, ev)
		}

		data, _ := json.Marshal(ev.Payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data)
		flusher.Flush()
	}

	// 8. Send done event with final state.
	res := <-result

	if runID != "" {
		s.runHistorySvc.CompleteRun(r.Context(), runID, res.State)
	}

	sendSSEDone(w, flusher, map[string]any{
		"status":     "completed",
		"session_id": res.SessionID,
		"state":      res.State,
		"run_id":     runID,
	})
}

// trackNodeRun updates the run record with node-level execution status.
func (s *Server) trackNodeRun(ctx context.Context, runID string, ev services.WorkflowEvent) {
	if s.runHistorySvc == nil || ev.NodeID == "" {
		return
	}

	now := time.Now()

	switch ev.Type {
	case services.EventNodeStarted:
		s.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:    ev.NodeID,
			Status:    "running",
			StartedAt: now,
		})
	case services.EventNodeCompleted:
		s.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      "completed",
			StartedAt:   now,
			CompletedAt: &now,
		})
	}
}
