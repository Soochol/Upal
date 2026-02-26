package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	if s.runHistorySvc == nil {
		writeJSON(w, map[string]any{"runs": []any{}, "total": 0})
		return
	}

	limit, offset := parsePagination(r)
	status := r.URL.Query().Get("status")

	runs, total, err := s.runHistorySvc.ListAllRuns(r.Context(), limit, offset, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"runs": runs, "total": total})
}

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
	writeJSON(w, run)
}

func (s *Server) listWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	if s.runHistorySvc == nil {
		writeJSON(w, map[string]any{"runs": []any{}, "total": 0})
		return
	}

	name := chi.URLParam(r, "name")
	limit, offset := parsePagination(r)

	runs, total, err := s.runHistorySvc.ListRuns(r.Context(), name, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"runs": runs, "total": total})
}

func (s *Server) getSchedulerStats(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{}
	if s.limiter != nil {
		resp["concurrency"] = s.limiter.Stats()
	}
	writeJSON(w, resp)
}

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
