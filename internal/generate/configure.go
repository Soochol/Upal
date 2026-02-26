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

// ---------- shared chat types ----------

// ChatMessage represents a single message in LLM configuration chat history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ---------- node configuration ----------

// ConfigureNodeInput holds the domain parameters for configuring a node via LLM.
type ConfigureNodeInput struct {
	NodeType      string
	NodeID        string
	CurrentConfig map[string]any
	Label         string
	Description   string
	Message       string
	Model         string // optional per-request model override
	Thinking      bool
	History       []ChatMessage
	UpstreamNodes []UpstreamNodeInfo
}

// UpstreamNodeInfo describes an upstream node for context in node configuration.
type UpstreamNodeInfo struct {
	ID    string
	Type  string
	Label string
}

// WorkflowRef is a lightweight workflow reference for pipeline configuration context.
type WorkflowRef struct {
	Name        string
	Description string
}

// ConfigureNodeOutput is the LLM-generated node configuration result.
type ConfigureNodeOutput struct {
	Config      map[string]any `json:"config"`
	Label       string         `json:"label,omitempty"`
	Description string         `json:"description,omitempty"`
	Explanation string         `json:"explanation"`
}

// ConfigureNode asks the LLM to configure a workflow node based on the user's message.
func (g *Generator) ConfigureNode(ctx context.Context, in ConfigureNodeInput) (*ConfigureNodeOutput, error) {
	llm, modelName := g.resolveLLM(in.Model)

	configJSON, err := json.Marshal(in.CurrentConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal current config: %w", err)
	}

	upstreamList := "none"
	if len(in.UpstreamNodes) > 0 {
		var parts []string
		for _, u := range in.UpstreamNodes {
			parts = append(parts, fmt.Sprintf("- %s (type=%s, label=%q)", u.ID, u.Type, u.Label))
		}
		upstreamList = strings.Join(parts, "\n")
	}

	contextMsg := fmt.Sprintf(
		"Current node: type=%s, id=%s, label=%q, description=%q\nCurrent config: %s\nUpstream nodes:\n%s\n\nUser request: %s",
		in.NodeType, in.NodeID, in.Label, in.Description, string(configJSON), upstreamList, in.Message,
	)

	contents := buildChatHistory(in.History)
	contents = append(contents, genai.NewContentFromText(contextMsg, genai.RoleUser))

	sysPrompt := ""
	if g.skills != nil {
		sysPrompt = g.skills.GetPrompt("node-configure")
		if nodeSkill := g.skills.Get(in.NodeType + "-node"); nodeSkill != "" {
			sysPrompt += "\n\n--- NODE TYPE GUIDE ---\n\n" + nodeSkill
		}
	}

	sysPrompt += g.buildModelCatalog(modelName)
	sysPrompt += "\nIMPORTANT: ONLY use models from this list. Match model category to the node's purpose."

	var out ConfigureNodeOutput
	if err := g.callAndParseJSON(ctx, llm, modelName, sysPrompt, contents, in.Thinking, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- pipeline configuration ----------

// ConfigurePipelineInput holds the domain parameters for configuring a pipeline via LLM.
type ConfigurePipelineInput struct {
	Message          string
	Model            string // optional per-request model override
	Thinking         bool
	History          []ChatMessage
	CurrentSources   json.RawMessage
	CurrentSchedule  string
	CurrentWorkflows json.RawMessage
	CurrentModel     string
	CurrentContext   json.RawMessage
	// Injected by the caller (HTTP handler) to enrich the system prompt.
	StageTypes         []string       // distinct stage types for skill injection
	AvailableWorkflows []WorkflowRef  // for workflow name reference
}

// ConfigurePipelineOutput is the LLM-generated pipeline configuration result.
type ConfigurePipelineOutput struct {
	Sources     json.RawMessage `json:"sources,omitempty"`
	Schedule    *string         `json:"schedule,omitempty"`
	Workflows   json.RawMessage `json:"workflows,omitempty"`
	Model       *string         `json:"model,omitempty"`
	Context     json.RawMessage `json:"context,omitempty"`
	Explanation string          `json:"explanation"`
}

// ConfigurePipeline asks the LLM to configure a pipeline based on the user's message.
func (g *Generator) ConfigurePipeline(ctx context.Context, in ConfigurePipelineInput) (*ConfigurePipelineOutput, error) {
	llm, modelName := g.resolveLLM(in.Model)

	contextMsg := fmt.Sprintf(
		"Current pipeline settings:\nSources: %s\nSchedule: %q\nWorkflows: %s\nModel: %q\nEditorial brief: %s\n\nUser request: %s",
		string(in.CurrentSources),
		in.CurrentSchedule,
		string(in.CurrentWorkflows),
		in.CurrentModel,
		string(in.CurrentContext),
		in.Message,
	)

	contents := buildChatHistory(in.History)
	contents = append(contents, genai.NewContentFromText(contextMsg, genai.RoleUser))

	sysPrompt := ""
	if g.skills != nil {
		sysPrompt = g.skills.GetPrompt("pipeline-configure")

		// Inject stage-type skills for the pipeline's stages.
		seen := map[string]bool{}
		for _, stType := range in.StageTypes {
			key := "stage-" + stType
			if !seen[key] {
				if skill := g.skills.Get(key); skill != "" {
					sysPrompt += "\n\n--- STAGE GUIDE: " + stType + " ---\n\n" + skill
				}
				seen[key] = true
			}
		}
	}

	sysPrompt += g.buildModelCatalog(modelName)

	// Inject available workflows so the LLM can reference real workflow names.
	if len(in.AvailableWorkflows) > 0 {
		sysPrompt += "\n\nAvailable workflows:\n"
		for _, wf := range in.AvailableWorkflows {
			desc := wf.Description
			if desc == "" {
				desc = wf.Name
			}
			sysPrompt += fmt.Sprintf("- %q (%s)\n", wf.Name, desc)
		}
	}

	var out ConfigurePipelineOutput
	if err := g.callAndParseJSON(ctx, llm, modelName, sysPrompt, contents, in.Thinking, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- internal helpers ----------

// buildChatHistory converts ChatMessage slices into genai.Content for LLM requests.
func buildChatHistory(history []ChatMessage) []*genai.Content {
	var contents []*genai.Content
	for _, h := range history {
		switch h.Role {
		case "user":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleUser))
		case "assistant":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleModel))
		}
	}
	return contents
}

