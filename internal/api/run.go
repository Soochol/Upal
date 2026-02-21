package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

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
	if s.runManager != nil && s.runPublisher != nil && runID != "" {
		s.runManager.Register(runID)
		go s.runPublisher.Launch(context.Background(), runID, wf, req.Inputs)
	}

	// 6. Return 202 Accepted with run ID.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"run_id": runID})
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

