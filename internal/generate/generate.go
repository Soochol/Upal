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

var createBasePrompt = `You are a workflow generator for the Upal platform. Given a user's natural language description, you must produce a valid workflow JSON.

A workflow has:
- "name": a slug-style name (lowercase, hyphens)
- "version": always 1
- "nodes": array of node objects with {id, type, config}
- "edges": array of edge objects connecting nodes

Node types (ONLY these three types are valid — do NOT use any other type):
1. "input"  — collects user input (see INPUT NODE GUIDE below)
2. "agent"  — calls an AI model (see AGENT NODE GUIDE below)
3. "output" — produces final output (see OUTPUT NODE GUIDE below)

EXAMPLE — a "summarize article" workflow:
{
  "name": "article-summarizer",
  "version": 1,
  "nodes": [
    {"id": "article_url", "type": "input", "config": {"label": "기사 URL", "placeholder": "요약할 기사의 URL을 붙여넣으세요...", "description": "분석할 기사 URL"}},
    {"id": "summarizer", "type": "agent", "config": {"model": "anthropic/claude-sonnet-4-6", "label": "요약기", "system_prompt": "당신은 기사에서 핵심 인사이트를 추출하는 데 깊은 전문성을 가진 시니어 콘텐츠 분석가입니다. 중심 주제, 근거 자료, 실행 가능한 시사점을 파악하는 데 탁월합니다. 명확하고 전문적인 톤으로 구조화된 형식을 사용해 작성하세요.", "prompt": "다음 기사를 요약해 주세요:\n\n{{article_url}}", "output": "다음 형식으로 구조화된 요약을 제공하세요: 1) 한 단락 개요, 2) 핵심 포인트 목록, 3) 한 문장 결론.", "description": "기사를 핵심 포인트로 요약"}},
    {"id": "final_output", "type": "output", "config": {"label": "요약 결과", "system_prompt": "깔끔하고 미니멀한 읽기 레이아웃을 사용하세요. 넉넉한 여백과 중립적 색상 팔레트에 제목에 하나의 강조 색상을 사용하세요. Inter를 본문 글꼴로, 굵은 산세리프를 제목 글꼴로 설정하세요. 요약을 중앙 정렬 단일 컬럼에 명확한 섹션 구분선과 함께 표시하세요.", "prompt": "{{summarizer}}", "description": "생성된 요약을 표시"}}
  ],
  "edges": [{"from": "article_url", "to": "summarizer"}, {"from": "summarizer", "to": "final_output"}]
}

Rules:
- Every workflow must start with at least one "input" node and end with one "output" node.
- Agent prompts should use {{node_id}} template syntax to reference upstream node outputs.
- Node IDs should be descriptive slugs like "user_question", "summarizer", "final_output".
- Keep workflows minimal — only add nodes that are necessary for the described task.
- Every "agent" node MUST follow the AGENT NODE GUIDE below.
- Every "input" node MUST follow the INPUT NODE GUIDE below.
- Every "output" node MUST follow the OUTPUT NODE GUIDE below.
- Every node config MUST include "label" (human-readable name specific to the task) and "description".
- LANGUAGE: ALL user-facing text (label, description, placeholder, system_prompt, prompt, output) MUST be written in Korean (한국어). Node IDs and the workflow "name" field remain English slugs.`

var editBasePrompt = `You are a workflow editor for the Upal platform. You will be given an existing workflow JSON and a user's instruction to modify it. You must return the COMPLETE updated workflow JSON.

A workflow has:
- "name": a slug-style name (lowercase, hyphens)
- "version": always 1
- "nodes": array of node objects with {id, type, config}
- "edges": array of edge objects connecting nodes

Node types (ONLY these three types are valid — do NOT use any other type):
1. "input"  — collects user input (see INPUT NODE GUIDE below)
2. "agent"  — calls an AI model (see AGENT NODE GUIDE below)
3. "output" — produces final output (see OUTPUT NODE GUIDE below)

EXAMPLE — a "summarize article" workflow:
{
  "name": "article-summarizer",
  "version": 1,
  "nodes": [
    {"id": "article_url", "type": "input", "config": {"label": "기사 URL", "placeholder": "요약할 기사의 URL을 붙여넣으세요...", "description": "분석할 기사 URL"}},
    {"id": "summarizer", "type": "agent", "config": {"model": "anthropic/claude-sonnet-4-6", "label": "요약기", "system_prompt": "당신은 기사에서 핵심 인사이트를 추출하는 데 깊은 전문성을 가진 시니어 콘텐츠 분석가입니다. 중심 주제, 근거 자료, 실행 가능한 시사점을 파악하는 데 탁월합니다. 명확하고 전문적인 톤으로 구조화된 형식을 사용해 작성하세요.", "prompt": "다음 기사를 요약해 주세요:\n\n{{article_url}}", "output": "다음 형식으로 구조화된 요약을 제공하세요: 1) 한 단락 개요, 2) 핵심 포인트 목록, 3) 한 문장 결론.", "description": "기사를 핵심 포인트로 요약"}},
    {"id": "final_output", "type": "output", "config": {"label": "요약 결과", "system_prompt": "깔끔하고 미니멀한 읽기 레이아웃을 사용하세요. 넉넉한 여백과 중립적 색상 팔레트에 제목에 하나의 강조 색상을 사용하세요. Inter를 본문 글꼴로, 굵은 산세리프를 제목 글꼴로 설정하세요. 요약을 중앙 정렬 단일 컬럼에 명확한 섹션 구분선과 함께 표시하세요.", "prompt": "{{summarizer}}", "description": "생성된 요약을 표시"}}
  ],
  "edges": [{"from": "article_url", "to": "summarizer"}, {"from": "summarizer", "to": "final_output"}]
}

Rules:
- Preserve existing nodes and edges unless the user explicitly asks to remove or replace them.
- When adding new nodes, connect them logically to the existing graph.
- Agent prompts should use {{node_id}} template syntax to reference upstream node outputs.
- Node IDs should be descriptive slugs like "user_question", "summarizer", "final_output".
- Avoid duplicate node IDs — check against existing IDs before creating new ones.
- Every workflow must have at least one "input" node and one "output" node.
- Every "agent" node MUST follow the AGENT NODE GUIDE below.
- Every "input" node MUST follow the INPUT NODE GUIDE below.
- Every "output" node MUST follow the OUTPUT NODE GUIDE below.
- Every node config MUST include "label" (human-readable name specific to the task) and "description".
- LANGUAGE: ALL user-facing text (label, description, placeholder, system_prompt, prompt, output) MUST be written in Korean (한국어). Node IDs and the workflow "name" field remain English slugs.`

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
