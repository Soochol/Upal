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
	if !decodeJSON(w, r, &wf) {
		return
	}

	respondWithName := func(name string) {
		writeJSON(w, map[string]string{"name": name})
	}

	if s.generator == nil {
		respondWithName(buildFallbackName(wf))
		return
	}

	wfJSON, _ := json.MarshalIndent(wf, "", "  ")

	llmReq := &adkmodel.LLMRequest{
		Model: s.generator.Model(),
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(s.skills.GetPrompt("workflow-name"), genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(string(wfJSON), genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range s.generator.LLM().GenerateContent(r.Context(), llmReq, false) {
		if err != nil {
			respondWithName(buildFallbackName(wf))
			return
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		respondWithName(buildFallbackName(wf))
		return
	}

	text := llmutil.ExtractText(resp)
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		respondWithName(buildFallbackName(wf))
		return
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&result); err != nil || result.Name == "" {
		respondWithName(buildFallbackName(wf))
		return
	}

	respondWithName(sanitizeSlug(result.Name))
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
	return sanitizeSlug(strings.Join(parts, "-"))
}
