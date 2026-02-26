package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createTrigger(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		http.Error(w, "triggers not available", http.StatusServiceUnavailable)
		return
	}

	var trigger upal.Trigger
	if !decodeJSON(w, r, &trigger) {
		return
	}
	if trigger.WorkflowName == "" && trigger.PipelineID == "" {
		http.Error(w, "workflow_name or pipeline_id is required", http.StatusBadRequest)
		return
	}

	trigger.ID = upal.GenerateID("trig")
	trigger.Type = upal.TriggerWebhook
	trigger.Enabled = true
	trigger.CreatedAt = time.Now()

	if trigger.Config.Secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		trigger.Config.Secret = "whsec_" + hex.EncodeToString(b)
	}

	if err := s.triggerRepo.Create(r.Context(), &trigger); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONStatus(w, http.StatusCreated, map[string]any{
		"trigger":     trigger,
		"webhook_url": "/api/hooks/" + trigger.ID,
	})
}

func (s *Server) listTriggers(w http.ResponseWriter, r *http.Request) {
	if s.triggerRepo == nil {
		writeJSON(w, []any{})
		return
	}

	name := chi.URLParam(r, "name")
	triggers, err := s.triggerRepo.ListByWorkflow(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(triggers))
}

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
