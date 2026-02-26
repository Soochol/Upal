package api

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

type GenerateRequest struct {
	Description      string                   `json:"description"`
	Model            string                   `json:"model"`
	ExistingWorkflow *upal.WorkflowDefinition `json:"existing_workflow,omitempty"`
}

type GeneratePipelineRequest struct {
	Description      string         `json:"description"`
	ExistingPipeline *upal.Pipeline `json:"existing_pipeline,omitempty"`
}

func (s *Server) generatePipeline(w http.ResponseWriter, r *http.Request) {
	var req GeneratePipelineRequest
	if !decodeJSON(w, r, &req) {
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

	var workflowSummaries []generate.WorkflowSummary
	if wfs, listErr := s.repo.List(r.Context()); listErr == nil {
		workflowSummaries = generate.BuildWorkflowSummaries(wfs)
	}

	var pipelineSummaries []generate.PipelineSummary
	if pipes, listErr := s.pipelineSvc.List(r.Context()); listErr == nil {
		pipelineSummaries = generate.BuildPipelineSummaries(pipes)
	}

	bundle, err := s.generator.GeneratePipelineBundle(r.Context(), req.Description, req.ExistingPipeline, workflowSummaries, pipelineSummaries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i := range bundle.Workflows {
		if err := s.repo.Create(r.Context(), &bundle.Workflows[i]); err != nil {
			log.Printf("generatePipeline: save workflow %q: %v", bundle.Workflows[i].Name, err)
		}
	}

	thumbCtx, cancel := context.WithTimeout(r.Context(), s.thumbnailTimeoutOrDefault())
	defer cancel()
	if svg, thumbErr := s.generator.GeneratePipelineThumbnail(thumbCtx, &bundle.Pipeline); thumbErr == nil {
		bundle.Pipeline.ThumbnailSVG = svg
	}

	writeJSON(w, bundle)
}

func (s *Server) generateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if !decodeJSON(w, r, &req) {
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

	var workflowSummaries []generate.WorkflowSummary
	if wfs, listErr := s.repo.List(r.Context()); listErr == nil {
		workflowSummaries = generate.BuildWorkflowSummaries(wfs)
	}

	wf, err := s.generator.Generate(r.Context(), req.Description, req.ExistingWorkflow, workflowSummaries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thumbCtx, cancel := context.WithTimeout(r.Context(), s.thumbnailTimeoutOrDefault())
	defer cancel()
	if svg, thumbErr := s.generator.GenerateThumbnail(thumbCtx, wf); thumbErr == nil {
		wf.ThumbnailSVG = svg
	}

	writeJSON(w, wf)
}

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

	thumbCtx, cancel := context.WithTimeout(r.Context(), s.thumbnailTimeoutOrDefault())
	defer cancel()
	svg, err := s.generator.GenerateThumbnail(thumbCtx, wf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wf.ThumbnailSVG = svg
	_ = s.repo.Update(r.Context(), name, wf)

	writeJSON(w, map[string]string{"thumbnail_svg": svg})
}

func (s *Server) backfillDescriptions(w http.ResponseWriter, r *http.Request) {
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	workflowsUpdated := 0
	nodesUpdated := 0
	stagesUpdated := 0

	if wfs, err := s.repo.List(r.Context()); err == nil {
		for _, wf := range s.generator.BackfillWorkflowDescriptions(r.Context(), wfs) {
			if saveErr := s.repo.Update(r.Context(), wf.Name, wf); saveErr != nil {
				log.Printf("backfill: save workflow %q failed: %v", wf.Name, saveErr)
				continue
			}
			workflowsUpdated++
		}
		for _, wf := range s.generator.BackfillNodeDescriptions(r.Context(), wfs) {
			if saveErr := s.repo.Update(r.Context(), wf.Name, wf); saveErr != nil {
				log.Printf("backfill: save workflow nodes %q failed: %v", wf.Name, saveErr)
				continue
			}
			nodesUpdated++
		}
	}

	if pipelines, err := s.pipelineSvc.List(r.Context()); err == nil {
		for _, p := range pipelines {
			if generate.BackfillStageDescriptions(p) {
				if saveErr := s.pipelineSvc.Update(r.Context(), p); saveErr != nil {
					log.Printf("backfill: save pipeline %q failed: %v", p.ID, saveErr)
					continue
				}
				stagesUpdated++
			}
		}
	}

	writeJSON(w, map[string]int{
		"workflows_updated": workflowsUpdated,
		"nodes_updated":     nodesUpdated,
		"stages_updated":    stagesUpdated,
	})
}

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

	thumbCtx, cancel := context.WithTimeout(r.Context(), s.thumbnailTimeoutOrDefault())
	defer cancel()
	svg, err := s.generator.GeneratePipelineThumbnail(thumbCtx, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.ThumbnailSVG = svg
	_ = s.pipelineSvc.Update(r.Context(), p)

	writeJSON(w, map[string]string{"thumbnail_svg": svg})
}
