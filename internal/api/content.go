package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) listContentSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pipelineID := r.URL.Query().Get("pipeline_id")
	statusStr := r.URL.Query().Get("status")

	if pipelineID != "" {
		templateOnly := r.URL.Query().Get("template_only") == "true"

		var details []*upal.ContentSessionDetail
		var err error
		if templateOnly {
			details, err = s.contentSvc.ListTemplateDetailsByPipeline(ctx, pipelineID)
		} else {
			details, err = s.contentSvc.ListSessionDetailsByPipelineAndStatus(ctx, pipelineID, upal.ContentSessionStatus(statusStr))
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, orEmpty(details))
		return
	}

	if r.URL.Query().Get("detail") == "true" && statusStr == "" {
		details, err := s.contentSvc.ListAllInstanceSessionDetails(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, orEmpty(details))
		return
	}

	if statusStr != "" {
		details, err := s.contentSvc.ListSessionDetailsByStatus(ctx, upal.ContentSessionStatus(statusStr))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, orEmpty(details))
		return
	}

	sessions, err := s.contentSvc.ListSessions(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(sessions))
}

func (s *Server) getContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := s.contentSvc.GetSessionDetail(r.Context(), id)
	if err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) patchContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Action string `json:"action"`
	}
	if !decodeJSON(w, r, &body) {
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
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	sess, _ := s.contentSvc.GetSession(ctx, id)
	writeJSON(w, sess)
}

func (s *Server) produceContentSession(w http.ResponseWriter, r *http.Request) {
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

	if _, err := s.contentSvc.GetSession(r.Context(), id); err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}

	seen := make(map[string]bool, len(body.Workflows))
	var requests []services.WorkflowRequest
	for _, bw := range body.Workflows {
		if seen[bw.Name] {
			continue
		}
		seen[bw.Name] = true
		requests = append(requests, services.WorkflowRequest{
			Name:      bw.Name,
			ChannelID: bw.ChannelID,
		})
	}

	if s.collector != nil {
		go s.collector.ProduceWorkflows(context.Background(), id, requests)
	} else {
		if err := s.contentSvc.UpdateSessionStatus(r.Context(), id, upal.SessionProducing); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writeJSONStatus(w, http.StatusAccepted, map[string]any{
		"session_id": id,
		"workflows":  body.Workflows,
		"status":     "accepted",
	})
}

func (s *Server) listSessionSources(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	fetches, err := s.contentSvc.ListSourceFetches(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(fetches))
}

func (s *Server) patchSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Summary  string   `json:"summary"`
		Insights []string `json:"insights"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.contentSvc.UpdateAnalysis(r.Context(), id, body.Summary, body.Insights); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	analysis, _ := s.contentSvc.GetAnalysis(r.Context(), id)
	writeJSON(w, analysis)
}

func (s *Server) getSessionAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	analysis, err := s.contentSvc.GetAnalysis(r.Context(), id)
	if err != nil {
		http.Error(w, "analysis not found", http.StatusNotFound)
		return
	}
	writeJSON(w, analysis)
}

func (s *Server) publishContentSession(w http.ResponseWriter, r *http.Request) {
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

	wfResults := s.contentSvc.GetWorkflowResults(ctx, id)
	runToName := make(map[string]string, len(wfResults))
	runToChannel := make(map[string]string, len(wfResults))
	for _, wr := range wfResults {
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
		if err := s.contentSvc.RecordPublished(ctx, pc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	results := s.contentSvc.GetWorkflowResults(ctx, id)
	for i, wr := range results {
		for _, runID := range body.RunIDs {
			if wr.RunID == runID {
				results[i].Status = upal.WFResultPublished
			}
		}
	}
	s.contentSvc.SetWorkflowResults(ctx, id, results)

	s.checkAndFinalizeSession(ctx, id)

	detail, err := s.contentSvc.GetSessionDetail(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) rejectWorkflowResult(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		RunID string `json:"run_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.RunID == "" {
		http.Error(w, "run_id is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	results := s.contentSvc.GetWorkflowResults(ctx, id)
	updated := false
	for i, wr := range results {
		if wr.RunID == body.RunID {
			results[i].Status = upal.WFResultRejected
			updated = true
			break
		}
	}
	if !updated {
		http.Error(w, "run_id not found", http.StatusNotFound)
		return
	}
	s.contentSvc.SetWorkflowResults(ctx, id, results)

	s.checkAndFinalizeSession(ctx, id)

	writeJSON(w, map[string]string{"status": "rejected"})
}

// checkAndFinalizeSession transitions session to published if all workflow results are terminal.
func (s *Server) checkAndFinalizeSession(ctx context.Context, id string) {
	results := s.contentSvc.GetWorkflowResults(ctx, id)
	if len(results) == 0 {
		return
	}
	for _, wr := range results {
		if wr.Status != upal.WFResultPublished && wr.Status != upal.WFResultRejected && wr.Status != upal.WFResultFailed {
			return
		}
	}
	_ = s.contentSvc.UpdateSessionStatus(ctx, id, upal.SessionPublished)
}

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
	writeJSON(w, orEmpty(items))
}

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
	writeJSON(w, orEmpty(events))
}

