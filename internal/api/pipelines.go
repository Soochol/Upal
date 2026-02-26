package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createPipeline(w http.ResponseWriter, r *http.Request) {
	var p upal.Pipeline
	if !decodeJSON(w, r, &p) {
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
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(r.Context(), &p); err == nil {
			_ = s.pipelineSvc.Update(r.Context(), &p)
		}
	}
	writeJSONStatus(w, http.StatusCreated, p)
}

func (s *Server) listPipelines(w http.ResponseWriter, r *http.Request) {
	pipelines, err := s.pipelineSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(pipelines))
}

func (s *Server) getPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.pipelineSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "pipeline not found", http.StatusNotFound)
		return
	}
	writeJSON(w, p)
}

func (s *Server) updatePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p upal.Pipeline
	if !decodeJSON(w, r, &p) {
		return
	}
	p.ID = id
	if err := s.pipelineSvc.Update(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.schedulerSvc != nil {
		if err := s.schedulerSvc.SyncPipelineSchedules(r.Context(), &p); err == nil {
			_ = s.pipelineSvc.Update(r.Context(), &p)
		}
	}
	writeJSON(w, p)
}

func (s *Server) deletePipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.schedulerSvc != nil {
		_ = s.schedulerSvc.RemovePipelineSchedules(r.Context(), id)
	}
	if s.contentSvc != nil {
		if err := s.contentSvc.DeleteSessionsByPipeline(r.Context(), id); err != nil {
			slog.Error("failed to clean up sessions for pipeline", "pipeline_id", id, "err", err)
		}
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

	run, err := s.pipelineRunner.Start(r.Context(), p, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	switch run.Status {
	case upal.PipelineRunWaiting:
		status = http.StatusAccepted
	case upal.PipelineRunFailed:
		status = http.StatusInternalServerError
	}
	writeJSONStatus(w, status, run)
}

func (s *Server) listPipelineRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	runs, err := s.pipelineSvc.ListRuns(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(runs))
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
	if run.Status != upal.PipelineRunWaiting {
		http.Error(w, "run is not waiting for approval", http.StatusBadRequest)
		return
	}

	run.Status = upal.PipelineRunRunning
	if err := s.pipelineSvc.UpdateRun(r.Context(), run); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode the response before launching the goroutine so that Resume's
	// mutations to run.StageResults do not race with json.Encode.
	writeJSONStatus(w, http.StatusAccepted, run)

	go func() {
		if err := s.pipelineRunner.Resume(context.Background(), p, run); err != nil {
			slog.Error("pipeline resume failed", "run_id", run.ID, "pipeline_id", p.ID, "error", err)
		}
	}()
}

func (s *Server) listPipelineTriggers(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		writeJSON(w, []*upal.Trigger{})
		return
	}

	id := chi.URLParam(r, "id")
	triggers, err := s.triggerRepo.ListByPipeline(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(triggers))
}

func (s *Server) rejectPipelineRun(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")
	runID := chi.URLParam(r, "runId")

	run, err := s.pipelineSvc.RejectRun(r.Context(), pipelineID, runID)
	if err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, run)
}
