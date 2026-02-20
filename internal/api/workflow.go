package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// --- HTTP handlers ---

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.repo.Create(r.Context(), &wf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	result, err := s.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result == nil {
		result = []*upal.WorkflowDefinition{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	wf, err := s.repo.Get(r.Context(), name)
	if err != nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.repo.Update(r.Context(), name, &wf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := s.repo.Delete(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
