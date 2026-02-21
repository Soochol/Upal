package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// listRuns returns all runs with pagination.
// GET /api/runs?limit=20&offset=0
func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	if s.runHistorySvc == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"runs": []any{}, "total": 0})
		return
	}

	limit, offset := parsePagination(r)
	status := r.URL.Query().Get("status")

	runs, total, err := s.runHistorySvc.ListAllRuns(r.Context(), limit, offset, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"runs":  runs,
		"total": total,
	})
}

// getRun returns a single run record with node-level detail.
// GET /api/runs/{id}
func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	if s.runHistorySvc == nil {
		http.Error(w, "run history not available", http.StatusNotFound)
		return
	}

	id := chi.URLParam(r, "id")
	run, err := s.runHistorySvc.GetRun(r.Context(), id)
	if err != nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// listWorkflowRuns returns runs for a specific workflow.
// GET /api/workflows/{name}/runs?limit=20&offset=0
func (s *Server) listWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	if s.runHistorySvc == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"runs": []any{}, "total": 0})
		return
	}

	name := chi.URLParam(r, "name")
	limit, offset := parsePagination(r)

	runs, total, err := s.runHistorySvc.ListRuns(r.Context(), name, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"runs":  runs,
		"total": total,
	})
}

// getSchedulerStats returns current concurrency and scheduler status.
// GET /api/scheduler/stats
func (s *Server) getSchedulerStats(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{}
	if s.limiter != nil {
		resp["concurrency"] = s.limiter.Stats()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// parsePagination extracts limit and offset query parameters with defaults.
func parsePagination(r *http.Request) (int, int) {
	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
