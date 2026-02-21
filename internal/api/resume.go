package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) resumeNode(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "id")
	nodeID := chi.URLParam(r, "nodeId")

	if s.executionReg == nil {
		http.Error(w, "execution registry not configured", http.StatusServiceUnavailable)
		return
	}

	handle, ok := s.executionReg.Get(runID)
	if !ok {
		http.Error(w, "run not found or already completed", http.StatusNotFound)
		return
	}

	var payload map[string]any
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&payload)
	}
	if payload == nil {
		payload = make(map[string]any)
	}

	if err := handle.Resume(nodeID, payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}
