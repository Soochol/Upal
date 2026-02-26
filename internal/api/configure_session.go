package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/generate"
)

func (s *Server) configureSession(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	var req ConfigurePipelineRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	// Load session to resolve pipeline context.
	sess, err := s.contentSvc.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("session not found: %v", err), http.StatusNotFound)
		return
	}

	input := s.buildConfigureInput(r, req, sess.PipelineID)
	input.Session = &generate.SessionContext{
		Name:       sess.Name,
		Status:     string(sess.Status),
		IsTemplate: sess.IsTemplate,
	}

	out, err := s.generator.ConfigurePipeline(r.Context(), input)
	if err != nil {
		http.Error(w, fmt.Sprintf("configure session: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, out)
}
