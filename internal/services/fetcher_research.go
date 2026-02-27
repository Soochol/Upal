package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

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
	toolReg    *tools.Registry
	progressFn ResearchProgressFn
}

func (f *researchFetcher) SetProgressFn(fn ResearchProgressFn) {
	f.progressFn = fn
}

func NewResearchFetcher(resolver ports.LLMResolver, skills skills.Provider, toolReg *tools.Registry) *researchFetcher {
	return &researchFetcher{resolver: resolver, skills: skills, toolReg: toolReg}
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

	// Resolve tools from registry (unified with agent node tool system).
	toolNames := []string{"web_search", "get_webpage"}
	nativeTools, customTools, funcDecls, err := tools.ResolveToolSet(f.toolReg, llm, toolNames)
	if err != nil {
		return "", nil, fmt.Errorf("resolve research tools: %w", err)
	}

	var allTools []*genai.Tool
	allTools = append(allTools, nativeTools...)
	if len(funcDecls) > 0 {
		allTools = append(allTools, &genai.Tool{FunctionDeclarations: funcDecls})
	}

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
		if toolResults := tools.ExecuteToolCalls(ctx, toolCalls, customTools); toolResults != nil {
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
		slog.Warn("stage-research skill not found; using hardcoded fallback")
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
