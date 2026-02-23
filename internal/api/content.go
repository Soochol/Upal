package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// GET /api/content-sessions
// Query params: pipeline_id=X, status=pending_review
func (s *Server) listContentSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pipelineID := r.URL.Query().Get("pipeline_id")
	statusStr := r.URL.Query().Get("status")
	var (
		sessions []*upal.ContentSession
		err      error
	)
	switch {
	case pipelineID != "" && statusStr != "":
		sessions, err = s.contentSvc.ListSessionsByPipelineAndStatus(ctx, pipelineID, upal.ContentSessionStatus(statusStr))
	case pipelineID != "":
		sessions, err = s.contentSvc.ListSessionsByPipeline(ctx, pipelineID)
	case statusStr != "":
		sessions, err = s.contentSvc.ListSessionsByStatus(ctx, upal.ContentSessionStatus(statusStr))
	default:
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
func (s *Server) getContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
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
// Phase 1: Records intent only (actual execution wired in Phase 3).
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
	ctx := r.Context()
	if err := s.contentSvc.UpdateSessionStatus(ctx, id, upal.SessionProducing); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
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
// Phase 1: Creates a session manually (actual collection wired in Phase 2).
func (s *Server) collectPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess := &upal.ContentSession{
		PipelineID:  id,
		TriggerType: "manual",
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": sess.ID})
}
