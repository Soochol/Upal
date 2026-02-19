package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

var nameSystemPrompt = `You name workflows. Given a workflow definition JSON, produce a short descriptive slug-style name.
Rules:
- lowercase letters and hyphens only
- max 4 words
- descriptive of what the workflow does
Examples: "content-pipeline", "multi-model-compare", "code-review-agent", "research-summarizer"
Respond with ONLY a JSON object: {"name": "the-slug-name"}`

var slugRegexp = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRegexp.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "untitled-workflow"
	}
	return s
}

func (s *Server) suggestWorkflowName(w http.ResponseWriter, r *http.Request) {
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if s.generator == nil {
		// Fallback: build name from node labels
		name := buildFallbackName(wf)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": name})
		return
	}

	wfJSON, _ := json.MarshalIndent(wf, "", "  ")

	llmReq := &adkmodel.LLMRequest{
		Model: s.generator.Model(),
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(nameSystemPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(string(wfJSON), genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range s.generator.LLM().GenerateContent(r.Context(), llmReq, false) {
		if err != nil {
			// Fallback on LLM error
			name := buildFallbackName(wf)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"name": name})
			return
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		name := buildFallbackName(wf)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": name})
		return
	}

	// Extract text from response and strip markdown fences
	text := llmutil.ExtractText(resp)
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		name := buildFallbackName(wf)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": name})
		return
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&result); err != nil || result.Name == "" {
		name := buildFallbackName(wf)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"name": name})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"name": sanitizeSlug(result.Name)})
}

// buildFallbackName creates a deterministic name from node labels when LLM is unavailable.
func buildFallbackName(wf upal.WorkflowDefinition) string {
	if len(wf.Nodes) == 0 {
		return "untitled-workflow"
	}
	var parts []string
	for _, n := range wf.Nodes {
		if n.Type == "agent" {
			label, ok := n.Config["label"].(string)
			if !ok || label == "" {
				continue
			}
			parts = append(parts, strings.ToLower(label))
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("workflow-%d-nodes", len(wf.Nodes))
	}
	name := strings.Join(parts, "-")
	return sanitizeSlug(name)
}
