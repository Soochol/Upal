package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

type RunRequest struct {
	Inputs   map[string]any           `json:"inputs"`
	Workflow *upal.WorkflowDefinition `json:"workflow,omitempty"`
}

func (s *Server) runWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var req RunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req.Inputs = nil
		}
	}

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

	if err := s.workflowSvc.Validate(wf); err != nil {
		writeJSONStatus(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var runID string
	if s.runHistorySvc != nil {
		record, err := s.runHistorySvc.StartRun(r.Context(), name, "manual", "", req.Inputs, wf)
		if err != nil {
			slog.Warn("failed to create run record", "err", err)
		} else {
			runID = record.ID
		}
	}

	if s.runManager != nil && s.runPublisher != nil && runID != "" {
		s.runManager.Register(runID)
		go s.runPublisher.Launch(context.Background(), runID, wf, req.Inputs)
	}

	writeJSONStatus(w, http.StatusAccepted, map[string]string{"run_id": runID})
}

// streamRunEvents streams execution events for a run via SSE.
// Supports reconnection via Last-Event-ID header.
func (s *Server) streamRunEvents(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "id")

	lastSeq := -1
	if idStr := r.Header.Get("Last-Event-ID"); idStr != "" {
		if n, err := strconv.Atoi(idStr); err == nil {
			lastSeq = n
		}
	}
	startSeq := lastSeq + 1

	if s.runManager == nil {
		http.Error(w, "run streaming not available", http.StatusServiceUnavailable)
		return
	}

	events, notify, done, donePayload, found := s.runManager.Subscribe(runID, startSeq)
	if !found {
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, ev := range events {
		writeSSEEvent(w, ev)
	}
	flusher.Flush()

	if done {
		writeDoneEvent(w, donePayload)
		flusher.Flush()
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-notify:
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

func writeSSEEvent(w http.ResponseWriter, ev upal.EventRecord) {
	data, _ := json.Marshal(ev.Payload)
	fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.Seq, ev.Type, data)
}

func writeDoneEvent(w http.ResponseWriter, payload map[string]any) {
	data, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", data)
}

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
