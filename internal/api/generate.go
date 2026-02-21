package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// GenerateRequest is the JSON body for workflow generation from natural language.
// When ExistingWorkflow is provided, the generator operates in edit mode.
type GenerateRequest struct {
	Description      string                     `json:"description"`
	Model            string                     `json:"model"`
	ExistingWorkflow *upal.WorkflowDefinition   `json:"existing_workflow,omitempty"`
}

// GeneratePipelineRequest is the JSON body for pipeline generation from natural language.
// When ExistingPipeline is provided, the generator operates in edit mode.
type GeneratePipelineRequest struct {
	Description      string         `json:"description"`
	ExistingPipeline *upal.Pipeline `json:"existing_pipeline,omitempty"`
}

func (s *Server) generatePipeline(w http.ResponseWriter, r *http.Request) {
	var req GeneratePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	bundle, err := s.generator.GeneratePipelineBundle(r.Context(), req.Description, req.ExistingPipeline)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate thumbnail best-effort — don't fail the request if this fails.
	thumbCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if svg, thumbErr := s.generator.GeneratePipelineThumbnail(thumbCtx, &bundle.Pipeline); thumbErr == nil {
		bundle.Pipeline.ThumbnailSVG = svg
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bundle)
}

func (s *Server) generateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Description == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return
	}

	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	// The model is already set on the generator at startup; the request model
	// field is accepted for API compatibility but not used to switch models
	// at runtime (the generator always uses its configured LLM).

	wf, err := s.generator.Generate(r.Context(), req.Description, req.ExistingWorkflow)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate thumbnail best-effort — don't fail the request if this fails.
	thumbCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if svg, thumbErr := s.generator.GenerateThumbnail(thumbCtx, wf); thumbErr == nil {
		wf.ThumbnailSVG = svg
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wf)
}

// generateWorkflowThumbnail generates (or re-generates) the thumbnail SVG for
// an existing workflow, saves it back to the repository, and returns it.
// POST /api/workflows/{name}/thumbnail
func (s *Server) generateWorkflowThumbnail(w http.ResponseWriter, r *http.Request) {
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	name := chi.URLParam(r, "name")
	wf, err := s.repo.Get(r.Context(), name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "workflow not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thumbCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	svg, err := s.generator.GenerateThumbnail(thumbCtx, wf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wf.ThumbnailSVG = svg
	// Best-effort save — if this fails the SVG is still returned to the caller.
	_ = s.repo.Update(r.Context(), name, wf)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"thumbnail_svg": svg})
}

// generatePipelineThumbnail generates (or re-generates) the thumbnail SVG for
// an existing pipeline, saves it back, and returns it.
// POST /api/pipelines/{id}/thumbnail
func (s *Server) generatePipelineThumbnail(w http.ResponseWriter, r *http.Request) {
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	p, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	thumbCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	svg, err := s.generator.GeneratePipelineThumbnail(thumbCtx, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.ThumbnailSVG = svg
	// Best-effort save — SVG is still returned even if save fails.
	_ = s.pipelineSvc.Update(r.Context(), p)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"thumbnail_svg": svg})
}
