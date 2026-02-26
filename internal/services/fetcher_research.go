package services

import (
	"context"
	"fmt"
	"log/slog"
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

type ResearchProgressFn func(progress upal.ResearchProgress)

type researchFetcher struct {
	resolver   ports.LLMResolver
	skills     skills.Provider
	progressFn ResearchProgressFn
}

func (f *researchFetcher) SetProgressFn(fn ResearchProgressFn) {
	f.progressFn = fn
}

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

	if _, ok := llm.(upalmodel.NativeToolProvider); !ok {
		return "", nil, fmt.Errorf("model %q does not support web search (native tools required)", modelID)
	}

	depth := src.Depth
	if depth == "" {
		depth = "light"
	}

	switch depth {
	case "light":
		return f.runResearch(ctx, src, llm, modelName, 3, 30*time.Second)
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

	skillContent := f.skills.Get("stage-research")
	systemPrompt := buildResearchSystemPrompt(skillContent, src.Depth)

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

	maxLoopTurns := max(maxSearches*3, 6)
	searchCount := 0
	findingsCount := 0
	budgetMsgSent := false

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

		var toolCalls []*genai.FunctionCall
		for _, p := range resp.Content.Parts {
			if p.FunctionCall != nil {
				toolCalls = append(toolCalls, p.FunctionCall)
			}
		}

		if len(toolCalls) == 0 {
			text := researchExtractText(resp.Content)
			sources := parseResearchSources(text)
			return fmt.Sprintf("=== Research: %s ===\n\n%s", src.Topic, text), sources, nil
		}

		lastQuery := ""
		for _, fc := range toolCalls {
			if fc.Name == "web_search" || fc.Name == "google_search" {
				searchCount++
				if q, ok := fc.Args["query"].(string); ok {
					lastQuery = q
				}
			}
			if fc.Name == "get_webpage" {
				findingsCount++
			}
		}

		if f.progressFn != nil && searchCount > 0 {
			f.progressFn(upal.ResearchProgress{
				CurrentStep:   searchCount,
				MaxSteps:      maxSearches,
				CurrentQuery:  lastQuery,
				FindingsCount: findingsCount,
			})
		}

		contents = append(contents, resp.Content)
		if toolResults := executeResearchToolCalls(ctx, toolCalls, webpageTool); toolResults != nil {
			contents = append(contents, toolResults)
		}

		if searchCount >= maxSearches && !budgetMsgSent {
			budgetMsgSent = true
			contents = append(contents, &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{genai.NewPartFromText("You have used all available search queries. Please synthesize your findings into a final report now.")},
			})
		}
	}

	return "", nil, fmt.Errorf("research exceeded max turns (%d)", maxLoopTurns)
}

func buildResearchSystemPrompt(skillContent, depth string) string {
	if skillContent == "" {
		slog.Warn("stage-research skill not found; using full skill content as fallback")
		return "You are a research analyst. Use web_search and get_webpage to research the topic. Provide a structured markdown report with sources."
	}

	if depth == "deep" {
		if _, after, ok := strings.Cut(skillContent, "## Deep Mode — System Prompt"); ok {
			return strings.TrimSpace(after)
		}
	} else {
		if _, after, ok := strings.Cut(skillContent, "## Light Mode — System Prompt"); ok {
			if before, _, ok := strings.Cut(after, "## Deep Mode — System Prompt"); ok {
				return strings.TrimSpace(before)
			}
			return strings.TrimSpace(after)
		}
	}

	return strings.TrimSpace(skillContent)
}

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

var markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)]+)\)`)

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
			"title":   m[1],
			"url":     url,
			"summary": m[1],
		})
	}
	return sources
}

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
		return nil
	}
	return &genai.Content{
		Role:  genai.RoleUser,
		Parts: parts,
	}
}

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
