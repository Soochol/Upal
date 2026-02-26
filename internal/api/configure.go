package api

import (
	"fmt"
	"net/http"

	"github.com/soochol/upal/internal/generate"
)

type ConfigureNodeRequest struct {
	NodeType      string                   `json:"node_type"`
	NodeID        string                   `json:"node_id"`
	CurrentConfig map[string]any           `json:"current_config"`
	Label         string                   `json:"label"`
	Description   string                   `json:"description"`
	Message       string                   `json:"message"`
	Model         string                   `json:"model,omitempty"`
	Thinking      bool                     `json:"thinking,omitempty"`
	History       []generate.ChatMessage   `json:"history,omitempty"`
	UpstreamNodes []ConfigNodeInfo         `json:"upstream_nodes"`
}

type ConfigNodeInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

func (s *Server) configureNode(w http.ResponseWriter, r *http.Request) {
	var req ConfigureNodeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if req.NodeType == "" {
		http.Error(w, "node_type is required", http.StatusBadRequest)
		return
	}
	if s.generator == nil {
		http.Error(w, "generator not configured (no providers available)", http.StatusServiceUnavailable)
		return
	}

	upstream := make([]generate.UpstreamNodeInfo, len(req.UpstreamNodes))
	for i, u := range req.UpstreamNodes {
		upstream[i] = generate.UpstreamNodeInfo{ID: u.ID, Type: u.Type, Label: u.Label}
	}

	out, err := s.generator.ConfigureNode(r.Context(), generate.ConfigureNodeInput{
		NodeType:      req.NodeType,
		NodeID:        req.NodeID,
		CurrentConfig: req.CurrentConfig,
		Label:         req.Label,
		Description:   req.Description,
		Message:       req.Message,
		Model:         req.Model,
		Thinking:      req.Thinking,
		History:       req.History,
		UpstreamNodes: upstream,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("configure node: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, out)
}
