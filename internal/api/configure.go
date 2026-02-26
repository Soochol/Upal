package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

type ConfigureNodeRequest struct {
	NodeType      string           `json:"node_type"`
	NodeID        string           `json:"node_id"`
	CurrentConfig map[string]any   `json:"current_config"`
	Label         string           `json:"label"`
	Description   string           `json:"description"`
	Message       string           `json:"message"`
	Model         string           `json:"model,omitempty"`
	Thinking      bool             `json:"thinking,omitempty"`
	History       []ConfigChatMsg  `json:"history,omitempty"`
	UpstreamNodes []ConfigNodeInfo `json:"upstream_nodes"`
}

type ConfigNodeInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type ConfigureNodeResponse struct {
	Config      map[string]any `json:"config"`
	Label       string         `json:"label,omitempty"`
	Description string         `json:"description,omitempty"`
	Explanation string         `json:"explanation"`
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

	llm, modelName := s.resolveLLM(req.Model)

	configJSON, _ := json.Marshal(req.CurrentConfig)

	upstreamList := "none"
	if len(req.UpstreamNodes) > 0 {
		var parts []string
		for _, u := range req.UpstreamNodes {
			parts = append(parts, fmt.Sprintf("- %s (type=%s, label=%q)", u.ID, u.Type, u.Label))
		}
		upstreamList = strings.Join(parts, "\n")
	}

	contextMsg := fmt.Sprintf(
		"Current node: type=%s, id=%s, label=%q, description=%q\nCurrent config: %s\nUpstream nodes:\n%s\n\nUser request: %s",
		req.NodeType, req.NodeID, req.Label, req.Description, string(configJSON), upstreamList, req.Message,
	)

	contents := buildChatHistory(req.History)
	contents = append(contents, genai.NewContentFromText(contextMsg, genai.RoleUser))

	sysPrompt := ""
	if s.skills != nil {
		sysPrompt = s.skills.GetPrompt("node-configure")
		if nodeSkill := s.skills.Get(req.NodeType + "-node"); nodeSkill != "" {
			sysPrompt += "\n\n--- NODE TYPE GUIDE ---\n\n" + nodeSkill
		}
	}

	sysPrompt = s.appendModelCatalog(sysPrompt, modelName)
	sysPrompt += "\nIMPORTANT: ONLY use models from this list. Match model category to the node's purpose."

	llmReq := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: contents,
	}

	ctx := upalmodel.WithThinking(r.Context(), req.Thinking)

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			http.Error(w, fmt.Sprintf("LLM call failed: %v", err), http.StatusInternalServerError)
			return
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		http.Error(w, "empty response from LLM", http.StatusInternalServerError)
		return
	}

	text := llmutil.ExtractText(resp)
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, text), http.StatusInternalServerError)
		return
	}

	var configResp ConfigureNodeResponse
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&configResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}

	writeJSON(w, configResp)
}
