package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
)

// PipelineBundle is what the LLM returns — a pipeline + all its workflow definitions.
type PipelineBundle struct {
	Pipeline  upal.Pipeline             `json:"pipeline"`
	Workflows []upal.WorkflowDefinition `json:"workflows"`
}

// StageSummary is a lightweight view of a pipeline stage for prompt injection.
type StageSummary struct {
	ID   string
	Name string
	Type string
	Desc string
}

// PipelineSummary is a lightweight view of a pipeline for prompt injection.
// The LLM reads name + description + stage summaries without seeing full stage configs.
type PipelineSummary struct {
	Name        string
	Description string
	Stages      []StageSummary
}

// BuildPipelineSummaries extracts lightweight summaries from full pipeline definitions.
func BuildPipelineSummaries(pipelines []*upal.Pipeline) []PipelineSummary {
	summaries := make([]PipelineSummary, 0, len(pipelines))
	for _, p := range pipelines {
		s := PipelineSummary{
			Name:        p.Name,
			Description: p.Description,
		}
		for _, stage := range p.Stages {
			s.Stages = append(s.Stages, StageSummary{
				ID:   stage.ID,
				Name: stage.Name,
				Type: stage.Type,
				Desc: stage.Description,
			})
		}
		summaries = append(summaries, s)
	}
	return summaries
}

// formatPipelineList formats pipeline summaries as a human-readable prompt section.
func formatPipelineList(summaries []PipelineSummary) string {
	var b strings.Builder
	for _, p := range summaries {
		if p.Description != "" {
			fmt.Fprintf(&b, "- %q: %s\n", p.Name, p.Description)
		} else {
			fmt.Fprintf(&b, "- %q\n", p.Name)
		}
		for _, st := range p.Stages {
			if st.Desc != "" {
				fmt.Fprintf(&b, "  · %s [%s]: %s — %s\n", st.ID, st.Type, st.Name, st.Desc)
			} else {
				fmt.Fprintf(&b, "  · %s [%s]: %s\n", st.ID, st.Type, st.Name)
			}
		}
	}
	return b.String()
}

// NodeSummary is a lightweight view of a workflow node for prompt injection.
type NodeSummary struct {
	ID    string
	Type  upal.NodeType
	Label string // from config["label"]
	Desc  string // from config["description"]
}

// WorkflowSummary is a lightweight view of a workflow for prompt injection.
// The LLM reads name + description + node summaries without seeing full node configs.
type WorkflowSummary struct {
	Name        string
	Description string
	Nodes       []NodeSummary
}

// BuildWorkflowSummaries extracts lightweight summaries from full workflow definitions.
// Only name, description, and per-node label/description are included — never full configs.
func BuildWorkflowSummaries(wfs []*upal.WorkflowDefinition) []WorkflowSummary {
	summaries := make([]WorkflowSummary, 0, len(wfs))
	for _, wf := range wfs {
		s := WorkflowSummary{
			Name:        wf.Name,
			Description: wf.Description,
		}
		for _, n := range wf.Nodes {
			ns := NodeSummary{ID: n.ID, Type: n.Type}
			if label, ok := n.Config["label"].(string); ok {
				ns.Label = label
			}
			if desc, ok := n.Config["description"].(string); ok {
				ns.Desc = desc
			}
			s.Nodes = append(s.Nodes, ns)
		}
		summaries = append(summaries, s)
	}
	return summaries
}

