package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
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
	Sources          json.RawMessage      `json:"sources,omitempty"`
	Schedule         *string              `json:"schedule,omitempty"`
	Workflows        json.RawMessage      `json:"workflows,omitempty"`
	Model            *string              `json:"model,omitempty"`
	Context          json.RawMessage      `json:"context,omitempty"`
	Explanation      string               `json:"explanation"`
	CreatedWorkflows []CreatedWorkflowInfo `json:"created_workflows,omitempty"`
}

// configureLLMResponse is the internal struct for parsing LLM output (includes create_workflows).
type configureLLMResponse struct {
	Sources         json.RawMessage     `json:"sources,omitempty"`
	Schedule        *string             `json:"schedule,omitempty"`
	Workflows       json.RawMessage     `json:"workflows,omitempty"`
	Model           *string             `json:"model,omitempty"`
	Context         json.RawMessage     `json:"context,omitempty"`
	Explanation     string              `json:"explanation"`
	CreateWorkflows []CreateWorkflowSpec `json:"create_workflows,omitempty"`
}

type CreateWorkflowSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreatedWorkflowInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "success" or "failed"
	Error  string `json:"error,omitempty"`
}

func (s *Server) configurePipeline(w http.ResponseWriter, r *http.Request) {
	var req ConfigurePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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

	llm := s.generator.LLM()
	modelName := s.generator.Model()
	if req.Model != "" && s.llmResolver != nil {
		if resolved, resolvedName, err := s.llmResolver.Resolve(req.Model); err == nil {
			llm = resolved
			modelName = resolvedName
		}
	}

	contextMsg := fmt.Sprintf(
		"Current pipeline settings:\nSources: %s\nSchedule: %q\nWorkflows: %s\nModel: %q\nEditorial brief: %s\n\nUser request: %s",
		string(req.CurrentSources),
		req.CurrentSchedule,
		string(req.CurrentWorkflows),
		req.CurrentModel,
		string(req.CurrentContext),
		req.Message,
	)

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

	sysPrompt := ""
	if s.skills != nil {
		sysPrompt = s.skills.GetPrompt("pipeline-configure")
	}

	// Inject available models (same pattern as configureNode in configure.go)
	if allModels := upalmodel.KnownModelsGrouped(s.providerConfigs); len(allModels) > 0 {
		sysPrompt += fmt.Sprintf("\n\nAvailable models (use in \"model\" field):\nDefault model: %q\n", modelName)
		var textModels, imageModels []upal.ModelInfo
		for _, m := range allModels {
			switch m.Category {
			case upal.ModelCategoryText:
				textModels = append(textModels, m)
			case upal.ModelCategoryImage:
				imageModels = append(imageModels, m)
			}
		}
		if len(textModels) > 0 {
			sysPrompt += "\nText/reasoning models:\n"
			for _, m := range textModels {
				sysPrompt += fmt.Sprintf("- %q [%s] — %s\n", m.ID, m.Tier, m.Hint)
			}
		}
		if len(imageModels) > 0 {
			sysPrompt += "\nImage generation models:\n"
			for _, m := range imageModels {
				sysPrompt += fmt.Sprintf("- %q — %s\n", m.ID, m.Hint)
			}
		}
	}

	// Inject available workflows so LLM only references real ones
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

	var llmResp configureLLMResponse
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&llmResp); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse LLM response: %v\nraw: %s", err, content), http.StatusInternalServerError)
		return
	}

	// Generate requested workflows (limit to 3)
	const maxCreateWorkflows = 3
	specs := llmResp.CreateWorkflows
	if len(specs) > maxCreateWorkflows {
		specs = specs[:maxCreateWorkflows]
	}

	var created []CreatedWorkflowInfo
	failedNames := map[string]bool{}
	for _, spec := range specs {
		if spec.Name == "" || spec.Description == "" {
			continue
		}
		// Skip if workflow already exists
		if s.repo != nil {
			if _, err := s.repo.Get(r.Context(), spec.Name); err == nil {
				created = append(created, CreatedWorkflowInfo{Name: spec.Name, Status: "exists"})
				continue
			}
		}
		wf, err := s.generator.GenerateWorkflow(r.Context(), spec.Description)
		if err != nil {
			failedNames[spec.Name] = true
			created = append(created, CreatedWorkflowInfo{Name: spec.Name, Status: "failed", Error: err.Error()})
			continue
		}
		wf.Name = spec.Name
		if s.repo != nil {
			if err := s.repo.Create(r.Context(), wf); err != nil {
				failedNames[spec.Name] = true
				created = append(created, CreatedWorkflowInfo{Name: spec.Name, Status: "failed", Error: err.Error()})
				continue
			}
		}
		created = append(created, CreatedWorkflowInfo{Name: spec.Name, Status: "success"})
	}

	// Filter out failed workflow references so the session doesn't reference non-existent workflows
	workflows := llmResp.Workflows
	if len(failedNames) > 0 {
		workflows = filterFailedWorkflows(llmResp.Workflows, failedNames)
	}

	configResp := ConfigurePipelineResponse{
		Sources:          llmResp.Sources,
		Schedule:         llmResp.Schedule,
		Workflows:        workflows,
		Model:            llmResp.Model,
		Context:          llmResp.Context,
		Explanation:      llmResp.Explanation,
		CreatedWorkflows: created,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configResp)
}

// filterFailedWorkflows removes workflow references whose names are in the failed set.
func filterFailedWorkflows(raw json.RawMessage, failed map[string]bool) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var wfs []struct {
		WorkflowName string `json:"workflow_name"`
	}
	if err := json.Unmarshal(raw, &wfs); err != nil {
		return raw
	}

	var kept []json.RawMessage
	var all []json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return raw
	}
	for i, wf := range wfs {
		if !failed[wf.WorkflowName] {
			kept = append(kept, all[i])
		}
	}
	if len(kept) == len(all) {
		return raw
	}
	out, err := json.Marshal(kept)
	if err != nil {
		return raw
	}
	return out
}
