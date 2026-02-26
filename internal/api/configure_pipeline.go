package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/generate"
)

// ConfigurePipelineRequest is the shared request body for both pipeline and session LLM configuration.
type ConfigurePipelineRequest struct {
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

// buildConfigureInput maps a ConfigurePipelineRequest into a generate.ConfigurePipelineInput,
// enriching it with stage types from the given pipeline and available workflow references.
func (s *Server) buildConfigureInput(r *http.Request, req ConfigurePipelineRequest, pipelineID string) generate.ConfigurePipelineInput {
	ctx := r.Context()

	var stageTypes []string
	if s.pipelineSvc != nil && pipelineID != "" {
		if p, err := s.pipelineSvc.Get(ctx, pipelineID); err == nil {
			for _, st := range p.Stages {
				stageTypes = append(stageTypes, st.Type)
			}
		} else {
			slog.Warn("failed to load pipeline for configure", "pipeline_id", pipelineID, "error", err)
		}
	}

	var workflows []generate.WorkflowRef
	if s.repo != nil {
		if wfs, err := s.repo.List(ctx); err == nil {
			for _, wf := range wfs {
				workflows = append(workflows, generate.WorkflowRef{Name: wf.Name, Description: wf.Description})
			}
		} else {
			slog.Warn("failed to list workflows for configure", "error", err)
		}
	}

	return generate.ConfigurePipelineInput{
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
	}
}

func (s *Server) configurePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

	var req ConfigurePipelineRequest
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

	out, err := s.generator.ConfigurePipeline(r.Context(), s.buildConfigureInput(r, req, pipelineID))
	if err != nil {
		http.Error(w, fmt.Sprintf("configure pipeline: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, out)
}
