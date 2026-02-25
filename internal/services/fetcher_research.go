package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"github.com/soochol/upal/internal/upal/ports"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// researchFetcher uses an LLM with web_search and get_webpage tools to
// research a topic. It supports two modes: "light" (single search pass) and
// "deep" (multi-round investigation loop).
type researchFetcher struct {
	resolver ports.LLMResolver
	skills   skills.Provider
}

// NewResearchFetcher creates a researchFetcher that drives an LLM through a
// manual tool-call loop (no ADK agent framework) to perform web research.
func NewResearchFetcher(resolver ports.LLMResolver, skills skills.Provider) *researchFetcher {
	return &researchFetcher{resolver: resolver, skills: skills}
}

func (f *researchFetcher) Type() string { return "research" }

func (f *researchFetcher) Fetch(ctx context.Context, src upal.CollectSource) (string, any, error) {
	if src.Topic == "" {
		return "", nil, fmt.Errorf("research source requires a topic")
	}

	modelID := src.Model
	llm, modelName, err := f.resolver.Resolve(modelID)
	if err != nil {
		return "", nil, fmt.Errorf("resolve model %q: %w", modelID, err)
	}

	// Validate model supports native tools (web_search).
	if _, ok := llm.(upalmodel.NativeToolProvider); !ok {
		return "", nil, fmt.Errorf("model %q does not support web search (native tools required)", modelID)
	}

	depth := src.Depth
	if depth == "" {
		depth = "light"
	}

	switch depth {
	case "light":
		return f.runResearch(ctx, src, llm, modelName, 1, 30*time.Second)
	case "deep":
		maxSearches := src.MaxSearches
		if maxSearches <= 0 {
			maxSearches = 10
		}
		return f.runResearch(ctx, src, llm, modelName, maxSearches, 5*time.Minute)
	default:
		return "", nil, fmt.Errorf("unknown research depth %q", depth)
	}
}

// runResearch drives the LLM in a generate-content loop, executing tool calls
// manually (same pattern as llm_builder.go). It counts web_search invocations
// and stops when maxSearches is reached or the LLM returns a final text answer.
func (f *researchFetcher) runResearch(
	ctx context.Context,
	src upal.CollectSource,
	llm adkmodel.LLM,
	modelName string,
	maxSearches int,
	timeout time.Duration,
) (string, any, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// --- system prompt from skill file ---
	skillContent := f.skills.Get("stage-research")
	systemPrompt := buildResearchSystemPrompt(skillContent, src.Depth)

	// --- build tools ---
	var nativeTools []*genai.Tool
	if spec, isGlobalNative := upalmodel.LookupNativeTool("web_search"); isGlobalNative {
		if provider, ok := llm.(upalmodel.NativeToolProvider); ok {
			if modelSpec, supported := provider.NativeTool("web_search"); supported {
				nativeTools = append(nativeTools, modelSpec)
			}
		} else {
			nativeTools = append(nativeTools, spec)
		}
	}

	webpageTool := &tools.GetWebpageTool{}
	funcDecls := []*genai.FunctionDeclaration{{
		Name:        webpageTool.Name(),
		Description: webpageTool.Description(),
		Parameters:  researchToGenaiSchema(webpageTool.InputSchema()),
	}}

	var allTools []*genai.Tool
	allTools = append(allTools, nativeTools...)
	allTools = append(allTools, &genai.Tool{FunctionDeclarations: funcDecls})

	genCfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		Tools:             allTools,
	}

	userPrompt := fmt.Sprintf("Research the following topic: %s", src.Topic)
	contents := []*genai.Content{
		{Role: genai.RoleUser, Parts: []*genai.Part{genai.NewPartFromText(userPrompt)}},
	}

	// --- agent loop ---
	// Each "search" may need multiple turns (search + read pages), so allow
	// generous headroom on total loop iterations.
	maxLoopTurns := max(maxSearches*3, 6)
	searchCount := 0

	for turn := 0; turn < maxLoopTurns; turn++ {
		req := &adkmodel.LLMRequest{
			Model:    modelName,
			Config:   genCfg,
			Contents: contents,
		}

		var resp *adkmodel.LLMResponse
		for r, err := range llm.GenerateContent(ctx, req, false) {
			if err != nil {
				return "", nil, fmt.Errorf("LLM call failed: %w", err)
			}
			resp = r
		}

		if resp == nil || resp.Content == nil {
			return "", nil, fmt.Errorf("empty LLM response")
		}

		// Check for tool calls.
		var toolCalls []*genai.FunctionCall
		for _, p := range resp.Content.Parts {
			if p.FunctionCall != nil {
				toolCalls = append(toolCalls, p.FunctionCall)
			}
		}

		if len(toolCalls) == 0 {
			// No tool calls -- extract final text result.
			text := researchExtractText(resp.Content)
			sources := parseResearchSources(text)
			return fmt.Sprintf("=== Research: %s ===\n\n%s", src.Topic, text), sources, nil
		}

		// Count web_search calls. The LLM provider handles web_search
		// natively so we don't execute it ourselves, but we still track
		// how many times the model invoked it.
		for _, fc := range toolCalls {
			if fc.Name == "web_search" || fc.Name == "google_search" {
				searchCount++
			}
		}

		// Execute tool calls and append results.
		contents = append(contents, resp.Content)
		toolResults := executeResearchToolCalls(ctx, toolCalls, webpageTool)
		contents = append(contents, toolResults)

		// If we've exhausted the search budget, ask the LLM to wrap up.
		if searchCount >= maxSearches {
			contents = append(contents, &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{genai.NewPartFromText("You have used all available search queries. Please synthesize your findings into a final report now.")},
			})
		}
	}

	return "", nil, fmt.Errorf("research exceeded max turns (%d)", maxLoopTurns)
}

