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

// ModelOption describes an available model for prompt injection.
// Constructed from api.ModelInfo at the call site to avoid import cycles.
type ModelOption struct {
	ID       string // "provider/model-name"
	Category string // "text" or "image"
	Tier     string // "high", "mid", "low"
	Hint     string // one-line capability description
}

// skillProvider abstracts read access to skill content.
type skillProvider interface {
	Get(name string) string
}

// Generator converts natural language descriptions into WorkflowDefinitions.
type Generator struct {
	llm       adkmodel.LLM
	model     string
	skills    skillProvider
	toolNames []string       // available tool names from the tool registry
	models    []ModelOption  // available models with category/tier metadata
}

// New creates a Generator that uses the given LLM and model name.
// toolNames lists the names of tools registered in the tool registry;
// these are injected into the generation prompt so the LLM only references real tools.
// models lists the available models with category/tier/hint metadata;
// these are injected so the LLM selects the right model for each node's purpose.
func New(llm adkmodel.LLM, model string, skills skillProvider, toolNames []string, models []ModelOption) *Generator {
	return &Generator{llm: llm, model: model, skills: skills, toolNames: toolNames, models: models}
}

// LLM returns the underlying LLM used by the generator.
func (g *Generator) LLM() adkmodel.LLM {
	return g.llm
}

// Model returns the model name used by the generator.
func (g *Generator) Model() string {
	return g.model
}

//go:embed prompts/workflow-create.md
var createBasePrompt string

//go:embed prompts/workflow-edit.md
var editBasePrompt string

// Generate creates a WorkflowDefinition from a natural language description.
// If existingWorkflow is non-nil, the generator operates in edit mode — modifying the
// existing workflow according to the description instead of creating from scratch.
func (g *Generator) Generate(ctx context.Context, description string, existingWorkflow *upal.WorkflowDefinition) (*upal.WorkflowDefinition, error) {
	sysPrompt := createBasePrompt
	userContent := description

	if existingWorkflow != nil {
		sysPrompt = editBasePrompt
		wfJSON, err := json.MarshalIndent(existingWorkflow, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal existing workflow: %w", err)
		}
		userContent = fmt.Sprintf("Current workflow:\n%s\n\nInstruction: %s", string(wfJSON), description)
	}

	// Inject available models grouped by category so the LLM matches purpose to model.
	if len(g.models) > 0 {
		sysPrompt += g.buildModelPrompt()
	}

	// Inject available tools so the LLM only references real tools.
	allTools := g.toolNames
	if len(allTools) > 0 {
		sysPrompt += "\n\nAvailable tools for agent nodes (use in config \"tools\" array):\n"
		for _, name := range allTools {
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

	// Final reinforcement — must be the LAST thing in the system prompt so it
	// benefits from recency bias and isn't buried under reference material.
	sysPrompt += "\n\nIMPORTANT: Your entire response must be ONLY the raw JSON object. No markdown fences, no explanation, no commentary before or after the JSON."

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userContent, genai.RoleUser),
		},
	}

	// Workflow generation requires careful instruction following (long system
	// prompt with detailed skill guides), so request high effort from the LLM.
	ctx = upalmodel.WithEffort(ctx, "high")

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
	text := llmutil.ExtractText(resp)

	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parse generated workflow (model output may be malformed): %w\nraw output: %s", err, text)
	}

	// Use json.Decoder to parse only the first JSON value and ignore trailing text.
	var wf upal.WorkflowDefinition
	if err := json.NewDecoder(strings.NewReader(content)).Decode(&wf); err != nil {
		return nil, fmt.Errorf("parse generated workflow (model output may be malformed): %w\nraw output: %s", err, content)
	}

	// Backfill workflow name when the LLM omits it.
	if wf.Name == "" && existingWorkflow != nil {
		wf.Name = existingWorkflow.Name
	}
	if wf.Name == "" {
		wf.Name = "generated-workflow"
	}

	// Strip hallucinated node types before validation.
	stripInvalidNodeTypes(&wf)

	if err := validate(&wf); err != nil {
		return nil, fmt.Errorf("invalid generated workflow: %w", err)
	}

	// Strip hallucinated tool names that don't exist in the registry.
	g.stripInvalidTools(&wf)

	// Replace invalid model IDs with the default model.
	g.fixInvalidModels(&wf)

	return &wf, nil
}

