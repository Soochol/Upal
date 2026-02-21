package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) createConnection(w http.ResponseWriter, r *http.Request) {
	var conn upal.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if conn.Name == "" || conn.Type == "" {
		http.Error(w, "name and type are required", http.StatusBadRequest)
		return
	}
	if err := s.connectionSvc.Create(r.Context(), &conn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conn.Safe())
}

func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := s.connectionSvc.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conns)
}

func (s *Server) getConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	conn, err := s.connectionSvc.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "connection not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conn.Safe())
}

func (s *Server) updateConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var conn upal.Connection
	if err := json.NewDecoder(r.Body).Decode(&conn); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	conn.ID = id
	if err := s.connectionSvc.Update(r.Context(), &conn); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conn.Safe())
}

func (s *Server) deleteConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.connectionSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
