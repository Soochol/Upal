package generate

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// PipelineBundle is what the LLM returns — a pipeline + all its workflow definitions.
type PipelineBundle struct {
	Pipeline  upal.Pipeline             `json:"pipeline"`
	Workflows []upal.WorkflowDefinition `json:"workflows"`
}

// PipelineStageDelta describes a single stage change operation.
type PipelineStageDelta struct {
	Op      string      `json:"op"`              // "add", "update", "remove"
	Stage   *upal.Stage `json:"stage,omitempty"` // for "add" and "update"
	StageID string      `json:"stage_id,omitempty"` // for "remove"
}

// PipelineEditDelta is the LLM response format in edit mode.
// It describes only what should change; unchanged stages are preserved by applyDelta.
type PipelineEditDelta struct {
	Name         string                    `json:"name,omitempty"`
	Description  string                    `json:"description,omitempty"`
	StageChanges []PipelineStageDelta      `json:"stage_changes"`
	StageOrder   []string                  `json:"stage_order,omitempty"` // full ordered list of stage IDs after all changes
	Workflows    []upal.WorkflowDefinition `json:"workflows"`
}

// applyDelta merges a PipelineEditDelta into an existing Pipeline.
// Stages not referenced in delta.StageChanges are kept verbatim.
func applyDelta(existing *upal.Pipeline, delta *PipelineEditDelta) *upal.Pipeline {
	result := *existing
	if delta.Name != "" {
		result.Name = delta.Name
	}
	if delta.Description != "" {
		result.Description = delta.Description
	}

	updates := make(map[string]upal.Stage)
	removals := make(map[string]bool)
	var additions []upal.Stage

	for _, change := range delta.StageChanges {
		switch change.Op {
		case "update":
			if change.Stage != nil {
				updates[change.Stage.ID] = *change.Stage
			}
		case "remove":
			if change.StageID != "" {
				removals[change.StageID] = true
			}
		case "add":
			if change.Stage != nil {
				additions = append(additions, *change.Stage)
			}
		}
	}

	stages := make([]upal.Stage, 0, len(existing.Stages)+len(additions))
	for _, s := range existing.Stages {
		if removals[s.ID] {
			continue
		}
		if updated, ok := updates[s.ID]; ok {
			stages = append(stages, updated)
		} else {
			stages = append(stages, s)
		}
	}
	stages = append(stages, additions...)

	// If stage_order is provided, reorder stages accordingly.
	if len(delta.StageOrder) > 0 {
		byID := make(map[string]upal.Stage, len(stages))
		for _, s := range stages {
			byID[s.ID] = s
		}
		reordered := make([]upal.Stage, 0, len(stages))
		used := make(map[string]bool)
		for _, id := range delta.StageOrder {
			if s, ok := byID[id]; ok {
				reordered = append(reordered, s)
				used[id] = true
			}
		}
		// Safety net: append any stages not listed in stage_order.
		for _, s := range stages {
			if !used[s.ID] {
				reordered = append(reordered, s)
			}
		}
		stages = reordered
	}

	result.Stages = stages
	return &result
}

//go:embed prompts/pipeline-edit.md
var editPipelinePrompt string

//go:embed prompts/pipeline-create.md
var createPipelinePrompt string

// GeneratePipelineBundle generates a Pipeline and its WorkflowDefinitions in a single LLM call.
// When existingPipeline is non-nil, the generator operates in edit mode — it asks the LLM for
// a delta (only the changes) and applies it to the existing pipeline, guaranteeing that
// unchanged stages are preserved verbatim.
func (g *Generator) GeneratePipelineBundle(ctx context.Context, description string, existingPipeline *upal.Pipeline) (*PipelineBundle, error) {
	if existingPipeline != nil {
		return g.generatePipelineEdit(ctx, description, existingPipeline)
	}
	return g.generatePipelineCreate(ctx, description)
}

// generatePipelineCreate creates a brand-new pipeline bundle from a description.
func (g *Generator) generatePipelineCreate(ctx context.Context, description string) (*PipelineBundle, error) {
	sysPrompt := g.buildPipelineSysPrompt(createPipelinePrompt)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(description, genai.RoleUser),
		},
	}

	ctx = upalmodel.WithEffort(ctx, "high")

	text, err := g.callLLM(ctx, req, "generate pipeline bundle")
	if err != nil {
		return nil, err
	}

	var bundle PipelineBundle
	if err := json.NewDecoder(strings.NewReader(text)).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("parse generated pipeline bundle (model output may be malformed): %w\nraw output: %s", err, text)
	}

	if bundle.Pipeline.Name == "" {
		bundle.Pipeline.Name = "generated-pipeline"
	}

	bundle.Pipeline.Stages = stripInvalidStageTypes(bundle.Pipeline.Stages)

	for i := range bundle.Pipeline.Stages {
		if bundle.Pipeline.Stages[i].ID == "" {
			bundle.Pipeline.Stages[i].ID = fmt.Sprintf("stage-%d", i+1)
		}
	}

	bundle.Workflows = g.cleanWorkflows(bundle.Workflows, bundle.Pipeline.Stages)
	return &bundle, nil
}

