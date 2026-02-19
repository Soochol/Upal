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

// ConfigureNodeRequest is the JSON body for AI-assisted node configuration.
type ConfigureNodeRequest struct {
	NodeType      string            `json:"node_type"`
	NodeID        string            `json:"node_id"`
	CurrentConfig map[string]any    `json:"current_config"`
	Label         string            `json:"label"`
	Description   string            `json:"description"`
	Message       string            `json:"message"`
	Model         string            `json:"model,omitempty"`
	Thinking      bool              `json:"thinking,omitempty"`
	History       []ConfigChatMsg   `json:"history,omitempty"`
	UpstreamNodes []ConfigNodeInfo  `json:"upstream_nodes"`
}

// ConfigChatMsg represents a single message in the configuration chat history.
type ConfigChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ConfigNodeInfo describes an upstream node for context.
type ConfigNodeInfo struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

// ConfigureNodeResponse is the AI-generated configuration update.
type ConfigureNodeResponse struct {
	Config      map[string]any `json:"config"`
	Label       string         `json:"label,omitempty"`
	Description string         `json:"description,omitempty"`
	Explanation string         `json:"explanation"`
}

// configureBasePrompt is the common system prompt for node configuration.
// Node-type-specific guidance is dynamically appended from the skills registry.
var configureBasePrompt = `You are an AI assistant that fully configures nodes in the Upal visual workflow platform.
When the user describes what a node should do, you MUST fill in ALL relevant config fields — not just one or two.
Be proactive: infer and set every field that makes sense given the user's description.

You MUST also set "label" (short name for the node) and "description" (brief explanation of its purpose).

Template syntax: {{node_id}} references the output of an upstream node at runtime. This is how data flows between nodes in the DAG.

IMPORTANT RULES:
1. ALWAYS set label and description based on the user's intent.
2. CRITICAL — upstream node references: When upstream nodes exist, you MUST use {{node_id}} template references to receive their output. NEVER write hardcoded placeholder text like "다음 내용을 분석해줘: [여기에 입력]" — instead write "다음 내용을 분석해줘:\n\n{{upstream_node_id}}". The {{node_id}} gets replaced with the actual upstream node's output at runtime.
3. Fill in ALL fields comprehensively — do not leave fields empty when you can infer reasonable values.

Return JSON format:
{"config": {ALL relevant fields}, "label": "descriptive name", "description": "what this node does", "explanation": "1-line summary of fields changed, e.g. 'Set model, wrote persona prompt, added upstream reference'"}

Return ONLY valid JSON, no markdown fences, no extra text.`

func (s *Server) configureNode(w http.ResponseWriter, r *http.Request) {
	var req ConfigureNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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

	// Use the requested model if provided, otherwise fall back to the default generator.
	llm := s.generator.LLM()
	modelName := s.generator.Model()
	if req.Model != "" {
		if resolved, ok := s.resolveModel(req.Model); ok {
			llm = resolved.llm
			modelName = resolved.model
		}
	}

	// Build context for the LLM
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

	// Build contents: system instruction + history + current user message
	var contents []*genai.Content
	for _, h := range req.History {
		switch h.Role {
		case "user":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleUser))
		case "assistant":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleModel))
		}
	}
	contents = append(contents, genai.NewContentFromText(contextMsg, genai.RoleUser))

	// Build system prompt: base prompt + node-type-specific skill (if available).
	sysPrompt := configureBasePrompt
	if s.skills != nil {
		if nodeSkill := s.skills.Get(req.NodeType + "-node"); nodeSkill != "" {
			sysPrompt += "\n\n--- NODE TYPE GUIDE ---\n\n" + nodeSkill
		}
	}

	llmReq := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: contents,
	}

	// Pass thinking preference via context for ClaudeCodeLLM.
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

	// Extract text from response.
	text := llmutil.ExtractText(resp)

	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, text), http.StatusInternalServerError)
		return
	}

	// Use json.Decoder to parse only the first JSON value and ignore trailing text.
	var configResp ConfigureNodeResponse
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&configResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}
