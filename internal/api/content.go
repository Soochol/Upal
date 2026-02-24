package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// GET /api/content-sessions
// Query params: pipeline_id=X, status=pending_review
// When pipeline_id is provided, returns composed ContentSessionDetail records.
func (s *Server) listContentSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pipelineID := r.URL.Query().Get("pipeline_id")
	statusStr := r.URL.Query().Get("status")

	// When pipeline_id is provided, return composed detail views.
	if pipelineID != "" {
		archivedOnly := r.URL.Query().Get("archived_only") == "true"

		var details []*upal.ContentSessionDetail
		var err error
		if archivedOnly {
			details, err = s.contentSvc.ListArchivedSessionDetails(ctx, pipelineID)
		} else {
			details, err = s.contentSvc.ListSessionDetails(ctx, pipelineID)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Filter by status if requested.
		if statusStr != "" {
			filtered := make([]*upal.ContentSessionDetail, 0, len(details))
			for _, d := range details {
				if string(d.Status) == statusStr {
					filtered = append(filtered, d)
				}
			}
			details = filtered
		}
		if details == nil {
			details = []*upal.ContentSessionDetail{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(details)
		return
	}

	// No pipeline_id: return raw sessions.
	var (
		sessions []*upal.ContentSession
		err      error
	)
	if statusStr != "" {
		sessions, err = s.contentSvc.ListSessionsByStatus(ctx, upal.ContentSessionStatus(statusStr))
	} else {
		sessions, err = s.contentSvc.ListSessions(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []*upal.ContentSession{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// GET /api/content-sessions/{id}
// Returns composed ContentSessionDetail with sources, analysis, and workflow results.
func (s *Server) getContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := s.contentSvc.GetSessionDetail(r.Context(), id)
	if err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// PATCH /api/content-sessions/{id}
// Body: {"action": "approve" | "reject"}
func (s *Server) patchContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	var err error
	switch body.Action {
	case "approve":
		err = s.contentSvc.ApproveSession(ctx, id)
	case "reject":
		err = s.contentSvc.RejectSession(ctx, id)
	default:
		http.Error(w, "action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	sess, _ := s.contentSvc.GetSession(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// POST /api/content-sessions/{id}/produce
// Body: {"workflows": ["blog", "shorts"]}
// Validates the session exists and launches background workflow production.
func (s *Server) produceContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Workflows []string `json:"workflows"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.Workflows) == 0 {
		http.Error(w, "workflows list is required", http.StatusBadRequest)
		return
	}

	// Verify session exists before accepting.
	if _, err := s.contentSvc.GetSession(r.Context(), id); err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}

	// Launch background production if collector is wired.
	if s.collector != nil {
		go s.collector.ProduceWorkflows(context.Background(), id, body.Workflows)
	} else {
		// Fallback: just update status directly (no collector available).
		if err := s.contentSvc.UpdateSessionStatus(r.Context(), id, upal.SessionProducing); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]any{
		"session_id": id,
		"workflows":  body.Workflows,
		"status":     "accepted",
	})
}

// GET /api/content-sessions/{id}/sources
func (s *Server) listSessionSources(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fetches, err := s.contentSvc.ListSourceFetches(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if fetches == nil {
		fetches = []*upal.SourceFetch{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fetches)
}

// PATCH /api/content-sessions/{id}/analysis
func (s *Server) patchSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Summary  string   `json:"summary"`
		Insights []string `json:"insights"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.contentSvc.UpdateAnalysis(r.Context(), id, body.Summary, body.Insights); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	analysis, _ := s.contentSvc.GetAnalysis(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// GET /api/content-sessions/{id}/analysis
func (s *Server) getSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.contentSvc.GetAnalysis(r.Context(), id)
	if err != nil {
		http.Error(w, "analysis not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// POST /api/content-sessions/{id}/publish
// Body: {"run_ids": ["run-xxx", ...]}
// Creates PublishedContent records for each run and transitions session to published.
func (s *Server) publishContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		RunIDs []string `json:"run_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.RunIDs) == 0 {
		http.Error(w, "run_ids list is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Look up workflow results to derive titles for published records.
	wfResults := s.contentSvc.GetWorkflowResults(ctx, id)
	runToName := make(map[string]string, len(wfResults))
	for _, wr := range wfResults {
		if wr.RunID != "" {
			runToName[wr.RunID] = wr.WorkflowName
		}
	}

	// Create a PublishedContent record for each run_id.
	for _, runID := range body.RunIDs {
		title := runToName[runID] // may be empty if not found
		pc := &upal.PublishedContent{
			SessionID:     id,
			WorkflowRunID: runID,
			Channel:       "default",
			Title:         title,
		}
		if err := s.contentSvc.RecordPublished(ctx, pc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Transition session to published.
	if err := s.contentSvc.UpdateSessionStatus(ctx, id, upal.SessionPublished); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return the composed session detail.
	detail, err := s.contentSvc.GetSessionDetail(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// GET /api/published
// Query params: session_id=X, channel=youtube
func (s *Server) listPublished(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var (
		items []*upal.PublishedContent
		err   error
	)
	if channel := r.URL.Query().Get("channel"); channel != "" {
		items, err = s.contentSvc.ListPublishedByChannel(ctx, channel)
	} else if sessionID := r.URL.Query().Get("session_id"); sessionID != "" {
		items, err = s.contentSvc.ListPublishedBySession(ctx, sessionID)
	} else {
		items, err = s.contentSvc.ListPublished(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []*upal.PublishedContent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// GET /api/surges
// Query param: active=true to filter to undismissed only
func (s *Server) listSurges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var (
		events []*upal.SurgeEvent
		err    error
	)
	if r.URL.Query().Get("active") == "true" {
		events, err = s.contentSvc.ListActiveSurges(ctx)
	} else {
		events, err = s.contentSvc.ListSurges(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []*upal.SurgeEvent{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// POST /api/surges/{id}/dismiss
func (s *Server) dismissSurge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DismissSurge(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/surges/{id}/create-session
// Phase 1 stub: actual implementation in Phase 2.
func (s *Server) createSessionFromSurge(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented in Phase 1", http.StatusNotImplemented)
}

// POST /api/pipelines/{id}/collect
// Body (optional): {"isTest": bool, "limit": int}
// Creates a content session and launches background collection + analysis.
func (s *Server) collectPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Parse optional body.
	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	// Body is optional — ignore decode errors for empty bodies.
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Fetch the pipeline to pass to the collector.
	pipeline, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	sess := &upal.ContentSession{
		PipelineID:  id,
		TriggerType: "manual",
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Launch background collection if collector is wired.
	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), pipeline, sess, body.IsTest, body.Limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": sess.ID})
}

// POST /api/content-sessions/{id}/archive
func (s *Server) archiveContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.ArchiveSession(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "already archived") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	sess, _ := s.contentSvc.GetSession(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// POST /api/content-sessions/{id}/unarchive
func (s *Server) unarchiveContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.UnarchiveSession(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "not archived") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	sess, _ := s.contentSvc.GetSession(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

// DELETE /api/content-sessions/{id}
func (s *Server) deleteContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DeleteSession(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if strings.Contains(err.Error(), "must be archived") {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
