package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// DefaultLLMFunc resolves the current default LLM dynamically at request time.
// Called at the start of each generation to pick up runtime changes
// (e.g. when the user changes the default provider in Settings).
// Returns an error if no default LLM is configured.
type DefaultLLMFunc func(ctx context.Context) (adkmodel.LLM, string, error)

// Generator converts natural language descriptions into WorkflowDefinitions
// and handles LLM-based configuration of nodes and pipelines.
type Generator struct {
	llm            adkmodel.LLM
	model          string
	skills         skills.Provider
	toolInfos      []upal.ToolSummary  // available tools with names and descriptions
	models         []upal.ModelSummary // available models with category/tier metadata
	defaultModelID string              // provider-prefixed form of model (e.g. "anthropic/claude-sonnet-4-6")
	llmResolver    ports.LLMResolver   // resolves "provider/model" → LLM instance (optional)
	defaultLLMFunc DefaultLLMFunc      // dynamic default resolver (optional)
}

// New creates a Generator that uses the given LLM and model name.
// toolNames lists the names of tools registered in the tool registry;
// these are injected into the generation prompt so the LLM only references real tools.
// models lists the available models with category/tier/hint metadata;
// these are injected so the LLM selects the right model for each node's purpose.
func New(llm adkmodel.LLM, model string, skillsProv skills.Provider, toolInfos []upal.ToolSummary, models []upal.ModelSummary) *Generator {
	// Resolve the full provider-prefixed model ID once at construction time.
	defaultModelID := ""
	for _, m := range models {
		if _, after, ok := strings.Cut(m.ID, "/"); ok && after == model {
			defaultModelID = m.ID
			break
		}
	}
	if defaultModelID == "" && len(models) > 0 {
		defaultModelID = models[0].ID
	}
	return &Generator{llm: llm, model: model, skills: skillsProv, toolInfos: toolInfos, models: models, defaultModelID: defaultModelID}
}

// SetLLMResolver sets the resolver used for per-request model overrides.
func (g *Generator) SetLLMResolver(r ports.LLMResolver) {
	g.llmResolver = r
}

// SetDefaultLLMFunc sets a function that dynamically resolves the current default LLM.
// This allows the generator to pick up runtime changes when the user changes
// the default provider in Settings, without requiring a server restart.
func (g *Generator) SetDefaultLLMFunc(fn DefaultLLMFunc) {
	g.defaultLLMFunc = fn
}

// currentDefault returns the current default LLM and model name.
// Resolves dynamically from the configured DefaultLLMFunc (backed by DB settings).
// Returns an error if no default LLM provider is configured.
func (g *Generator) currentDefault(ctx context.Context) (adkmodel.LLM, string, error) {
	if g.defaultLLMFunc != nil {
		return g.defaultLLMFunc(ctx)
	}
	if g.llm != nil {
		return g.llm, g.model, nil
	}
	return nil, "", fmt.Errorf("no default LLM provider configured")
}

// resolveLLM returns the LLM and model name for a request.
// Priority: explicit request model > dynamic default.
func (g *Generator) resolveLLM(ctx context.Context, requestModel string) (adkmodel.LLM, string, error) {
	if requestModel != "" && g.llmResolver != nil {
		if resolved, resolvedName, err := g.llmResolver.Resolve(requestModel); err == nil {
			return resolved, resolvedName, nil
		}
	}
	return g.currentDefault(ctx)
}

// LLM returns the current default LLM, or nil if not configured.
func (g *Generator) LLM(ctx context.Context) adkmodel.LLM {
	llm, _, _ := g.currentDefault(ctx)
	return llm
}

// Model returns the current default model name, or empty if not configured.
func (g *Generator) Model(ctx context.Context) string {
	_, model, _ := g.currentDefault(ctx)
	return model
}

// GenerateWorkflow creates a WorkflowDefinition from a description.
// Satisfies ports.WorkflowGenerator interface.
func (g *Generator) GenerateWorkflow(ctx context.Context, description string) (*upal.WorkflowDefinition, error) {
	return g.Generate(ctx, description, nil, nil)
}

