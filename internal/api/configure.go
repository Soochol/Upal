package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

const configureSystemPrompt = `You are an AI assistant that fully configures nodes in the Upal visual workflow platform.
When the user describes what a node should do, you MUST fill in ALL relevant config fields — not just one or two.
Be proactive: infer and set every field that makes sense given the user's description.

Node types and their configurable fields (set ALL that apply):
- "agent": model (format: "provider/model-name", e.g. "anthropic/claude-sonnet-4-20250514" or "gemini/gemini-2.0-flash"), system_prompt (expert persona — see PERSONA FRAMEWORK below), prompt (the user message template — use {{node_id}} to reference upstream node outputs), max_turns (integer, default 1)
- "input": placeholder (string hint shown to user)
- "tool": tool_name (string), input (string with {{node_id}} refs)
- "output": (no configurable fields)
- "external": endpoint_url (string), timeout (integer seconds)

PERSONA FRAMEWORK — For "agent" nodes, the system_prompt MUST be a rich expert persona with these sections:

1. ROLE — Define a specific expert identity. Not "You are a helpful assistant" but a concrete specialist.
   Example: "You are a senior tech blog editor with deep expertise in developer content strategy."

2. EXPERTISE — List 3-5 core competencies the agent excels at.
   Example: "Your expertise includes: SEO-optimized writing, audience engagement, technical accuracy, narrative structure."

3. STYLE — Specify tone and communication approach appropriate for the task.
   Example: "Write in a conversational yet authoritative tone. Use short paragraphs and clear subheadings."

4. CONSTRAINTS — Set clear rules and boundaries for the agent's behavior.
   Example: "Always include a strong opening hook. Keep paragraphs under 4 sentences. Never fabricate data."

5. OUTPUT FORMAT — Define the expected structure of the agent's output.
   Example: "Output in Markdown with H2/H3 heading hierarchy. End with a summary or call-to-action."

Combine all sections into a single cohesive system_prompt string (not with literal section headers — weave them naturally).
The persona should feel like briefing a real human expert on exactly how to perform their role.

You MUST also set "label" (short name for the node) and "description" (brief explanation of its purpose).

Template syntax: {{node_id}} references the output of an upstream node. When upstream nodes exist, use them in the prompt field.

IMPORTANT RULES:
1. For "agent" nodes: ALWAYS set model, system_prompt, prompt, and max_turns. Choose an appropriate model if the user doesn't specify one.
2. The system_prompt must follow the PERSONA FRAMEWORK — generic or shallow prompts are not acceptable.
3. ALWAYS set label and description based on the user's intent.
4. If upstream nodes exist, incorporate {{node_id}} references in the prompt.
5. Fill in ALL fields comprehensively — do not leave fields empty when you can infer reasonable values.

Return JSON format:
{"config": {ALL relevant fields}, "label": "descriptive name", "description": "what this node does", "explanation": "what you configured and why"}

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

	llm := s.generator.LLM()
	modelName := s.generator.Model()

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

	llmReq := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(configureSystemPrompt, genai.RoleUser),
		},
		Contents: contents,
	}

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(r.Context(), llmReq, false) {
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
	var text string
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			text += p.Text
		}
	}

	content := strings.TrimSpace(text)
	// Strip markdown code fences if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var configResp ConfigureNodeResponse
	if err := json.Unmarshal([]byte(content), &configResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}
