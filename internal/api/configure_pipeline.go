package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

type ConfigurePipelineRequest struct {
	Message          string          `json:"message"`
	Model            string          `json:"model,omitempty"`
	Thinking         bool            `json:"thinking,omitempty"`
	History          []ConfigChatMsg `json:"history,omitempty"`
	CurrentSources   json.RawMessage `json:"current_sources"`
	CurrentSchedule  string          `json:"current_schedule"`
	CurrentWorkflows json.RawMessage `json:"current_workflows"`
	CurrentModel     string          `json:"current_model"`
	CurrentContext   json.RawMessage `json:"current_context,omitempty"`
}

type ConfigurePipelineResponse struct {
	Sources     json.RawMessage `json:"sources,omitempty"`
	Schedule    *string         `json:"schedule,omitempty"`
	Workflows   json.RawMessage `json:"workflows,omitempty"`
	Model       *string         `json:"model,omitempty"`
	Context     json.RawMessage `json:"context,omitempty"`
	Explanation string          `json:"explanation"`
}

func (s *Server) configurePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "id")

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

	llm, modelName := s.resolveLLM(req.Model)

	contextMsg := fmt.Sprintf(
		"Current pipeline settings:\nSources: %s\nSchedule: %q\nWorkflows: %s\nModel: %q\nEditorial brief: %s\n\nUser request: %s",
		string(req.CurrentSources),
		req.CurrentSchedule,
		string(req.CurrentWorkflows),
		req.CurrentModel,
		string(req.CurrentContext),
		req.Message,
	)

	contents := buildChatHistory(req.History)
	contents = append(contents, genai.NewContentFromText(contextMsg, genai.RoleUser))

	sysPrompt := ""
	if s.skills != nil {
		sysPrompt = s.skills.GetPrompt("pipeline-configure")

		// Inject stage-type skills for the pipeline's stages (mirrors configureNode pattern)
		if s.pipelineSvc != nil {
			if p, err := s.pipelineSvc.Get(r.Context(), pipelineID); err == nil {
				seen := map[string]bool{}
				for _, st := range p.Stages {
					key := "stage-" + st.Type
					if !seen[key] {
						if skill := s.skills.Get(key); skill != "" {
							sysPrompt += "\n\n--- STAGE GUIDE: " + st.Type + " ---\n\n" + skill
						}
						seen[key] = true
					}
				}
			}
		}
	}

	sysPrompt = s.appendModelCatalog(sysPrompt, modelName)

	if s.repo != nil {
		if wfs, err := s.repo.List(r.Context()); err == nil && len(wfs) > 0 {
			sysPrompt += "\n\nAvailable workflows:\n"
			for _, wf := range wfs {
				desc := wf.Description
				if desc == "" {
					desc = wf.Name
				}
				sysPrompt += fmt.Sprintf("- %q (%s)\n", wf.Name, desc)
			}
		}
	}

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

	var configResp ConfigurePipelineResponse
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&configResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}

	writeJSON(w, configResp)
}
