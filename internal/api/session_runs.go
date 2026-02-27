package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// runConfigBody is the shared request body for creating or updating a Run's configuration.
type runConfigBody struct {
	Name      string                 `json:"name"`
	Sources   []upal.SessionSource   `json:"sources,omitempty"`
	Workflows []upal.SessionWorkflow `json:"workflows,omitempty"`
	Context   *upal.SessionContext   `json:"context,omitempty"`
	Schedule  string                 `json:"schedule,omitempty"`
}

func (b *runConfigBody) hasConfig() bool {
	return b.Name != "" || len(b.Sources) > 0 || len(b.Workflows) > 0 || b.Context != nil || b.Schedule != ""
}

func (s *Server) createNewRun(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var body runConfigBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var run *upal.Run
	var err error
	if body.hasConfig() {
		run, err = s.runSvc.CreateRunWithConfig(r.Context(), sessionID, "manual", body.Name, body.Sources, body.Workflows, body.Context, body.Schedule)
	} else {
		run, err = s.runSvc.CreateRun(r.Context(), sessionID, "manual")
	}
	if err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSONStatus(w, http.StatusCreated, run)
}

func (s *Server) listNewSessionRuns(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	details, err := s.runSvc.ListRunsBySession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(details))
}

func (s *Server) listAllNewRuns(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get("status")
	var (
		details []*upal.RunDetail
		err     error
	)
	if statusStr != "" {
		details, err = s.runSvc.ListRunsByStatus(r.Context(), upal.SessionRunStatus(statusStr))
	} else {
		details, err = s.runSvc.ListRuns(r.Context())
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(details))
}

func (s *Server) getNewRunDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := s.runSvc.GetRunDetail(r.Context(), id)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) produceNewRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Workflows []struct {
			Name      string `json:"name"`
			ChannelID string `json:"channel_id,omitempty"`
		} `json:"workflows"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if len(body.Workflows) == 0 {
		http.Error(w, "workflows list is required", http.StatusBadRequest)
		return
	}

	if s.collector != nil {
		requests := make([]services.WorkflowRequest, len(body.Workflows))
		for i, w := range body.Workflows {
			requests[i] = services.WorkflowRequest{Name: w.Name, ChannelID: w.ChannelID}
		}
		go s.collector.ProduceWorkflowsV2(context.Background(), id, requests)
	} else {
		if err := s.runSvc.UpdateRunStatus(r.Context(), id, upal.SessionRunProducing); err != nil {
			writeServiceError(w, err, http.StatusInternalServerError)
			return
		}
	}
	writeJSONStatus(w, http.StatusAccepted, map[string]any{
		"run_id":    id,
		"workflows": body.Workflows,
		"status":    "accepted",
	})
}

func (s *Server) publishNewRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		RunIDs []string `json:"run_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if len(body.RunIDs) == 0 {
		http.Error(w, "run_ids list is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Build a set for O(1) lookup of which workflow runs to publish.
	publishSet := make(map[string]struct{}, len(body.RunIDs))
	for _, rid := range body.RunIDs {
		publishSet[rid] = struct{}{}
	}

	// Single fetch: record published content and update statuses in one pass.
	wfRuns := s.runSvc.GetWorkflowRuns(ctx, id)
	for i, wr := range wfRuns {
		if _, ok := publishSet[wr.RunID]; !ok {
			continue
		}
		channel := wr.ChannelID
		if channel == "" {
			channel = "default"
		}
		pc := &upal.PublishedContent{
			SessionID:     id,
			WorkflowRunID: wr.RunID,
			Channel:       channel,
			Title:         wr.WorkflowName,
		}
		if err := s.runSvc.RecordPublished(ctx, pc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		wfRuns[i].Status = upal.WFRunPublished
	}
	s.runSvc.SetWorkflowRuns(ctx, id, wfRuns)

	// Transition run to published if all workflow runs are terminal.
	allTerminal := len(wfRuns) > 0
	for _, wr := range wfRuns {
		if wr.Status != upal.WFRunPublished && wr.Status != upal.WFRunRejected && wr.Status != upal.WFRunFailed {
			allTerminal = false
			break
		}
	}
	if allTerminal {
		if err := s.runSvc.UpdateRunStatus(ctx, id, upal.SessionRunPublished); err != nil {
			writeServiceError(w, err, http.StatusInternalServerError)
			return
		}
	}

	detail, err := s.runSvc.GetRunDetail(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) rejectNewRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.runSvc.RejectRun(r.Context(), id); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "rejected"})
}

func (s *Server) listNewRunSources(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fetches, err := s.runSvc.ListSourceFetches(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(fetches))
}

func (s *Server) getNewRunAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.runSvc.GetAnalysis(r.Context(), id)
	if err != nil {
		http.Error(w, "analysis not found", http.StatusNotFound)
		return
	}
	writeJSON(w, analysis)
}

func (s *Server) patchNewRunAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Summary  string   `json:"summary"`
		Insights []string `json:"insights"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.runSvc.UpdateAnalysis(r.Context(), id, body.Summary, body.Insights); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	analysis, _ := s.runSvc.GetAnalysis(r.Context(), id)
	writeJSON(w, analysis)
}

func (s *Server) updateRunConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body runConfigBody
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.runSvc.UpdateRunConfig(r.Context(), id, body.Name, body.Sources, body.Workflows, body.Context, body.Schedule); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	detail, err := s.runSvc.GetRunDetail(r.Context(), id)
	if err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) collectNewRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := s.runSvc.GetRun(r.Context(), id)
	if err != nil {
		writeServiceError(w, err, http.StatusNotFound)
		return
	}
	if run.Status != upal.SessionRunDraft {
		http.Error(w, "run is not in draft status", http.StatusConflict)
		return
	}
	if err := s.runSvc.UpdateRunStatus(r.Context(), id, upal.SessionRunCollecting); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	run.Status = upal.SessionRunCollecting
	if s.collector != nil && s.sessionSvc != nil {
		sess, err := s.sessionSvc.Get(r.Context(), run.SessionID)
		if err == nil {
			go s.collector.CollectAndAnalyzeV2(context.Background(), sess, run, false, 0)
		}
	}
	writeJSONStatus(w, http.StatusAccepted, run)
}

func (s *Server) cancelNewRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := s.runSvc.GetRun(r.Context(), id)
	if err != nil {
		writeServiceError(w, err, http.StatusNotFound)
		return
	}
	if run.Status != upal.SessionRunCollecting && run.Status != upal.SessionRunAnalyzing {
		http.Error(w, "run is not in collecting or analyzing status", http.StatusConflict)
		return
	}
	if err := s.runSvc.UpdateRunStatus(r.Context(), id, upal.SessionRunDraft); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	run.Status = upal.SessionRunDraft
	writeJSON(w, run)
}

func (s *Server) deleteNewRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.runSvc.DeleteRun(r.Context(), id); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) toggleRunSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Active bool `json:"active"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.runSvc.ToggleRunSchedule(r.Context(), id, body.Active); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	run, err := s.runSvc.GetRun(r.Context(), id)
	if err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, run)
}
