package generate

import (
	"context"
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

var createPipelinePrompt = `You are a pipeline generator for the Upal platform. Given a user's natural language description, you must produce a valid pipeline bundle JSON containing a pipeline and all its workflow definitions.

A pipeline bundle has two top-level keys:
1. "pipeline" — the pipeline metadata and stages
2. "workflows" — array of WorkflowDefinition objects, one per workflow stage

PIPELINE STRUCTURE:
{
  "name": "slug-style-name",
  "description": "파이프라인 설명",
  "stages": [
    {
      "id": "stage-1",
      "name": "Stage Name",
      "type": "workflow",
      "config": {
        "workflow_name": "my-workflow-name"
      }
    },
    {
      "id": "stage-2",
      "name": "Approval Stage",
      "type": "approval",
      "config": {
        "message": "승인을 기다리고 있습니다.",
        "timeout": 3600
      },
      "depends_on": ["stage-1"]
    }
  ]
}

Valid stage types:
- "workflow"  — runs a workflow (config must include "workflow_name" matching a workflow in "workflows" array)
- "approval"  — pauses for human approval (config: message, connection_id, timeout)
- "schedule"  — triggers on a cron schedule (config: cron, timezone)
- "transform" — transforms data between stages (config: expression)

WORKFLOW STRUCTURE (same rules as standalone workflows):
A workflow has:
- "name": a slug-style name (lowercase, hyphens) — must match the "workflow_name" in the corresponding pipeline stage
- "version": always 1
- "nodes": array of node objects with {id, type, config}
- "edges": array of edge objects connecting nodes

Node types (ONLY these three types are valid):
1. "input"  — collects user input
2. "agent"  — calls an AI model
3. "output" — produces final output

EXAMPLE — a two-stage pipeline: research then approve:
{
  "pipeline": {
    "name": "research-and-approve",
    "description": "리서치를 수행하고 결과를 승인합니다.",
    "stages": [
      {
        "id": "stage-1",
        "name": "Research Workflow",
        "type": "workflow",
        "config": {
          "workflow_name": "research-workflow"
        }
      },
      {
        "id": "stage-2",
        "name": "Approval",
        "type": "approval",
        "config": {
          "message": "리서치 결과를 검토하고 승인해 주세요.",
          "timeout": 86400
        },
        "depends_on": ["stage-1"]
      }
    ]
  },
  "workflows": [
    {
      "name": "research-workflow",
      "version": 1,
      "nodes": [
        {"id": "topic", "type": "input", "config": {"label": "리서치 주제", "placeholder": "조사할 주제를 입력하세요...", "description": "리서치할 주제"}},
        {"id": "researcher", "type": "agent", "config": {"model": "anthropic/claude-sonnet-4-6", "label": "리서처", "system_prompt": "당신은 전문 리서처입니다.", "prompt": "다음 주제를 조사하세요:\n\n{{topic}}", "description": "주제를 리서치"}},
        {"id": "final_output", "type": "output", "config": {"label": "리서치 결과", "prompt": "{{researcher}}", "description": "리서치 결과를 표시"}}
      ],
      "edges": [{"from": "topic", "to": "researcher"}, {"from": "researcher", "to": "final_output"}]
    }
  ]
}

Rules:
- Every "workflow" stage MUST have a corresponding entry in "workflows[]" with a "name" matching the stage's "workflow_name".
- Stage IDs must be "stage-1", "stage-2", etc. (sequential).
- The pipeline "name" and stage "workflow_name" values must be English slugs (lowercase, hyphens).
- ALL user-facing text (pipeline description, stage names, node labels, descriptions, prompts, system_prompt, placeholder) MUST be written in Korean (한국어).
- Node IDs and workflow names remain English slugs.
- Every workflow must have at least one "input" node and one "output" node.
- Agent prompts should use {{node_id}} template syntax to reference upstream node outputs.
- Keep workflows minimal — only add nodes necessary for the described task.`

// GeneratePipelineBundle generates a Pipeline and its WorkflowDefinitions in a single LLM call.
func (g *Generator) GeneratePipelineBundle(ctx context.Context, description string) (*PipelineBundle, error) {
	sysPrompt := createPipelinePrompt

	// Inject available models grouped by category so the LLM matches purpose to model.
	if len(g.models) > 0 {
		sysPrompt += g.buildModelPrompt()
	}

	// Inject available tools so the LLM only references real tools.
	if len(g.toolNames) > 0 {
		sysPrompt += "\n\nAvailable tools for agent nodes (use in config \"tools\" array):\n"
		for _, name := range g.toolNames {
			sysPrompt += fmt.Sprintf("- %q\n", name)
		}
		sysPrompt += "IMPORTANT: ONLY use tools from this list. Do NOT invent or hallucinate tool names that are not listed here. If the task requires a capability not covered by these tools, use an agent node with a detailed prompt instead."
	}

	// Inject node-type skills so the LLM has detailed config guidance.
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

	// Final reinforcement — must be the LAST thing in the system prompt.
	sysPrompt += "\n\nOutput ONLY raw JSON"

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(description, genai.RoleUser),
		},
	}

	// Pipeline generation requires careful instruction following, so request high effort.
	ctx = upalmodel.WithEffort(ctx, "high")

	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return nil, fmt.Errorf("generate pipeline bundle: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract text from response parts.
	text := llmutil.ExtractText(resp)

	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parse generated pipeline bundle (model output may be malformed): %w\nraw output: %s", err, text)
	}

	// Use json.Decoder to parse only the first JSON value and ignore trailing text.
	var bundle PipelineBundle
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("parse generated pipeline bundle (model output may be malformed): %w\nraw output: %s", err, content)
	}

	// Backfill pipeline name when the LLM omits it.
	if bundle.Pipeline.Name == "" {
		bundle.Pipeline.Name = "generated-pipeline"
	}

	// Backfill missing stage IDs.
	for i := range bundle.Pipeline.Stages {
		if bundle.Pipeline.Stages[i].ID == "" {
			bundle.Pipeline.Stages[i].ID = fmt.Sprintf("stage-%d", i+1)
		}
	}

	// Build a set of workflow_name values referenced by pipeline stages.
	referencedWorkflows := make(map[string]bool)
	for _, stage := range bundle.Pipeline.Stages {
		if stage.Type == "workflow" && stage.Config.WorkflowName != "" {
			referencedWorkflows[stage.Config.WorkflowName] = true
		}
	}

	// Post-process each workflow: strip invalid node types, tools, and models.
	cleaned := make([]upal.WorkflowDefinition, 0, len(bundle.Workflows))
	for i := range bundle.Workflows {
		wf := &bundle.Workflows[i]
		stripInvalidNodeTypes(wf)
		g.stripInvalidTools(wf)
		g.fixInvalidModels(wf)

		// Only keep workflows that are referenced by a pipeline stage.
		if referencedWorkflows[wf.Name] {
			cleaned = append(cleaned, *wf)
		}
	}
	bundle.Workflows = cleaned

	return &bundle, nil
}