// generatePipelineEdit asks the LLM for a delta and applies it to the existing pipeline.
// Only stages explicitly mentioned in the delta are changed; all others are preserved verbatim.
func (g *Generator) generatePipelineEdit(ctx context.Context, description string, existing *upal.Pipeline) (*PipelineBundle, error) {
	sysPrompt := g.buildPipelineSysPrompt(editPipelinePrompt)

	pipelineJSON, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal existing pipeline: %w", err)
	}
	userContent := fmt.Sprintf("Current pipeline:\n%s\n\nInstruction: %s", string(pipelineJSON), description)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userContent, genai.RoleUser),
		},
	}

	ctx = upalmodel.WithEffort(ctx, "high")

	text, err := g.callLLM(ctx, req, "generate pipeline edit delta")
	if err != nil {
		return nil, err
	}

	var delta PipelineEditDelta
	if err := json.NewDecoder(strings.NewReader(text)).Decode(&delta); err != nil {
		return nil, fmt.Errorf("parse generated pipeline delta (model output may be malformed): %w\nraw output: %s", err, text)
	}

	// Strip hallucinated stage types from the delta before applying.
	for i := range delta.StageChanges {
		if delta.StageChanges[i].Stage != nil && !validStageTypes[delta.StageChanges[i].Stage.Type] {
			delta.StageChanges[i].Stage = nil // nullify so applyDelta skips it
		}
	}

	// Apply delta: unchanged stages are preserved, only the LLM's explicit changes go through.
	merged := applyDelta(existing, &delta)

	// Backfill missing stage IDs on any newly added stages.
	usedIDs := make(map[string]bool)
	for _, s := range merged.Stages {
		usedIDs[s.ID] = true
	}
	counter := len(merged.Stages)
	for i := range merged.Stages {
		if merged.Stages[i].ID == "" {
			for {
				counter++
				id := fmt.Sprintf("stage-%d", counter)
				if !usedIDs[id] {
					merged.Stages[i].ID = id
					usedIDs[id] = true
					break
				}
			}
		}
	}

	workflows := g.cleanWorkflows(delta.Workflows, merged.Stages)
	return &PipelineBundle{Pipeline: *merged, Workflows: workflows}, nil
}

// buildPipelineSysPrompt constructs the system prompt from a base prompt,
// injecting available models, tools, and skill guides.
func (g *Generator) buildPipelineSysPrompt(base string) string {
	sysPrompt := base

	if len(g.models) > 0 {
		sysPrompt += g.buildModelPrompt()
	}
	if len(g.toolNames) > 0 {
		sysPrompt += "\n\nAvailable tools for agent nodes (use in config \"tools\" array):\n"
		for _, name := range g.toolNames {
			sysPrompt += fmt.Sprintf("- %q\n", name)
		}
		sysPrompt += "IMPORTANT: ONLY use tools from this list. Do NOT invent or hallucinate tool names that are not listed here. If the task requires a capability not covered by these tools, use an agent node with a detailed prompt instead."
	}
	if g.skills != nil {
		if agentSkill := g.skills.Get("agent-node"); agentSkill != "" {
			sysPrompt += "\n\n--- AGENT NODE GUIDE ---\n\n" + agentSkill
		}
		if inputSkill := g.skills.Get("input-node"); inputSkill != "" {
			sysPrompt += "\n\n--- INPUT NODE GUIDE ---\n\n" + inputSkill
		}
		if outputSkill := g.skills.Get("output-node"); outputSkill != "" {
			sysPrompt += "\n\n--- OUTPUT NODE GUIDE ---\n\n" + outputSkill
		}
	}
	sysPrompt += "\n\nOutput ONLY raw JSON"
	return sysPrompt
}

// callLLM sends a request and returns the stripped JSON text from the response.
func (g *Generator) callLLM(ctx context.Context, req *adkmodel.LLMRequest, opName string) (string, error) {
	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("%s: %w", opName, err)
		}
		resp = r
	}
	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	text := llmutil.ExtractText(resp)
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		return "", fmt.Errorf("parse %s (model output may be malformed): %w\nraw output: %s", opName, err, text)
	}
	return content, nil
}

// validStageTypes is the set of stage types the pipeline editor supports.
var validStageTypes = map[string]bool{
	"workflow":  true,
	"approval":  true,
	"schedule":  true,
	"trigger":   true,
	"transform": true,
}

// stripInvalidStageTypes removes stages whose type is not in validStageTypes.
func stripInvalidStageTypes(stages []upal.Stage) []upal.Stage {
	filtered := make([]upal.Stage, 0, len(stages))
	for _, s := range stages {
		if validStageTypes[s.Type] {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// cleanWorkflows post-processes workflows: strips invalid node types/tools/models,
// and keeps only those referenced by the given stages.
func (g *Generator) cleanWorkflows(workflows []upal.WorkflowDefinition, stages []upal.Stage) []upal.WorkflowDefinition {
	referenced := make(map[string]bool)
	for _, stage := range stages {
		if stage.Type == "workflow" && stage.Config.WorkflowName != "" {
			referenced[stage.Config.WorkflowName] = true
		}
	}

	cleaned := make([]upal.WorkflowDefinition, 0, len(workflows))
	for i := range workflows {
		wf := &workflows[i]
		stripInvalidNodeTypes(wf)
		g.stripInvalidTools(wf)
		g.fixInvalidModels(wf)
		if referenced[wf.Name] {
			cleaned = append(cleaned, *wf)
		}
	}
	return cleaned
}
