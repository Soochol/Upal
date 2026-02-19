package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/engine"
)

// RunRequest is the JSON body for workflow execution.
type RunRequest struct {
	Inputs map[string]any `json:"inputs"`
}

func (s *Server) runWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// 1. Look up workflow (in-memory first, then DB fallback)
	wf, ok := s.workflows.Get(name)
	if !ok && s.db != nil {
		row, err := s.db.GetWorkflow(r.Context(), name)
		if err == nil {
			wf = &row.Definition
			s.workflows.Put(wf) // cache in memory for the run
			ok = true
		}
	}
	if !ok {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	// 2. Parse RunRequest from body
	var req RunRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Allow empty body (no inputs)
			req.Inputs = nil
		}
	}

	// 3. Set SSE headers
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 4. Subscribe to eventBus for this workflow's events.
	// We use a channel to receive events from the event bus handler.
	eventCh := make(chan engine.Event, 64)

	// We don't know the session ID yet, so we capture all events for this
	// workflow name initially, then filter by session ID once the runner starts.
	var sessionID string
	unsub := s.eventBus.Subscribe(func(e engine.Event) {
		if e.WorkflowID == name {
			select {
			case eventCh <- e:
			default:
				// Drop event if channel is full to avoid blocking the publisher.
			}
		}
	})
	defer unsub()

	// 5. Start runner.Run in a goroutine
	type runResult struct {
		session *engine.Session
		err     error
	}
	doneCh := make(chan runResult, 1)
	go func() {
		if s.a2aRunner != nil {
			baseURL := getBaseURL(r)
			nodeURLs := make(map[string]string)
			for _, n := range wf.Nodes {
				if n.Type == engine.NodeTypeExternal {
					if url, ok := n.Config["endpoint_url"].(string); ok {
						nodeURLs[n.ID] = url
						continue
					}
				}
				nodeURLs[n.ID] = fmt.Sprintf("%s/a2a/nodes/%s", baseURL, n.ID)
			}
			sess, err := s.a2aRunner.Run(r.Context(), wf, nodeURLs, req.Inputs)
			doneCh <- runResult{session: sess, err: err}
		} else {
			sess, err := s.runner.Run(r.Context(), wf, s.executors, req.Inputs)
			doneCh <- runResult{session: sess, err: err}
		}
	}()

	// 6. Stream events via SSE until done
	writeSSE := func(eventType string, data any) {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonData)
		flusher.Flush()
	}

	for {
		select {
		case evt := <-eventCh:
			// Capture the session ID from the first event
			if sessionID == "" && evt.SessionID != "" {
				sessionID = evt.SessionID
			}
			// Filter: only forward events for our session
			if sessionID != "" && evt.SessionID != sessionID {
				continue
			}
			writeSSE(string(evt.Type), map[string]any{
				"node_id":   evt.NodeID,
				"payload":   evt.Payload,
				"timestamp": evt.Timestamp,
			})
		case result := <-doneCh:
			// Drain remaining events from the channel
			for {
				select {
				case evt := <-eventCh:
					if sessionID != "" && evt.SessionID != sessionID {
						continue
					}
					writeSSE(string(evt.Type), map[string]any{
						"node_id":   evt.NodeID,
						"payload":   evt.Payload,
						"timestamp": evt.Timestamp,
					})
				default:
					goto drained
				}
			}
		drained:
			// 7. Send final "done" event with session state
			status := "completed"
			var state map[string]any
			if result.err != nil {
				status = "failed"
			}
			if result.session != nil {
				status = string(result.session.Status)
				state = result.session.State
			}
			doneData := map[string]any{
				"session_id": sessionID,
				"status":     status,
				"state":      state,
			}
			if result.err != nil {
				doneData["error"] = result.err.Error()
			}
			writeSSE("done", doneData)
			return
		case <-r.Context().Done():
			return
		}
	}
}
