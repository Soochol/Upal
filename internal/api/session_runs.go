package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createNewRun(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	run, err := s.runSvc.CreateRun(r.Context(), sessionID, "manual")
	if err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	// Collection bridge (go s.collector.CollectAndAnalyzeV2) will be added later.
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

	if err := s.runSvc.UpdateRunStatus(r.Context(), id, upal.SessionRunProducing); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	// Collection bridge (async workflow execution) will be added later.
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

	wfRuns := s.runSvc.GetWorkflowRuns(ctx, id)
	runToName := make(map[string]string, len(wfRuns))
	runToChannel := make(map[string]string, len(wfRuns))
	for _, wr := range wfRuns {
		if wr.RunID != "" {
			runToName[wr.RunID] = wr.WorkflowName
			runToChannel[wr.RunID] = wr.ChannelID
		}
	}

	for _, runID := range body.RunIDs {
		channel := runToChannel[runID]
		if channel == "" {
			channel = "default"
		}
		pc := &upal.PublishedContent{
			SessionID:     id,
			WorkflowRunID: runID,
			Channel:       channel,
			Title:         runToName[runID],
		}
		if err := s.runSvc.RecordPublished(ctx, pc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Update workflow run statuses to published.
	results := s.runSvc.GetWorkflowRuns(ctx, id)
	for i, wr := range results {
		for _, runID := range body.RunIDs {
			if wr.RunID == runID {
				results[i].Status = upal.WFRunPublished
			}
		}
	}
	s.runSvc.SetWorkflowRuns(ctx, id, results)

	// Transition run to published if all workflow runs are terminal.
	allTerminal := len(results) > 0
	for _, wr := range results {
		if wr.Status != upal.WFRunPublished && wr.Status != upal.WFRunRejected && wr.Status != upal.WFRunFailed {
			allTerminal = false
			break
		}
	}
	if allTerminal {
		_ = s.runSvc.UpdateRunStatus(ctx, id, upal.SessionRunPublished)
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
