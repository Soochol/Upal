// internal/api/pipelines.go
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createPipeline(w http.ResponseWriter, r *http.Request) {
	var p upal.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if p.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := s.pipelineSvc.Create(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Sync schedule stages: register cron jobs and fill in schedule_id.
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(r.Context(), &p); err == nil {
			_ = s.pipelineSvc.Update(r.Context(), &p)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	pipelines, err := s.pipelineSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pipelines == nil {
		pipelines = []*upal.Pipeline{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipelines)
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (s *Server) updatePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p upal.Pipeline
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	p.ID = id
	if err := s.pipelineSvc.Update(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Sync schedule stages: register cron jobs and fill in schedule_id.
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(r.Context(), &p); err == nil {
			_ = s.pipelineSvc.Update(r.Context(), &p)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func (s *Server) deletePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Clean up associated cron jobs before deleting.
	if s.schedulerSvc != nil {
		_ = s.schedulerSvc.RemovePipelineSchedules(r.Context(), id)
	}
	if err := s.pipelineSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) startPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	run, err := s.pipelineRunner.Start(r.Context(), p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if run.Status == "waiting" {
		w.WriteHeader(http.StatusAccepted)
	} else if run.Status == "failed" {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(run)
}

func (s *Server) listPipelineRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	runs, err := s.pipelineSvc.ListRuns(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if runs == nil {
		runs = []*upal.PipelineRun{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (s *Server) approvePipelineRun(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	runID := chi.URLParam(r, "runId")

	p, err := s.pipelineSvc.Get(r.Context(), pipelineID)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}

	run, err := s.pipelineSvc.GetRun(r.Context(), runID)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.PipelineID != pipelineID {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.Status != "waiting" {
		http.Error(w, "run is not waiting for approval", http.StatusBadRequest)
		return
	}

	run.Status = "running"
	if err := s.pipelineSvc.UpdateRun(r.Context(), run); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode the response before launching the goroutine so that Resume's
	// mutations to run.StageResults do not race with json.Encode.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(run)

	go func() {
		if err := s.pipelineRunner.Resume(context.Background(), p, run); err != nil {
			slog.Error("pipeline resume failed", "run_id", run.ID, "pipeline_id", p.ID, "error", err)
		}
	}()
}

func (s *Server) listPipelineTriggers(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]*upal.Trigger{})
		return
	}

	id := chi.URLParam(r, "id")
	triggers, err := s.triggerRepo.ListByPipeline(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if triggers == nil {
		triggers = []*upal.Trigger{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(triggers)
}

func (s *Server) rejectPipelineRun(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	runID := chi.URLParam(r, "runId")

	run, err := s.pipelineSvc.GetRun(r.Context(), runID)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.PipelineID != pipelineID {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if run.Status != "waiting" {
		http.Error(w, "run is not waiting for approval", http.StatusBadRequest)
		return
	}

	now := time.Now()
	run.Status = "failed"
	run.CompletedAt = &now
	if result, ok := run.StageResults[run.CurrentStage]; ok {
		result.Status = "failed"
		result.Error = "rejected by user"
		result.CompletedAt = &now
	}
	if err := s.pipelineSvc.UpdateRun(r.Context(), run); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}
