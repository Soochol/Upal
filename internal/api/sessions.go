package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createNewSession(w http.ResponseWriter, r *http.Request) {
	var sess upal.Session
	if !decodeJSON(w, r, &sess) {
		return
	}
	if sess.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	created, err := s.sessionSvc.Create(r.Context(), &sess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusCreated, created)
}

func (s *Server) listNewSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.sessionSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, orEmpty(sessions))
}

func (s *Server) getNewSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.sessionSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sess)
}

func (s *Server) updateNewSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sess upal.Session
	if !decodeJSON(w, r, &sess) {
		return
	}
	sess.ID = id
	if err := s.sessionSvc.Update(r.Context(), &sess); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, sess)
}

func (s *Server) deleteNewSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.runSvc != nil {
		if err := s.runSvc.DeleteRunsBySession(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := s.sessionSvc.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) activateNewSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.sessionSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if sess.Status != upal.SessionStatusDraft {
		http.Error(w, "only draft sessions can be activated", http.StatusConflict)
		return
	}
	sess.Status = upal.SessionStatusActive
	if err := s.sessionSvc.Update(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, sess)
}

func (s *Server) deactivateNewSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := s.sessionSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if sess.Status != upal.SessionStatusActive {
		http.Error(w, "only active sessions can be deactivated", http.StatusConflict)
		return
	}
	sess.Status = upal.SessionStatusDraft
	if err := s.sessionSvc.Update(r.Context(), sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, sess)
}
