package api

import (
	"encoding/json"
	"net/http"

	"github.com/soochol/upal/internal/upal"
)

// GenerateRequest is the JSON body for workflow generation from natural language.
// When ExistingWorkflow is provided, the generator operates in edit mode.
type GenerateRequest struct {
	Description      string                     `json:"description"`
	Model            string                     `json:"model"`
	ExistingWorkflow *upal.WorkflowDefinition   `json:"existing_workflow,omitempty"`
}

func (s *Server) generateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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

	// The model is already set on the generator at startup; the request model
	// field is accepted for API compatibility but not used to switch models
	// at runtime (the generator always uses its configured LLM).

	wf, err := s.generator.Generate(r.Context(), req.Description, req.ExistingWorkflow)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wf)
}