// stripInvalidTools removes tool names from agent node configs that don't exist
// in the tool registry. This prevents hallucinated tools from reaching execution.
func (g *Generator) stripInvalidTools(wf *upal.WorkflowDefinition) {
	valid := make(map[string]bool, len(g.toolNames))
	for _, name := range g.toolNames {
		valid[name] = true
	}

	for i, n := range wf.Nodes {
		toolNames, ok := n.Config["tools"].([]any)
		if !ok || len(toolNames) == 0 {
			continue
		}
		filtered := make([]any, 0, len(toolNames))
		for _, tn := range toolNames {
			name, ok := tn.(string)
			if ok && valid[name] {
				filtered = append(filtered, tn)
			}
		}
		if len(filtered) == 0 {
			delete(wf.Nodes[i].Config, "tools")
		} else {
			wf.Nodes[i].Config["tools"] = filtered
		}
	}
}

// buildModelPrompt creates a categorized model list for the system prompt.
func (g *Generator) buildModelPrompt() string {
	// Group models by category.
	groups := map[string][]ModelOption{}
	for _, m := range g.models {
		groups[m.Category] = append(groups[m.Category], m)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n\nAvailable models for agent nodes (use in config \"model\" field):\nDefault model: %q\n", g.model))

	if text := groups["text"]; len(text) > 0 {
		b.WriteString("\nText/reasoning models — use for analysis, generation, conversation, tool-use, and any task that processes or produces text:\n")
		for _, m := range text {
			b.WriteString(fmt.Sprintf("- %q [%s] — %s\n", m.ID, m.Tier, m.Hint))
		}
	}

	if image := groups["image"]; len(image) > 0 {
		b.WriteString("\nImage generation models — use ONLY when the task requires creating, editing, or generating images:\n")
		for _, m := range image {
			b.WriteString(fmt.Sprintf("- %q — %s\n", m.ID, m.Hint))
		}
	}

	b.WriteString(`
MODEL SELECTION RULES:
1. ONLY use models from the lists above.
2. Choose the model category that matches the node's PURPOSE: text models for reasoning/text tasks, image models for image generation tasks.
3. Within text models, match tier to complexity: "high" for complex reasoning, "mid" for general tasks, "low" for simple/fast tasks.
4. Use the default model when no specific model is needed.`)

	return b.String()
}

// fixInvalidModels replaces model IDs that don't exist in the available models
// with the generator's default model. This mirrors stripInvalidTools for models.
func (g *Generator) fixInvalidModels(wf *upal.WorkflowDefinition) {
	if len(g.models) == 0 {
		return
	}
	valid := make(map[string]bool, len(g.models))
	for _, m := range g.models {
		valid[m.ID] = true
	}

	// Also accept the generator's own default model (may use short name like "sonnet").
	// Build the full ID by checking if any valid model ends with the default name.
	defaultFull := ""
	for _, m := range g.models {
		parts := strings.SplitN(m.ID, "/", 2)
		if len(parts) == 2 && parts[1] == g.model {
			defaultFull = m.ID
			break
		}
	}
	if defaultFull == "" && len(g.models) > 0 {
		defaultFull = g.models[0].ID
	}

	for i, n := range wf.Nodes {
		if n.Type != upal.NodeTypeAgent {
			continue
		}
		model, _ := n.Config["model"].(string)
		if model != "" && !valid[model] {
			wf.Nodes[i].Config["model"] = defaultFull
		}
	}
}

// stripInvalidNodeTypes removes nodes whose type is not one of the valid
// generatable types (input, agent, output). Also removes edges referencing
// removed nodes.
func stripInvalidNodeTypes(wf *upal.WorkflowDefinition) {
	generatable := map[upal.NodeType]bool{
		upal.NodeTypeInput:  true,
		upal.NodeTypeAgent:  true,
		upal.NodeTypeOutput: true,
	}

	removed := map[string]bool{}
	filtered := make([]upal.NodeDefinition, 0, len(wf.Nodes))
	for _, n := range wf.Nodes {
		if generatable[n.Type] {
			filtered = append(filtered, n)
		} else {
			removed[n.ID] = true
		}
	}
	wf.Nodes = filtered

	if len(removed) > 0 {
		edges := make([]upal.EdgeDefinition, 0, len(wf.Edges))
		for _, e := range wf.Edges {
			if !removed[e.From] && !removed[e.To] {
				edges = append(edges, e)
			}
		}
		wf.Edges = edges
	}
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
		case upal.NodeTypeAgent:
			if n.Config == nil {
				return fmt.Errorf("agent node %q missing config", n.ID)
			}
			if _, ok := n.Config["model"].(string); !ok {
				return fmt.Errorf("agent node %q missing required field \"model\"", n.ID)
			}
			if _, ok := n.Config["prompt"].(string); !ok {
				return fmt.Errorf("agent node %q missing required field \"prompt\"", n.ID)
			}
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
