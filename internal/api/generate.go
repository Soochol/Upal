package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

type GenerateRequest struct {
	Description      string                   `json:"description"`
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

	// Gather summaries using request context (fast DB reads).
	var workflowSummaries []generate.WorkflowSummary
	if wfs, listErr := s.repo.List(r.Context()); listErr == nil {
		workflowSummaries = generate.BuildWorkflowSummaries(wfs)
	}

	var pipelineSummaries []generate.PipelineSummary
	if pipes, listErr := s.pipelineSvc.List(r.Context()); listErr == nil {
		pipelineSummaries = generate.BuildPipelineSummaries(pipes)
	}

	genID := upal.GenerateID("gen")
	s.generationManager.Register(genID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		bundle, err := s.generator.GeneratePipelineBundle(ctx, req.Description, req.ExistingPipeline, workflowSummaries, pipelineSummaries)
		if err != nil {
			slog.Error("generatePipeline: generation failed", "gen_id", genID, "err", err)
			s.generationManager.Fail(genID, err.Error())
			return
		}

		for i := range bundle.Workflows {
			if err := s.repo.Create(ctx, &bundle.Workflows[i]); err != nil {
				slog.Error("generatePipeline: save workflow failed", "gen_id", genID, "workflow", bundle.Workflows[i].Name, "err", err)
			}
		}

		thumbCtx, thumbCancel := context.WithTimeout(ctx, s.thumbnailTimeoutOrDefault())
		defer thumbCancel()
		if svg, thumbErr := s.generator.GeneratePipelineThumbnail(thumbCtx, &bundle.Pipeline); thumbErr == nil {
			bundle.Pipeline.ThumbnailSVG = svg
		}

		slog.Info("generatePipeline: completed", "gen_id", genID)
		s.generationManager.Complete(genID, bundle)
	}()

	writeJSONStatus(w, http.StatusAccepted, map[string]string{"generation_id": genID})
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

	// Gather summaries using request context (fast DB reads).
	var workflowSummaries []generate.WorkflowSummary
	if wfs, listErr := s.repo.List(r.Context()); listErr == nil {
		workflowSummaries = generate.BuildWorkflowSummaries(wfs)
	}

	genID := upal.GenerateID("gen")
	s.generationManager.Register(genID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		wf, err := s.generator.Generate(ctx, req.Description, req.ExistingWorkflow, workflowSummaries)
		if err != nil {
			slog.Error("generateWorkflow: generation failed", "gen_id", genID, "err", err)
			s.generationManager.Fail(genID, err.Error())
			return
		}

		thumbCtx, thumbCancel := context.WithTimeout(ctx, s.thumbnailTimeoutOrDefault())
		defer thumbCancel()
		if svg, thumbErr := s.generator.GenerateThumbnail(thumbCtx, wf); thumbErr == nil {
			wf.ThumbnailSVG = svg
		}

		slog.Info("generateWorkflow: completed", "gen_id", genID, "workflow", wf.Name)
		s.generationManager.Complete(genID, wf)
	}()

	writeJSONStatus(w, http.StatusAccepted, map[string]string{"generation_id": genID})
}

func (s *Server) getGeneration(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	entry, ok := s.generationManager.Get(id)
	if !ok {
		http.Error(w, "generation not found", http.StatusNotFound)
		return
	}
	writeJSON(w, entry)
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
				slog.Warn("backfill: save workflow failed", "name", wf.Name, "err", saveErr)
				continue
			}
			workflowsUpdated++
		}
		for _, wf := range s.generator.BackfillNodeDescriptions(r.Context(), wfs) {
			if saveErr := s.repo.Update(r.Context(), wf.Name, wf); saveErr != nil {
				slog.Warn("backfill: save workflow nodes failed", "name", wf.Name, "err", saveErr)
				continue
			}
			nodesUpdated++
		}
	}

	if pipelines, err := s.pipelineSvc.List(r.Context()); err == nil {
		for _, p := range pipelines {
			if generate.BackfillStageDescriptions(p) {
				if saveErr := s.pipelineSvc.Update(r.Context(), p); saveErr != nil {
					slog.Warn("backfill: save pipeline failed", "id", p.ID, "err", saveErr)
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
