package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

// createTrigger creates a new webhook trigger for a workflow.
// POST /api/triggers
func (s *Server) createTrigger(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		http.Error(w, "triggers not available", http.StatusServiceUnavailable)
		return
	}

	var trigger upal.Trigger
	if err := json.NewDecoder(r.Body).Decode(&trigger); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if trigger.WorkflowName == "" {
		http.Error(w, "workflow_name is required", http.StatusBadRequest)
		return
	}

	trigger.ID = upal.GenerateID("trig")
	trigger.Type = upal.TriggerWebhook
	trigger.Enabled = true
	trigger.CreatedAt = time.Now()

	// Auto-generate secret if not provided.
	if trigger.Config.Secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		trigger.Config.Secret = "whsec_" + hex.EncodeToString(b)
	}

	if err := s.triggerRepo.Create(r.Context(), &trigger); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"trigger":     trigger,
		"webhook_url": "/api/hooks/" + trigger.ID,
	})
}

// listTriggers returns triggers for a specific workflow.
// GET /api/workflows/{name}/triggers
func (s *Server) listTriggers(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]any{})
		return
	}

	name := chi.URLParam(r, "name")
	triggers, err := s.triggerRepo.ListByWorkflow(r.Context(), name)
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

// deleteTrigger removes a trigger.
// DELETE /api/triggers/{id}
func (s *Server) deleteTrigger(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		http.Error(w, "triggers not available", http.StatusServiceUnavailable)
		return
	}

	id := chi.URLParam(r, "id")
	if err := s.triggerRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
