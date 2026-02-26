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
	var p upal.AIProvider
	if !decodeJSON(w, r, &p) {
		return
	}
	p.ID = id
	if err := p.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.aiProviderSvc.Update(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, p.Safe())
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
