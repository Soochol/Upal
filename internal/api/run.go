package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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

// runWorkflow starts a workflow execution in the background and returns the
// run ID immediately. Clients connect to GET /api/runs/{id}/events to stream
// execution events via SSE.
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

	// 3. Validate workflow.
	if err := s.workflowSvc.Validate(wf); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 4. Create run record if history service is available.
	var runID string
	if s.runHistorySvc != nil {
		record, err := s.runHistorySvc.StartRun(r.Context(), name, "manual", "", req.Inputs)
		if err != nil {
			slog.Warn("failed to create run record", "err", err)
		} else {
			runID = record.ID
		}
	}

	// 5. Register in RunManager and launch background execution.
	if s.runManager != nil && runID != "" {
		s.runManager.Register(runID)
		go s.executeRunBackground(runID, wf, req.Inputs)
	}

	// 6. Return 202 Accepted with run ID.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"run_id": runID})
}

// executeRunBackground runs a workflow with a detached context and publishes
// events to the RunManager. This goroutine is not tied to any HTTP request.
func (s *Server) executeRunBackground(runID string, wf *upal.WorkflowDefinition, inputs map[string]any) {
	ctx := context.Background()

	events, result, err := s.workflowSvc.Run(ctx, wf, inputs)
	if err != nil {
		slog.Error("background run failed to start", "run_id", runID, "err", err)
		if s.runHistorySvc != nil {
			s.runHistorySvc.FailRun(ctx, runID, err.Error())
		}
		s.runManager.Fail(runID, err.Error())
		return
	}

	// Stream events into the RunManager buffer.
	for ev := range events {
		if ev.Type == "error" {
			errMsg := fmt.Sprintf("%v", ev.Payload["error"])
			slog.Error("background run error", "run_id", runID, "err", errMsg)
			if s.runHistorySvc != nil {
				s.runHistorySvc.FailRun(ctx, runID, errMsg)
			}
			s.runManager.Fail(runID, errMsg)
			return
		}

		// Inject server timestamp into node_started events so reconnecting
		// clients can restore accurate elapsed timers.
		if ev.Type == upal.EventNodeStarted {
			ev.Payload["started_at"] = time.Now().UnixMilli()
		}

		s.runManager.Append(runID, services.EventRecord{
			Type:    ev.Type,
			NodeID:  ev.NodeID,
			Payload: ev.Payload,
		})

		if s.runHistorySvc != nil {
			s.trackNodeRun(ctx, runID, ev)
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

	if s.runHistorySvc != nil {
		s.runHistorySvc.CompleteRun(ctx, runID, res.State)
	}
	s.runManager.Complete(runID, donePayload)
}

// streamRunEvents streams execution events for a run via SSE.
// Supports initial connection (replays all buffered events) and reconnection
// (replays from Last-Event-ID onward). The run continues in the background
// regardless of client connection state.
func (s *Server) streamRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "id")

	// Parse Last-Event-ID for reconnection support.
	lastSeq := -1
	if idStr := r.Header.Get("Last-Event-ID"); idStr != "" {
		if n, err := strconv.Atoi(idStr); err == nil {
			lastSeq = n
		}
	}
	startSeq := lastSeq + 1

	// Check if RunManager tracks this run.
	if s.runManager == nil {
		http.Error(w, "run streaming not available", http.StatusServiceUnavailable)
		return
	}

	events, notify, done, donePayload, found := s.runManager.Subscribe(runID, startSeq)
	if !found {
		// Fallback: check run history for completed runs.
		if s.runHistorySvc != nil {
			record, err := s.runHistorySvc.GetRun(r.Context(), runID)
			if err == nil {
				s.sendSyntheticDone(w, record)
				return
			}
		}
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	// Set SSE headers.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Replay buffered events.
	for _, ev := range events {
		writeSSEEvent(w, ev)
	}
	flusher.Flush()

	// If the run is already done, send the done event and return.
	if done {
		writeDoneEvent(w, donePayload)
		flusher.Flush()
		return
	}

	// Live-stream: wait for new events or client disconnect.
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected — run continues in background.
			return
		case <-notify:
			// New events available. Re-subscribe to get the latest snapshot.
			nextSeq := startSeq + len(events)
			events, notify, done, donePayload, found = s.runManager.Subscribe(runID, nextSeq)
			if !found {
				return
			}
			startSeq = nextSeq

			for _, ev := range events {
				writeSSEEvent(w, ev)
			}
			flusher.Flush()

			if done {
				writeDoneEvent(w, donePayload)
				flusher.Flush()
				return
			}
		}
	}
}

// writeSSEEvent writes a single event as an SSE frame with the seq as the id.
func writeSSEEvent(w http.ResponseWriter, ev services.EventRecord) {
	data, _ := json.Marshal(ev.Payload)
	fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.Seq, ev.Type, data)
}

// writeDoneEvent writes the final "done" SSE event.
func writeDoneEvent(w http.ResponseWriter, payload map[string]any) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
}

// sendSyntheticDone sends a minimal SSE stream with a synthetic done event
// from a completed run record (for runs whose buffer has already been GC'd).
func (s *Server) sendSyntheticDone(w http.ResponseWriter, record *upal.RunRecord) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	payload := map[string]any{
		"status": string(record.Status),
		"state":  record.Outputs,
		"run_id": record.ID,
	}
	if record.Error != nil {
		payload["error"] = *record.Error
	}
	writeDoneEvent(w, payload)
	flusher.Flush()
}

// trackNodeRun updates the run record with node-level execution status.
func (s *Server) trackNodeRun(ctx context.Context, runID string, ev upal.WorkflowEvent) {
	if s.runHistorySvc == nil || ev.NodeID == "" {
		return
	}

	now := time.Now()

	switch ev.Type {
	case upal.EventNodeStarted:
		s.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:    ev.NodeID,
			Status:    "running",
			StartedAt: now,
		})
	case upal.EventNodeCompleted:
		s.runHistorySvc.UpdateNodeRun(ctx, runID, upal.NodeRunRecord{
			NodeID:      ev.NodeID,
			Status:      "completed",
			StartedAt:   now,
			CompletedAt: &now,
		})
	}
}
