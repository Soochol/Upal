package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// validateWorkflowTools checks that every tool-type node references a tool
// that is actually registered (custom or native). Returns a descriptive error
// so the caller can return 400 instead of letting execution fail at runtime.
func (s *Server) validateWorkflowTools(wf *upal.WorkflowDefinition) error {
	if s.toolReg == nil {
		return nil
	}
	for _, n := range wf.Nodes {
		if n.Type != upal.NodeTypeTool {
			continue
		}
		toolName, _ := n.Config["tool"].(string)
		if toolName == "" {
			return fmt.Errorf("tool node %q: missing required config field \"tool\"", n.ID)
		}
		_, isCustom := s.toolReg.Get(toolName)
		isNative := s.toolReg.IsNative(toolName)
		if !isCustom && !isNative {
			return fmt.Errorf("tool node %q: unknown tool %q (available: use GET /api/tools to list registered tools)", n.ID, toolName)
		}
	}
	return nil
}

// --- HTTP handlers ---

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.validateWorkflowTools(&wf); err != nil {
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

	if err := s.validateWorkflowTools(&wf); err != nil {
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