// skillLoaderTool is the FunctionDeclaration that allows the generation LLM
// to load skill guides on demand instead of receiving all guides upfront.
var skillLoaderTool = &genai.Tool{
	FunctionDeclarations: []*genai.FunctionDeclaration{{
		Name:        "get_skill",
		Description: "Retrieve the full content of a skill guide by name. Call this before generating to understand configuration requirements for node or stage types you will use.",
		Parameters: &genai.Schema{
			Type: "OBJECT",
			Properties: map[string]*genai.Schema{
				"skill_name": {
					Type:        "STRING",
					Description: `Name of the skill guide to load (e.g. "agent-node", "stage-collect")`,
				},
			},
			Required: []string{"skill_name"},
		},
	}},
}

// Generate creates a WorkflowDefinition from a natural language description.
// If existingWorkflow is non-nil, the generator operates in edit mode — modifying the
// existing workflow according to the description instead of creating from scratch.
func (g *Generator) Generate(ctx context.Context, description string, existingWorkflow *upal.WorkflowDefinition, availableWorkflows []WorkflowSummary) (*upal.WorkflowDefinition, error) {
	var sysPrompt string
	if g.skills != nil {
		sysPrompt = g.skills.GetPrompt("workflow-create")
	}
	userContent := description

	if existingWorkflow != nil {
		if g.skills != nil {
			sysPrompt = g.skills.GetPrompt("workflow-edit")
		}
		wfJSON, err := json.MarshalIndent(existingWorkflow, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal existing workflow: %w", err)
		}
		userContent = fmt.Sprintf("Current workflow:\n%s\n\nInstruction: %s", string(wfJSON), description)
	}

	// Inject existing workflow summaries so the LLM understands what already exists.
	if len(availableWorkflows) > 0 {
		sysPrompt += "\n\nExisting workflows (for reference — avoid duplicating; choose a different name if creating a new workflow):\n"
		sysPrompt += formatWorkflowList(availableWorkflows)
	}

	// Inject available models grouped by category.
	if len(g.models) > 0 {
		sysPrompt += g.buildModelPrompt()
	}

	// Inject available tools so the LLM only references real tools.
	if len(g.toolInfos) > 0 {
		sysPrompt += "\n\nAvailable tools for agent nodes (use in config \"tools\" array):\n"
		for _, t := range g.toolInfos {
			sysPrompt += fmt.Sprintf("- %q — %s\n", t.Name, t.Description)
		}
		sysPrompt += "For detailed usage (parameters, return values, prompt patterns), call get_skill(\"tool-{name}\") e.g. get_skill(\"tool-web_search\").\n"
		sysPrompt += "IMPORTANT: ONLY use tools from this list. Do NOT invent or hallucinate tool names that are not listed here. If the task requires a capability not covered by these tools, use an agent node with a detailed prompt instead."
	}

	// Final reinforcement — must be last so it benefits from recency bias.
	sysPrompt += "\n\nIMPORTANT: Your entire response must be ONLY the raw JSON object. No markdown fences, no explanation, no commentary before or after the JSON."

	// Workflow generation requires careful instruction following, so request high effort.
	ctx = upalmodel.WithEffort(ctx, "high")

	content, err := g.generateWithSkills(ctx, sysPrompt, userContent, "generate workflow")
	if err != nil {
		return nil, err
	}

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

// generateWithSkills runs a multi-turn LLM call that allows the model to call
// get_skill() to load skill documentation on demand before producing the final JSON.
func (g *Generator) generateWithSkills(ctx context.Context, sysPrompt, userContent, opName string) (string, error) {
	llm, modelName, err := g.currentDefault(ctx)
	if err != nil {
		return "", err
	}

	genCfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		Tools:             []*genai.Tool{skillLoaderTool},
	}

	contents := []*genai.Content{
		genai.NewContentFromText(userContent, genai.RoleUser),
	}

	for turn := 0; turn < 10; turn++ {
		req := &adkmodel.LLMRequest{
			Model:    modelName,
			Config:   genCfg,
			Contents: contents,
		}

		var resp *adkmodel.LLMResponse
		for r, err := range llm.GenerateContent(ctx, req, false) {
			if err != nil {
				return "", fmt.Errorf("%s (turn %d): %w", opName, turn+1, err)
			}
			resp = r
		}

		if resp == nil || resp.Content == nil {
			return "", fmt.Errorf("%s: empty response (turn %d)", opName, turn+1)
		}

		// Check for skill tool calls.
		var toolCalls []*genai.FunctionCall
		for _, p := range resp.Content.Parts {
			if p.FunctionCall != nil {
				toolCalls = append(toolCalls, p.FunctionCall)
			}
		}

		if len(toolCalls) == 0 {
			// Final response — extract and strip markdown JSON wrapper.
			text := llmutil.ExtractText(resp)
			stripped, err := llmutil.StripMarkdownJSON(text)
			if err != nil {
				return "", fmt.Errorf("parse %s (model output may be malformed): %w\nraw output: %s", opName, err, text)
			}
			return stripped, nil
		}

		// Execute skill tool calls and continue to the next turn.
		contents = append(contents, resp.Content)
		contents = append(contents, g.executeSkillCalls(toolCalls))
	}

	return "", fmt.Errorf("%s: exceeded maximum turns without producing output", opName)
}