// buildResearchSystemPrompt extracts the appropriate section (light or deep)
// from the stage-research skill markdown.
func buildResearchSystemPrompt(skillContent, depth string) string {
	if skillContent == "" {
		// Fallback if skill file is missing.
		if depth == "deep" {
			return "You are an expert research analyst. Use web_search and get_webpage to thoroughly investigate the topic. Provide a structured markdown report with sources."
		}
		return "You are a research analyst. Use web_search and get_webpage to find information about the topic. Provide a concise markdown report with sources."
	}

	// The skill file has two sections:
	//   "## Light Mode — System Prompt" and "## Deep Mode — System Prompt"
	// Each section contains the system prompt for that mode.
	if depth == "deep" {
		if _, after, ok := strings.Cut(skillContent, "## Deep Mode — System Prompt"); ok {
			return strings.TrimSpace(after)
		}
	} else {
		if _, after, ok := strings.Cut(skillContent, "## Light Mode — System Prompt"); ok {
			// Trim at the next top-level heading.
			if before, _, ok := strings.Cut(after, "## Deep Mode — System Prompt"); ok {
				return strings.TrimSpace(before)
			}
			return strings.TrimSpace(after)
		}
	}

	// If parsing fails, return the entire skill content.
	return strings.TrimSpace(skillContent)
}

// researchExtractText concatenates all text parts from a genai.Content.
func researchExtractText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var sb strings.Builder
	for _, p := range content.Parts {
		if p.Text != "" {
			sb.WriteString(p.Text)
		}
	}
	return strings.TrimSpace(sb.String())
}

// markdownLinkRe matches [Title](URL) patterns in markdown text.
var markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

// parseResearchSources extracts URLs and titles from markdown-formatted
// source links in the LLM's research output.
func parseResearchSources(text string) []map[string]any {
	matches := markdownLinkRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var sources []map[string]any
	for _, m := range matches {
		url := m[2]
		if seen[url] {
			continue
		}
		seen[url] = true
		sources = append(sources, map[string]any{
			"title": m[1],
			"url":   url,
		})
	}
	return sources
}

// executeResearchToolCalls executes tool calls and returns a Content with
// FunctionResponse parts for feeding back to the LLM.
//
// web_search is handled natively by the provider, so we only execute
// get_webpage calls ourselves. For any other tool name the result indicates
// it's provider-managed.
func executeResearchToolCalls(ctx context.Context, calls []*genai.FunctionCall, webpageTool *tools.GetWebpageTool) *genai.Content {
	var parts []*genai.Part
	for _, fc := range calls {
		var output map[string]any
		switch fc.Name {
		case webpageTool.Name():
			result, err := webpageTool.Execute(ctx, fc.Args)
			if err != nil {
				output = map[string]any{"error": err.Error()}
			} else if m, ok := result.(map[string]any); ok {
				output = m
			} else {
				output = map[string]any{"result": fmt.Sprintf("%v", result)}
			}
		default:
			// Native tools (web_search, google_search) are handled by the
			// provider; we don't need to return a FunctionResponse for them.
			continue
		}
		parts = append(parts, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     fc.Name,
				Response: output,
			},
		})
	}
	if len(parts) == 0 {
		// All calls were native -- return an empty content so the loop
		// can continue without appending a nil content.
		return &genai.Content{Role: genai.RoleUser, Parts: []*genai.Part{}}
	}
	return &genai.Content{
		Role:  genai.RoleUser,
		Parts: parts,
	}
}

// researchToGenaiSchema converts a map[string]any JSON schema to a *genai.Schema.
// This is a simplified version of the same conversion in internal/agents/builders.go.
func researchToGenaiSchema(schema map[string]any) *genai.Schema {
	if schema == nil {
		return nil
	}
	s := &genai.Schema{Type: genai.TypeObject}
	if props, ok := schema["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*genai.Schema)
		for k, v := range props {
			prop, _ := v.(map[string]any)
			ps := &genai.Schema{}
			if t, ok := prop["type"].(string); ok {
				switch t {
				case "string":
					ps.Type = genai.TypeString
				case "number":
					ps.Type = genai.TypeNumber
				case "integer":
					ps.Type = genai.TypeInteger
				case "boolean":
					ps.Type = genai.TypeBoolean
				default:
					ps.Type = genai.TypeString
				}
			}
			if d, ok := prop["description"].(string); ok {
				ps.Description = d
			}
			s.Properties[k] = ps
		}
	}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if rs, ok := r.(string); ok {
				s.Required = append(s.Required, rs)
			}
		}
	}
	return s
}
