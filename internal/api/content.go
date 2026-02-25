package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// POST /api/content-sessions
// Body: {"pipeline_id": "...", "trigger_type": "manual"}
// Creates a draft session that the user can configure before starting collection.
func (s *Server) createDraftSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PipelineID string `json:"pipeline_id"`
		Name       string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PipelineID == "" {
		http.Error(w, "pipeline_id is required", http.StatusBadRequest)
		return
	}
	if _, err := s.pipelineSvc.Get(r.Context(), body.PipelineID); err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}
	sess := &upal.ContentSession{
		PipelineID:  body.PipelineID,
		Name:        body.Name,
		Status:      upal.SessionDraft,
		TriggerType: "manual",
		IsTemplate:  true,
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	detail, _ := s.contentSvc.GetSessionDetail(r.Context(), sess.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(detail)
}

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
		templatesOnly := r.URL.Query().Get("templates") != "false" // default true for pipeline queries

		var details []*upal.ContentSessionDetail
		var err error
		if archivedOnly {
			details, err = s.contentSvc.ListArchivedSessionDetails(ctx, pipelineID)
		} else if templatesOnly {
			details, err = s.contentSvc.ListTemplateDetailsByPipeline(ctx, pipelineID)
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

	// No pipeline_id but status provided: return composed details (includes pipeline_name).
	if statusStr != "" {
		includeArchived := r.URL.Query().Get("include_archived") == "true"
		var details []*upal.ContentSessionDetail
		var err error
		if includeArchived {
			details, err = s.contentSvc.ListSessionDetailsByStatusIncludeArchived(ctx, upal.ContentSessionStatus(statusStr))
		} else {
			details, err = s.contentSvc.ListSessionDetailsByStatus(ctx, upal.ContentSessionStatus(statusStr))
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if details == nil {
			details = []*upal.ContentSessionDetail{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(details)
		return
	}

	// No pipeline_id, no status: return raw sessions.
	sessions, err := s.contentSvc.ListSessions(ctx)
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
		if err == nil && s.collector != nil {
			// Auto-produce using session's pre-configured workflows.
			if sess, e := s.contentSvc.GetSession(ctx, id); e == nil && len(sess.Workflows) > 0 {
				var requests []services.WorkflowRequest
				for _, pw := range sess.Workflows {
					requests = append(requests, services.WorkflowRequest{
						Name:      pw.WorkflowName,
						ChannelID: pw.ChannelID,
					})
				}
				go s.collector.ProduceWorkflows(context.Background(), id, requests)
			}
		}
	case "reject":
		err = s.contentSvc.RejectSession(ctx, id)
	default:
		http.Error(w, "action must be 'approve' or 'reject'", http.StatusBadRequest)
		return
	}
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	detail, _ := s.contentSvc.GetSessionDetail(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// PATCH /api/content-sessions/{id}/settings
func (s *Server) patchSessionSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Name      string                  `json:"name,omitempty"`
		Sources   []upal.PipelineSource   `json:"sources,omitempty"`
		Schedule  *string                 `json:"schedule,omitempty"`
		Model     string                  `json:"model,omitempty"`
		Workflows []upal.PipelineWorkflow `json:"workflows,omitempty"`
		Context   *upal.PipelineContext   `json:"context,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	settings := services.SessionSettings{
		Name:      body.Name,
		Sources:   body.Sources,
		Model:     body.Model,
		Workflows: body.Workflows,
		Context:   body.Context,
	}
	if body.Schedule != nil {
		if *body.Schedule == "" {
			settings.ClearSchedule = true
		} else {
			settings.Schedule = *body.Schedule
		}
	}
	if err := s.contentSvc.UpdateSessionSettings(r.Context(), id, settings); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	detail, _ := s.contentSvc.GetSessionDetail(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// POST /api/content-sessions/{id}/produce
// Body: {"workflows": [{"name": "blog", "channel_id": "youtube"}, ...]}
// Validates the session exists and launches background workflow production.
func (s *Server) produceContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Workflows []struct {
			Name      string `json:"name"`
			ChannelID string `json:"channel_id,omitempty"`
		} `json:"workflows"`
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

	// Deduplicate by workflow name (first occurrence wins).
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

	// Launch background production if collector is wired.
	if s.collector != nil {
		go s.collector.ProduceWorkflows(context.Background(), id, requests)
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
		Summary      string   `json:"summary"`
		Insights     []string `json:"insights"`
		AngleID      string   `json:"angle_id,omitempty"`
		WorkflowName string   `json:"workflow_name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Summary != "" || body.Insights != nil {
		if err := s.contentSvc.UpdateAnalysis(r.Context(), id, body.Summary, body.Insights); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	if body.AngleID != "" && body.WorkflowName != "" {
		if err := s.contentSvc.UpdateAngleWorkflow(r.Context(), id, body.AngleID, body.WorkflowName); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
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

	// Look up workflow results to derive titles and channels for published records.
	wfResults := s.contentSvc.GetWorkflowResults(ctx, id)
	runToName := make(map[string]string, len(wfResults))
	runToChannel := make(map[string]string, len(wfResults))
	for _, wr := range wfResults {
		if wr.RunID != "" {
			runToName[wr.RunID] = wr.WorkflowName
			runToChannel[wr.RunID] = wr.ChannelID
		}
	}

	// Create a PublishedContent record for each run_id.
	for _, runID := range body.RunIDs {
		title := runToName[runID] // may be empty if not found
		channel := runToChannel[runID]
		if channel == "" {
			channel = "default"
		}
		pc := &upal.PublishedContent{
			SessionID:     id,
			WorkflowRunID: runID,
			Channel:       channel,
			Title:         title,
		}
		if err := s.contentSvc.RecordPublished(ctx, pc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Mark each published run_id in workflow results.
	results := s.contentSvc.GetWorkflowResults(ctx, id)
	for i, wr := range results {
		for _, runID := range body.RunIDs {
			if wr.RunID == runID {
				results[i].Status = upal.WFResultPublished
			}
		}
	}
	s.contentSvc.SetWorkflowResults(ctx, id, results)

	// Check if all results are terminal — only then transition session.
	s.checkAndFinalizeSession(ctx, id)

	// Return the composed session detail.
	detail, err := s.contentSvc.GetSessionDetail(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// POST /api/content-sessions/{id}/reject-result
// Body: {"run_id": "..."}
func (s *Server) rejectWorkflowResult(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		RunID string `json:"run_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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

	// Check if all results are now terminal (published or rejected)
	s.checkAndFinalizeSession(ctx, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

// checkAndFinalizeSession transitions session to published if all workflow results are terminal.
func (s *Server) checkAndFinalizeSession(ctx context.Context, id string) {
	results := s.contentSvc.GetWorkflowResults(ctx, id)
	if len(results) == 0 {
		return
	}
	allTerminal := true
	for _, wr := range results {
		if wr.Status != upal.WFResultPublished && wr.Status != upal.WFResultRejected && wr.Status != upal.WFResultFailed {
			allTerminal = false
			break
		}
	}
	if allTerminal {
		_ = s.contentSvc.UpdateSessionStatus(ctx, id, upal.SessionPublished)
	}
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
		if errors.Is(err, repository.ErrNotFound) {
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
		IsTemplate:  false,
		Sources:     pipeline.Sources,
		Schedule:    pipeline.Schedule,
		Model:       pipeline.Model,
		Workflows:   pipeline.Workflows,
		Context:     pipeline.Context,
	}
	if err := s.contentSvc.CreateSession(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Launch background collection if collector is wired.
	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), sess, body.IsTest, body.Limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": sess.ID})
}

// POST /api/content-sessions/{id}/collect
// Always creates a new child instance from the given template/session and runs collection on it.
func (s *Server) collectSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	session, err := s.contentSvc.GetSession(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var body struct {
		IsTest bool `json:"isTest"`
		Limit  int  `json:"limit"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Always create a new instance from this template/session.
	// Append timestamp to distinguish instances from the parent template.
	instanceName := session.Name
	if instanceName != "" {
		instanceName += " — " + time.Now().Format("01/02 15:04")
	}
	execSess := &upal.ContentSession{
		PipelineID:      session.PipelineID,
		Name:            instanceName,
		TriggerType:     "manual",
		IsTemplate:      false,
		ParentSessionID: session.ID,
		Sources:         session.Sources,
		Model:           session.Model,
		Workflows:       session.Workflows,
		Context:         session.Context,
	}
	if err := s.contentSvc.CreateSession(r.Context(), execSess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.collector != nil {
		go s.collector.CollectAndAnalyze(context.Background(), execSess, body.IsTest, body.Limit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"session_id": execSess.ID})
}

// POST /api/content-sessions/{id}/retry-analyze
// Re-runs LLM analysis for a session stuck in "analyzing" state.
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// POST /api/content-sessions/{id}/generate-workflow
// Body: {"angle_id": "angle-1"}
// Generates a new workflow for an unmatched content angle using the Generator.
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
		log.Printf("generate-workflow: %v", err)
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(angle)
}

// POST /api/content-sessions/{id}/archive
func (s *Server) archiveContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.ArchiveSession(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if errors.Is(err, upal.ErrAlreadyArchived) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	detail, err := s.contentSvc.GetSessionDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// POST /api/content-sessions/{id}/unarchive
func (s *Server) unarchiveContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.UnarchiveSession(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if errors.Is(err, upal.ErrNotArchived) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	detail, err := s.contentSvc.GetSessionDetail(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// DELETE /api/content-sessions/{id}
func (s *Server) deleteContentSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.contentSvc.DeleteSession(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else if errors.Is(err, upal.ErrMustBeArchived) {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