func (s *Server) dismissSurge(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DismissSurge(r.Context(), id); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createSessionFromSurge(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented in Phase 1", http.StatusNotImplemented)
}

func (s *Server) createDraftSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PipelineID string                  `json:"pipeline_id"`
		Name       string                  `json:"name"`
		IsTemplate bool                    `json:"is_template"`
		Sources    []upal.PipelineSource   `json:"sources,omitempty"`
		Schedule   string                  `json:"schedule,omitempty"`
		Model      string                  `json:"model,omitempty"`
		Workflows  []upal.PipelineWorkflow `json:"workflows,omitempty"`
		Context    *upal.PipelineContext   `json:"context,omitempty"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.PipelineID == "" {
		http.Error(w, "pipeline_id is required", http.StatusBadRequest)
		return
	}

	sess := &upal.ContentSession{
		PipelineID:  body.PipelineID,
		Name:        body.Name,
		IsTemplate:  body.IsTemplate,
		Sources:     body.Sources,
		Schedule:    body.Schedule,
		Model:       body.Model,
		Workflows:   body.Workflows,
		Context:     body.Context,
		Status:      upal.SessionDraft,
		TriggerType: "manual",
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	detail, err := s.contentSvc.GetSessionDetail(r.Context(), sess.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusCreated, detail)
}

func (s *Server) patchSessionSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name          string                  `json:"name,omitempty"`
		Sources       []upal.PipelineSource   `json:"sources,omitempty"`
		Schedule      string                  `json:"schedule,omitempty"`
		ClearSchedule bool                    `json:"clear_schedule,omitempty"`
		Model         string                  `json:"model,omitempty"`
		Workflows     []upal.PipelineWorkflow `json:"workflows,omitempty"`
		Context       *upal.PipelineContext   `json:"context,omitempty"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	settings := upal.SessionSettings{
		Name:          body.Name,
		Sources:       body.Sources,
		Schedule:      body.Schedule,
		ClearSchedule: body.ClearSchedule,
		Model:         body.Model,
		Workflows:     body.Workflows,
		Context:       body.Context,
	}
	if err := s.contentSvc.UpdateSessionSettings(r.Context(), id, settings); err != nil {
		writeServiceError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.contentSvc.GetSessionDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) collectSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	sess, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}

	pipeline, err := s.pipelineSvc.Get(r.Context(), sess.PipelineID)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), pipeline, sess, body.IsTest, body.Limit)
	}

	writeJSONStatus(w, http.StatusAccepted, map[string]any{"session_id": id, "status": "accepted"})
}

// transitionSessionStatus validates the current status, applies the new status, and returns the detail.
func (s *Server) transitionSessionStatus(w http.ResponseWriter, r *http.Request, requiredStatus, newStatus upal.ContentSessionStatus, errMsg string) {
	id := chi.URLParam(r, "id")
	sess, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "content session not found", http.StatusNotFound)
		return
	}
	if sess.Status != requiredStatus {
		http.Error(w, errMsg, http.StatusConflict)
		return
	}
	if err := s.contentSvc.UpdateSessionStatus(r.Context(), id, newStatus); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	detail, err := s.contentSvc.GetSessionDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, detail)
}

func (s *Server) activateSession(w http.ResponseWriter, r *http.Request) {
	s.transitionSessionStatus(w, r, upal.SessionDraft, upal.SessionActive, "only draft sessions can be activated")
}

func (s *Server) deactivateSession(w http.ResponseWriter, r *http.Request) {
	s.transitionSessionStatus(w, r, upal.SessionActive, upal.SessionDraft, "only active sessions can be deactivated")
}

// newInstanceFromTemplate creates a new session instance inheriting settings from a template.
func newInstanceFromTemplate(tmpl *upal.ContentSession) *upal.ContentSession {
	return &upal.ContentSession{
		PipelineID:      tmpl.PipelineID,
		TriggerType:     "manual",
		ParentSessionID: tmpl.ID,
		Sources:         tmpl.Sources,
		Schedule:        tmpl.Schedule,
		Model:           tmpl.Model,
		Workflows:       tmpl.Workflows,
		Context:         tmpl.Context,
	}
}

func (s *Server) collectPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	pipeline, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	templates, tErr := s.contentSvc.ListTemplatesByPipeline(r.Context(), id)
	if tErr != nil || len(templates) == 0 {
		http.Error(w, "no template session found for this pipeline", http.StatusBadRequest)
		return
	}

	sess := newInstanceFromTemplate(templates[0])
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), pipeline, sess, body.IsTest, body.Limit)
	}

	writeJSONStatus(w, http.StatusCreated, map[string]any{"session_id": sess.ID})
}

func (s *Server) retryAnalyze(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.collector == nil {
		http.Error(w, "collector not available", http.StatusServiceUnavailable)
		return
	}
	if err := s.collector.RetryAnalysis(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) generateAngleWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		AngleID string `json:"angle_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AngleID == "" {
		http.Error(w, `{"error":"angle_id is required"}`, http.StatusBadRequest)
		return
	}

	if s.collector == nil {
		http.Error(w, `{"error":"content collector not available"}`, http.StatusServiceUnavailable)
		return
	}

	angle, err := s.collector.GenerateWorkflowForAngle(r.Context(), id, body.AngleID)
	if err != nil {
		slog.Error("generate-workflow failed", "error", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	writeJSON(w, angle)
}

// POST /api/content-sessions/{id}/run
// Creates a new instance from this template session and launches collection.
func (s *Server) runSessionInstance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tmpl, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !tmpl.IsTemplate {
		http.Error(w, "only template sessions can spawn instances", http.StatusBadRequest)
		return
	}

	pipeline, err := s.pipelineSvc.Get(r.Context(), tmpl.PipelineID)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	sess := newInstanceFromTemplate(tmpl)
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), pipeline, sess, body.IsTest, body.Limit)
	}

	writeJSONStatus(w, http.StatusCreated, map[string]any{"session_id": sess.ID})
}

func (s *Server) deleteContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DeleteSession(r.Context(), id); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
