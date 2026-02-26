package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createPublishChannel(w http.ResponseWriter, r *http.Request) {
	var ch upal.PublishChannel
	if !decodeJSON(w, r, &ch) {
		return
	}
	if ch.Name == "" || ch.Type == "" {
		http.Error(w, "name and type are required", http.StatusBadRequest)
		return
	}
	if !upal.ValidChannelType(ch.Type) {
		http.Error(w, "invalid channel type", http.StatusBadRequest)
		return
	}
	ch.ID = upal.NewPublishChannelID()
	if err := s.publishChannelRepo.Create(r.Context(), &ch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusCreated, ch)
}

func (s *Server) listPublishChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.publishChannelRepo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, channels)
}

func (s *Server) getPublishChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := s.publishChannelRepo.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "publish channel not found", http.StatusNotFound)
		return
	}
	writeJSON(w, ch)
}

func (s *Server) updatePublishChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var ch upal.PublishChannel
	if !decodeJSON(w, r, &ch) {
		return
	}
	ch.ID = id
	if err := s.publishChannelRepo.Update(r.Context(), &ch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, ch)
}

func (s *Server) deletePublishChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.publishChannelRepo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