// formatWorkflowList formats workflow summaries as a human-readable prompt section.
func formatWorkflowList(summaries []WorkflowSummary) string {
	var b strings.Builder
	for _, wf := range summaries {
		if wf.Description != "" {
			fmt.Fprintf(&b, "- %q: %s\n", wf.Name, wf.Description)
		} else {
			fmt.Fprintf(&b, "- %q\n", wf.Name)
		}
		for _, n := range wf.Nodes {
			label := n.Label
			if n.Desc != "" {
				if label != "" {
					label += " — " + n.Desc
				} else {
					label = n.Desc
				}
			}
			if label != "" {
				fmt.Fprintf(&b, "  · %s [%s]: %s\n", n.ID, n.Type, label)
			} else {
				fmt.Fprintf(&b, "  · %s [%s]\n", n.ID, n.Type)
			}
		}
	}
	return b.String()
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

// GeneratePipelineBundle generates a Pipeline in a single LLM call.
// availableWorkflows is the list of existing workflows the LLM may reference by name.
// When existingPipeline is non-nil, the generator operates in edit mode — it asks the LLM for
// a delta (only the changes) and applies it to the existing pipeline, guaranteeing that
// unchanged stages are preserved verbatim.
func (g *Generator) GeneratePipelineBundle(ctx context.Context, description string, existingPipeline *upal.Pipeline, availableWorkflows []WorkflowSummary, existingPipelines []PipelineSummary) (*PipelineBundle, error) {
	if existingPipeline != nil {
		return g.generatePipelineEdit(ctx, description, existingPipeline, availableWorkflows, existingPipelines)
	}
	return g.generatePipelineCreate(ctx, description, availableWorkflows, existingPipelines)
}

// generatePipelineCreate creates a brand-new pipeline bundle from a description.
func (g *Generator) generatePipelineCreate(ctx context.Context, description string, availableWorkflows []WorkflowSummary, existingPipelines []PipelineSummary) (*PipelineBundle, error) {
	sysPrompt := g.buildPipelineSysPrompt(g.skills.GetPrompt("pipeline-create"), availableWorkflows, existingPipelines)

	ctx = upalmodel.WithEffort(ctx, "high")

	text, err := g.generateWithSkills(ctx, sysPrompt, description, "generate pipeline bundle")
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

	// Keep only workflow stages that reference a known available workflow.
	// Workflow stages referencing unknown names are stripped (LLM hallucination guard).
	bundle.Pipeline.Stages = filterWorkflowStages(bundle.Pipeline.Stages, workflowNameSet(availableWorkflows))

	originalStageCount := len(bundle.Pipeline.Stages)
	bundle.Pipeline.Stages = stripInvalidStageTypes(bundle.Pipeline.Stages)
	if len(bundle.Pipeline.Stages) == 0 && originalStageCount > 0 {
		return nil, fmt.Errorf("generated pipeline has no valid stages (LLM used unsupported stage types or unknown workflow names)")
	}

	for i := range bundle.Pipeline.Stages {
		if bundle.Pipeline.Stages[i].ID == "" {
			bundle.Pipeline.Stages[i].ID = fmt.Sprintf("stage-%d", i+1)
		}
	}

	// Workflows are managed separately — never carry them over from LLM output.
	bundle.Workflows = nil
	return &bundle, nil
}

// generatePipelineEdit asks the LLM for a delta and applies it to the existing pipeline.
// Only stages explicitly mentioned in the delta are changed; all others are preserved verbatim.
func (g *Generator) generatePipelineEdit(ctx context.Context, description string, existing *upal.Pipeline, availableWorkflows []WorkflowSummary, existingPipelines []PipelineSummary) (*PipelineBundle, error) {
	sysPrompt := g.buildPipelineSysPrompt(g.skills.GetPrompt("pipeline-edit"), availableWorkflows, existingPipelines)

	pipelineJSON, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal existing pipeline: %w", err)
	}
	userContent := fmt.Sprintf("Current pipeline:\n%s\n\nInstruction: %s", string(pipelineJSON), description)

	ctx = upalmodel.WithEffort(ctx, "high")

	text, err := g.generateWithSkills(ctx, sysPrompt, userContent, "generate pipeline edit delta")
	if err != nil {
		return nil, err
	}

	var delta PipelineEditDelta
	if err := json.NewDecoder(strings.NewReader(text)).Decode(&delta); err != nil {
		return nil, fmt.Errorf("parse generated pipeline delta (model output may be malformed): %w\nraw output: %s", err, text)
	}

	// Validate stage changes: strip invalid types and workflow stages with unknown names.
	validWF := workflowNameSet(availableWorkflows)
	for i := range delta.StageChanges {
		s := delta.StageChanges[i].Stage
		if s == nil {
			continue
		}
		if !validStageTypes[s.Type] {
			delta.StageChanges[i].Stage = nil
		} else if s.Type == "workflow" && !validWF[s.Config.WorkflowName] {
			delta.StageChanges[i].Stage = nil
		}
	}

	// Ignore any workflows the LLM may have generated — workflow management is separate.
	delta.Workflows = nil

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

	return &PipelineBundle{Pipeline: *merged}, nil
}

// buildPipelineSysPrompt constructs the system prompt from a base prompt,
// injecting available workflows, models, tools, and skill guides.
func (g *Generator) buildPipelineSysPrompt(base string, availableWorkflows []WorkflowSummary, existingPipelines []PipelineSummary) string {
	sysPrompt := base

	if len(availableWorkflows) > 0 {
		sysPrompt += "\n\nAvailable workflows (use \"workflow\" stage type with workflow_name to reference — ONLY use names from this list):\n"
		sysPrompt += formatWorkflowList(availableWorkflows)
	}
	if len(existingPipelines) > 0 {
		sysPrompt += "\n\nExisting pipelines (for reference — avoid duplicating; understand patterns from stage summaries):\n"
		sysPrompt += formatPipelineList(existingPipelines)
	}

	// Final reinforcement — must be last so it benefits from recency bias.
	sysPrompt += "\n\nIMPORTANT: Your entire response must be ONLY the raw JSON object. No markdown fences, no explanation, no commentary before or after the JSON."
	return sysPrompt
}

// validStageTypes is the set of stage types the pipeline editor supports.
var validStageTypes = map[string]bool{
	"workflow":     true,
	"approval":     true,
	"notification": true,
	"schedule":     true,
	"trigger":      true,
	"transform":    true,
	"collect":      true,
}

// workflowNameSet builds a set of workflow names from a summary list.
func workflowNameSet(workflows []WorkflowSummary) map[string]bool {
	m := make(map[string]bool, len(workflows))
	for _, wf := range workflows {
		m[wf.Name] = true
	}
	return m
}

// filterWorkflowStages keeps "workflow" stages only if their workflow_name is in
// valid. Stages of other types pass through unchanged.
// If valid is empty, all "workflow" stages are stripped.
func filterWorkflowStages(stages []upal.Stage, valid map[string]bool) []upal.Stage {
	filtered := make([]upal.Stage, 0, len(stages))
	for _, s := range stages {
		if s.Type == "workflow" && !valid[s.Config.WorkflowName] {
			continue // unknown workflow name — strip it
		}
		filtered = append(filtered, s)
	}
	return filtered
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

