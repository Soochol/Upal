package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// createSchedule creates a new cron schedule for a workflow.
// POST /api/schedules
func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		http.Error(w, "scheduler not available", http.StatusServiceUnavailable)
		return
	}

	var schedule upal.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if schedule.WorkflowName == "" || schedule.CronExpr == "" {
		http.Error(w, "workflow_name and cron_expr are required", http.StatusBadRequest)
		return
	}

	if err := s.schedulerSvc.AddSchedule(r.Context(), &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(schedule)
}

// listSchedules returns all schedules.
// GET /api/schedules
func (s *Server) listSchedules(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}

	schedules, err := s.schedulerSvc.ListSchedules(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedules)
}

// getSchedule returns a single schedule.
// GET /api/schedules/{id}
func (s *Server) getSchedule(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		http.Error(w, "scheduler not available", http.StatusNotFound)
		return
	}

	id := chi.URLParam(r, "id")
	schedule, err := s.schedulerSvc.GetSchedule(r.Context(), id)
	if err != nil {
		http.Error(w, "schedule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// updateSchedule modifies an existing schedule.
// PUT /api/schedules/{id}
func (s *Server) updateSchedule(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		http.Error(w, "scheduler not available", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")

	var schedule upal.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	schedule.ID = id

	if err := s.schedulerSvc.UpdateSchedule(r.Context(), &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// deleteSchedule removes a schedule.
// DELETE /api/schedules/{id}
func (s *Server) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		http.Error(w, "scheduler not available", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if err := s.schedulerSvc.RemoveSchedule(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// pauseSchedule temporarily disables a schedule.
// POST /api/schedules/{id}/pause
func (s *Server) pauseSchedule(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		http.Error(w, "scheduler not available", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if err := s.schedulerSvc.PauseSchedule(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// resumeSchedule re-enables a paused schedule.
// POST /api/schedules/{id}/resume
func (s *Server) resumeSchedule(w http.ResponseWriter, r *http.Request) {
	if s.schedulerSvc == nil {
		http.Error(w, "scheduler not available", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if err := s.schedulerSvc.ResumeSchedule(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
