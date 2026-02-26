package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/generate"
)

type ConfigureSessionRequest struct {
	Message          string                 `json:"message"`
	Model            string                 `json:"model,omitempty"`
	Thinking         bool                   `json:"thinking,omitempty"`
	History          []generate.ChatMessage `json:"history,omitempty"`
	CurrentSources   json.RawMessage        `json:"current_sources"`
	CurrentSchedule  string                 `json:"current_schedule"`
	CurrentWorkflows json.RawMessage        `json:"current_workflows"`
	CurrentModel     string                 `json:"current_model"`
	CurrentContext   json.RawMessage        `json:"current_context,omitempty"`
}

func (s *Server) configureSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var req ConfigureSessionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	// Load session to resolve pipeline context.
	sess, err := s.contentSvc.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("session not found: %v", err), http.StatusNotFound)
		return
	}

	// Collect stage types from the parent pipeline for skill injection.
	var stageTypes []string
	if s.pipelineSvc != nil && sess.PipelineID != "" {
		if p, err := s.pipelineSvc.Get(r.Context(), sess.PipelineID); err == nil {
			for _, st := range p.Stages {
				stageTypes = append(stageTypes, st.Type)
			}
		} else {
			slog.Warn("failed to load pipeline for session configure", "pipeline_id", sess.PipelineID, "error", err)
		}
	}

	// Collect available workflows for reference.
	var workflows []generate.WorkflowRef
	if s.repo != nil {
		if wfs, err := s.repo.List(r.Context()); err == nil {
			for _, wf := range wfs {
				workflows = append(workflows, generate.WorkflowRef{Name: wf.Name, Description: wf.Description})
			}
		} else {
			slog.Warn("failed to list workflows for session configure", "error", err)
		}
	}

	out, err := s.generator.ConfigurePipeline(r.Context(), generate.ConfigurePipelineInput{
		Message:            req.Message,
		Model:              req.Model,
		Thinking:           req.Thinking,
		History:            req.History,
		CurrentSources:     req.CurrentSources,
		CurrentSchedule:    req.CurrentSchedule,
		CurrentWorkflows:   req.CurrentWorkflows,
		CurrentModel:       req.CurrentModel,
		CurrentContext:     req.CurrentContext,
		StageTypes:         stageTypes,
		AvailableWorkflows: workflows,
		Session: &generate.SessionContext{
			Name:       sess.Name,
			Status:     string(sess.Status),
			IsTemplate: sess.IsTemplate,
		},
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("configure session: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, out)
}