// executeSkillCalls handles get_skill function calls from the generation LLM.
func (g *Generator) executeSkillCalls(calls []*genai.FunctionCall) *genai.Content {
	parts := make([]*genai.Part, 0, len(calls))
	for _, fc := range calls {
		var result map[string]any
		if fc.Name == "get_skill" {
			skillName, _ := fc.Args["skill_name"].(string)
			content := g.skills.Get(skillName)
			if content == "" {
				result = map[string]any{"error": fmt.Sprintf("skill %q not found", skillName)}
			} else {
				result = map[string]any{"content": content}
			}
		} else {
			result = map[string]any{"error": fmt.Sprintf("unknown tool %q", fc.Name)}
		}
		parts = append(parts, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     fc.Name,
				Response: result,
			},
		})
	}
	return &genai.Content{Role: genai.RoleUser, Parts: parts}
}

// stripInvalidTools removes tool names from agent node configs that don't exist
// in the tool registry. This prevents hallucinated tools from reaching execution.
func (g *Generator) stripInvalidTools(wf *upal.WorkflowDefinition) {
	valid := make(map[string]bool, len(g.toolInfos))
	for _, t := range g.toolInfos {
		valid[t.Name] = true
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
	groups := map[string][]upal.ModelSummary{}
	for _, m := range g.models {
		groups[m.Category] = append(groups[m.Category], m)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n\nAvailable models for agent nodes and HTML-format output nodes (use in config \"model\" field):\nDefault model: %q\n", g.model)

	if text := groups["text"]; len(text) > 0 {
		b.WriteString("\nText/reasoning models — use for analysis, generation, conversation, tool-use, and any task that processes or produces text:\n")
		for _, m := range text {
			fmt.Fprintf(&b, "- %q [%s] — %s\n", m.ID, m.Tier, m.Hint)
		}
	}

	if image := groups["image"]; len(image) > 0 {
		b.WriteString("\nImage generation models — use ONLY when the task requires creating, editing, or generating images:\n")
		for _, m := range image {
			fmt.Fprintf(&b, "- %q — %s\n", m.ID, m.Hint)
		}
	}

	if tts := groups["tts"]; len(tts) > 0 {
		b.WriteString("\nTTS (text-to-speech) models — use ONLY when the task requires converting text to spoken audio:\n")
		for _, m := range tts {
			fmt.Fprintf(&b, "- %q — %s\n", m.ID, m.Hint)
		}
	}

	b.WriteString(`
MODEL SELECTION RULES:
1. ONLY use models from the lists above.
2. Choose the model category that matches the node's PURPOSE: text models for reasoning/text tasks, image models for image generation tasks, tts models for speech synthesis tasks.
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
	// Use pre-computed default; fall back to inline resolution for direct struct construction (tests).
	defaultID := g.defaultModelID
	if defaultID == "" {
		for _, m := range g.models {
			if _, after, ok := strings.Cut(m.ID, "/"); ok && after == g.model {
				defaultID = m.ID
				break
			}
		}
		if defaultID == "" {
			defaultID = g.models[0].ID
		}
	}
	for i, n := range wf.Nodes {
		switch n.Type {
		case upal.NodeTypeAgent:
			model, _ := n.Config["model"].(string)
			if model != "" && !valid[model] {
				wf.Nodes[i].Config["model"] = defaultID
			}
		case upal.NodeTypeOutput:
			format, _ := n.Config["output_format"].(string)
			if format != "html" {
				continue
			}
			model, _ := n.Config["model"].(string)
			if model == "" || !valid[model] {
				wf.Nodes[i].Config["model"] = defaultID
			}
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

