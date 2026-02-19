package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Generator converts natural language descriptions into WorkflowDefinitions.
type Generator struct {
	llm   adkmodel.LLM
	model string
}

// New creates a Generator that uses the given LLM and model name.
func New(llm adkmodel.LLM, model string) *Generator {
	return &Generator{llm: llm, model: model}
}

// LLM returns the underlying LLM used by the generator.
func (g *Generator) LLM() adkmodel.LLM {
	return g.llm
}

// Model returns the model name used by the generator.
func (g *Generator) Model() string {
	return g.model
}

const createSystemPrompt = `You are a workflow generator for the Upal platform. Given a user's natural language description, you must produce a valid workflow JSON.

A workflow has:
- "name": a slug-style name (lowercase, hyphens)
- "version": always 1
- "nodes": array of node objects
- "edges": array of edge objects connecting nodes

Node types:
1. "input"  — collects user input. Config: {} (no special config needed)
2. "agent"  — calls an AI model. Config: {"model": "provider/model-name", "system_prompt": "...", "prompt": "Use {{node_id}} to reference input from previous nodes"}
3. "tool"   — calls a registered tool. Config: {"tool": "tool_name", "input": "{{node_id}}"}
4. "output" — produces final output. Config: {} (aggregates all upstream state)

Edge format: {"from": "node_id", "to": "node_id"}

Rules:
- Every workflow must start with at least one "input" node and end with one "output" node.
- Agent prompts should use {{node_id}} template syntax to reference upstream node outputs.
- Node IDs should be descriptive slugs like "user_question", "summarizer", "final_output".
- For the "model" field, use "openai/gpt-4o" as the default unless the user specifies otherwise.
- Keep workflows minimal — only add nodes that are necessary for the described task.

Respond with ONLY valid JSON, no markdown fences, no explanation.`

const editSystemPrompt = `You are a workflow editor for the Upal platform. You will be given an existing workflow JSON and a user's instruction to modify it. You must return the COMPLETE updated workflow JSON.

A workflow has:
- "name": a slug-style name (lowercase, hyphens)
- "version": always 1
- "nodes": array of node objects
- "edges": array of edge objects connecting nodes

Node types:
1. "input"  — collects user input. Config: {} (no special config needed)
2. "agent"  — calls an AI model. Config: {"model": "provider/model-name", "system_prompt": "...", "prompt": "Use {{node_id}} to reference input from previous nodes"}
3. "tool"   — calls a registered tool. Config: {"tool": "tool_name", "input": "{{node_id}}"}
4. "output" — produces final output. Config: {} (aggregates all upstream state)

Edge format: {"from": "node_id", "to": "node_id"}

Rules:
- Preserve existing nodes and edges unless the user explicitly asks to remove or replace them.
- When adding new nodes, connect them logically to the existing graph.
- Agent prompts should use {{node_id}} template syntax to reference upstream node outputs.
- Node IDs should be descriptive slugs like "user_question", "summarizer", "final_output".
- Avoid duplicate node IDs — check against existing IDs before creating new ones.
- For the "model" field, use "openai/gpt-4o" as the default unless the user specifies otherwise.
- Every workflow must have at least one "input" node and one "output" node.

Respond with the COMPLETE updated workflow as valid JSON. No markdown fences, no explanation.`

// Generate creates a WorkflowDefinition from a natural language description.
// If existingWorkflow is non-nil, the generator operates in edit mode — modifying the
// existing workflow according to the description instead of creating from scratch.
func (g *Generator) Generate(ctx context.Context, description string, existingWorkflow *upal.WorkflowDefinition) (*upal.WorkflowDefinition, error) {
	sysPrompt := createSystemPrompt
	userContent := description

	if existingWorkflow != nil {
		sysPrompt = editSystemPrompt
		wfJSON, err := json.MarshalIndent(existingWorkflow, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal existing workflow: %w", err)
		}
		userContent = fmt.Sprintf("Current workflow:\n%s\n\nInstruction: %s", string(wfJSON), description)
	}

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userContent, genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return nil, fmt.Errorf("generate workflow: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract text from response parts.
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

	var wf upal.WorkflowDefinition
	if err := json.Unmarshal([]byte(content), &wf); err != nil {
		return nil, fmt.Errorf("parse generated workflow (model output may be malformed): %w\nraw output: %s", err, content)
	}

	if err := validate(&wf); err != nil {
		return nil, fmt.Errorf("invalid generated workflow: %w", err)
	}

	return &wf, nil
}

// validate checks that the generated workflow has the minimum required structure.
func validate(wf *upal.WorkflowDefinition) error {
	if wf.Name == "" {
		return fmt.Errorf("missing workflow name")
	}
	if len(wf.Nodes) == 0 {
		return fmt.Errorf("workflow has no nodes")
	}

	nodeIDs := map[string]bool{}
	hasInput := false
	hasOutput := false

	for _, n := range wf.Nodes {
		if n.ID == "" {
			return fmt.Errorf("node missing ID")
		}
		if nodeIDs[n.ID] {
			return fmt.Errorf("duplicate node ID: %q", n.ID)
		}
		nodeIDs[n.ID] = true

		switch n.Type {
		case upal.NodeTypeInput:
			hasInput = true
		case upal.NodeTypeOutput:
			hasOutput = true
		case upal.NodeTypeAgent, upal.NodeTypeTool:
			// valid
		default:
			return fmt.Errorf("unknown node type %q for node %q", n.Type, n.ID)
		}
	}

	if !hasInput {
		return fmt.Errorf("workflow must have at least one input node")
	}
	if !hasOutput {
		return fmt.Errorf("workflow must have at least one output node")
	}

	for _, e := range wf.Edges {
		if !nodeIDs[e.From] {
			return fmt.Errorf("edge references unknown source node %q", e.From)
		}
		if !nodeIDs[e.To] {
			return fmt.Errorf("edge references unknown target node %q", e.To)
		}
	}

	return nil
}
