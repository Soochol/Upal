package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/generate"
)

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

	// Collect stage types from the pipeline for skill injection.
	var stageTypes []string
	if s.pipelineSvc != nil {
		if p, err := s.pipelineSvc.Get(r.Context(), pipelineID); err == nil {
			for _, st := range p.Stages {
				stageTypes = append(stageTypes, st.Type)
			}
		}
	}

	// Collect available workflows for reference.
	var workflows []generate.WorkflowRef
	if s.repo != nil {
		if wfs, err := s.repo.List(r.Context()); err == nil {
			for _, wf := range wfs {
				workflows = append(workflows, generate.WorkflowRef{Name: wf.Name, Description: wf.Description})
			}
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
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("configure pipeline: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, out)
}
