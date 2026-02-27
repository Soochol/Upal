package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createAIProvider(w http.ResponseWriter, r *http.Request) {
	var p upal.AIProvider
	if !decodeJSON(w, r, &p) {
		return
	}
	if err := p.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.aiProviderSvc.Create(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusCreated, p.Safe())
}

func (s *Server) listAIProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.aiProviderSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, providers)
}

func (s *Server) updateAIProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var patch upal.AIProvider
	if !decodeJSON(w, r, &patch) {
		return
	}
	// Load existing provider and merge non-zero patch fields.
	existing, err := s.aiProviderSvc.Resolve(r.Context(), id)
	if err != nil {
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	if patch.Name != "" {
		existing.Name = patch.Name
	}
	if patch.Model != "" {
		existing.Model = patch.Model
	}
	if patch.APIKey != "" {
		existing.APIKey = patch.APIKey
	}
	if patch.IsDefault {
		existing.IsDefault = true
	}
	if err := existing.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.aiProviderSvc.Update(r.Context(), existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, existing.Safe())
}

func (s *Server) deleteAIProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.aiProviderSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setAIProviderDefault(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.aiProviderSvc.SetDefault(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Return updated provider list so frontend can refresh
	providers, err := s.aiProviderSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, providers)
}

func (s *Server) getAIProviderDefaults(w http.ResponseWriter, r *http.Request) {
	providers, err := s.aiProviderSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defaults := make(map[string]upal.AIProviderSafe)
	for _, p := range providers {
		if p.IsDefault {
			defaults[string(p.Category)] = p
		}
	}
	writeJSON(w, defaults)
}