// buildModelCatalog appends a compact model catalog to configure system prompts.
// Unlike buildModelPrompt (used in generation), this omits TTS/image model selection
// rules since configure only needs the model ID list for field assignment.
func (g *Generator) buildModelCatalog(modelName string) string {
	if len(g.models) == 0 {
		return ""
	}

	groups := map[string][]upal.ModelSummary{}
	for _, m := range g.models {
		groups[m.Category] = append(groups[m.Category], m)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n\nAvailable models (use in \"model\" field):\nDefault model: %q\n", modelName)

	if text := groups["text"]; len(text) > 0 {
		b.WriteString("\nText/reasoning models:\n")
		for _, m := range text {
			fmt.Fprintf(&b, "- %q [%s] — %s\n", m.ID, m.Tier, m.Hint)
		}
	}
	if image := groups["image"]; len(image) > 0 {
		b.WriteString("\nImage generation models:\n")
		for _, m := range image {
			fmt.Fprintf(&b, "- %q — %s\n", m.ID, m.Hint)
		}
	}

	return b.String()
}

// callAndParseJSON performs a single-turn LLM call and JSON-decodes the response into dest.
func (g *Generator) callAndParseJSON(ctx context.Context, llm adkmodel.LLM, modelName, sysPrompt string, contents []*genai.Content, thinking bool, dest any) error {
	llmReq := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
		},
		Contents: contents,
	}

	ctx = upalmodel.WithThinking(ctx, thinking)

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, llmReq, false) {
		if err != nil {
			return fmt.Errorf("LLM call failed: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return fmt.Errorf("empty response from LLM")
	}

	text := llmutil.ExtractText(resp)
	content, err := llmutil.StripMarkdownJSON(text)
	if err != nil {
		return fmt.Errorf("failed to parse LLM response: %w\nraw: %s", err, text)
	}

	if err := json.NewDecoder(strings.NewReader(content)).Decode(dest); err != nil {
		return fmt.Errorf("failed to parse LLM response: %w\nraw: %s", err, content)
	}

	return nil
}
