package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/agents"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// RunRequest is the JSON body for workflow execution.
type RunRequest struct {
	Inputs map[string]any `json:"inputs"`
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

	// 4. Build DAGAgent from workflow
	dagAgent, err := agents.NewDAGAgent(wf, s.llms, s.toolReg)
	if err != nil {
		sendSSEError(w, flusher, err)
		return
	}

	// 5. Create ADK Runner
	adkRunner, err := runner.New(runner.Config{
		AppName:        wf.Name,
		Agent:          dagAgent,
		SessionService: s.sessionService,
	})
	if err != nil {
		sendSSEError(w, flusher, err)
		return
	}

	// 6. Create session with user inputs as initial state
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())
	userID := "default"

	inputState := make(map[string]any)
	for k, v := range req.Inputs {
		inputState["__user_input__"+k] = v
	}

	_, err = s.sessionService.Create(r.Context(), &session.CreateRequest{
		AppName:   wf.Name,
		UserID:    userID,
		SessionID: sessionID,
		State:     inputState,
	})
	if err != nil {
		sendSSEError(w, flusher, err)
		return
	}

	// 7. Run and stream events
	userContent := genai.NewContentFromText("run", genai.RoleUser)
	for event, err := range adkRunner.Run(r.Context(), userID, sessionID, userContent, agent.RunConfig{}) {
		if err != nil {
			sendSSEError(w, flusher, err)
			return
		}
		if event == nil {
			continue
		}
		jsonData, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	// 8. Collect final session state
	finalState := make(map[string]any)
	getResp, err := s.sessionService.Get(r.Context(), &session.GetRequest{
		AppName:   wf.Name,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err == nil {
		for k, v := range getResp.Session.State().All() {
			if !strings.HasPrefix(k, "__") {
				finalState[k] = v
			}
		}
	}

	// 9. Send done event with state
	sendSSEDone(w, flusher, map[string]any{"status": "completed", "session_id": sessionID, "state": finalState})
}
