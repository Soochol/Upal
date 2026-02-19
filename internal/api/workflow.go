package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// WorkflowStore provides in-memory workflow CRUD.
// It also serves as the backend for run.go's workflow lookup.
type WorkflowStore struct {
	mu        sync.RWMutex
	workflows map[string]*upal.WorkflowDefinition
}

func NewWorkflowStore() *WorkflowStore {
	return &WorkflowStore{workflows: make(map[string]*upal.WorkflowDefinition)}
}

func (ws *WorkflowStore) Get(name string) (*upal.WorkflowDefinition, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	wf, ok := ws.workflows[name]
	return wf, ok
}

func (ws *WorkflowStore) Put(wf *upal.WorkflowDefinition) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.workflows[wf.Name] = wf
}

func (ws *WorkflowStore) Delete(name string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.workflows, name)
}

func (ws *WorkflowStore) List() []*upal.WorkflowDefinition {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	result := make([]*upal.WorkflowDefinition, 0, len(ws.workflows))
	for _, wf := range ws.workflows {
		result = append(result, wf)
	}
	return result
}

// --- HTTP handlers ---

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Always keep in-memory store updated (used by runner for fast lookups).
	s.workflows.Put(&wf)

	if s.db != nil {
		if _, err := s.db.CreateWorkflow(r.Context(), &wf); err != nil {
			slog.Warn("db create workflow failed, in-memory only", "err", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	// Prefer DB when available for durable listing.
	if s.db != nil {
		rows, err := s.db.ListWorkflows(r.Context())
		if err == nil {
			result := make([]*upal.WorkflowDefinition, len(rows))
			for i := range rows {
				result[i] = &rows[i].Definition
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
			return
		}
		slog.Warn("db list workflows failed, falling back to in-memory", "err", err)
	}

	result := s.workflows.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	// Try in-memory first (fast path, also covers newly created workflows).
	wf, ok := s.workflows.Get(name)
	if ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wf)
		return
	}

	// Fall back to DB if available.
	if s.db != nil {
		row, err := s.db.GetWorkflow(r.Context(), name)
		if err == nil {
			s.workflows.Put(&row.Definition) // cache in memory
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(row.Definition)
			return
		}
	}

	http.Error(w, "workflow not found", http.StatusNotFound)
}

func (s *Server) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	wf.Name = name
	s.workflows.Put(&wf)

	if s.db != nil {
		if err := s.db.UpdateWorkflow(r.Context(), name, &wf); err != nil {
			slog.Warn("db update workflow failed, in-memory only", "err", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wf)
}

func (s *Server) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	s.workflows.Delete(name)

	if s.db != nil {
		if err := s.db.DeleteWorkflow(r.Context(), name); err != nil {
			slog.Warn("db delete workflow failed", "err", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
