package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/engine"
)

type WorkflowStore struct {
	mu        sync.RWMutex
	workflows map[string]*engine.WorkflowDefinition
}

func NewWorkflowStore() *WorkflowStore {
	return &WorkflowStore{workflows: make(map[string]*engine.WorkflowDefinition)}
}

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf engine.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.workflows.mu.Lock()
	s.workflows.workflows[wf.Name] = &wf
	s.workflows.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	s.workflows.mu.RLock()
	defer s.workflows.mu.RUnlock()
	result := make([]*engine.WorkflowDefinition, 0, len(s.workflows.workflows))
	for _, wf := range s.workflows.workflows {
		result = append(result, wf)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	s.workflows.mu.RLock()
	wf, ok := s.workflows.workflows[name]
	s.workflows.mu.RUnlock()
	if !ok {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var wf engine.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	wf.Name = name
	s.workflows.mu.Lock()
	s.workflows.workflows[name] = &wf
	s.workflows.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	s.workflows.mu.Lock()
	delete(s.workflows.workflows, name)
	s.workflows.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}
