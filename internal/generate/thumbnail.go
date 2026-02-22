package generate

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// GenerateThumbnail asks the LLM to create a small SVG banner that visually
// represents the given workflow. The returned SVG is sanitized. Errors are
// non-fatal — callers should treat a failure as "no thumbnail".
func (g *Generator) GenerateThumbnail(ctx context.Context, wf *upal.WorkflowDefinition) (string, error) {
	userPrompt := buildThumbnailPrompt(wf)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(g.skills.GetPrompt("thumbnail"), genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userPrompt, genai.RoleUser),
		},
	}

	// Use default (non-high) effort for thumbnail — it's a best-effort visual.
	ctx = upalmodel.WithEffort(ctx, "low")

	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("generate thumbnail: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	raw := llmutil.ExtractText(resp)
	svg, err := extractSVG(raw)
	if err != nil {
		return "", fmt.Errorf("extract SVG from response: %w", err)
	}
	return svg, nil
}

// buildThumbnailPrompt creates the user message for thumbnail generation.
func buildThumbnailPrompt(wf *upal.WorkflowDefinition) string {
	// Count node types
	typeCounts := make(map[upal.NodeType]int)
	for _, n := range wf.Nodes {
		typeCounts[n.Type]++
	}

	var nodeDesc []string
	for nt, count := range typeCounts {
		nodeDesc = append(nodeDesc, fmt.Sprintf("%d %s", count, nt))
	}

	// Extract first agent's prompt text for semantic context (first 120 chars)
	agentContext := ""
	for _, n := range wf.Nodes {
		if n.Type == upal.NodeTypeAgent {
			if p, ok := n.Config["prompt"].(string); ok && len(strings.TrimSpace(p)) > 5 {
				p = strings.TrimSpace(p)
				if len(p) > 120 {
					p = p[:120] + "…"
				}
				agentContext = fmt.Sprintf(" The main task is: \"%s\"", p)
				break
			}
		}
	}

	return fmt.Sprintf(
		"Create a thumbnail for an AI workflow named %q.\nNodes: %s.%s\nDesign a visual that captures the essence of this workflow.",
		wf.Name,
		strings.Join(nodeDesc, ", "),
		agentContext,
	)
}

// GeneratePipelineThumbnail asks the LLM to create a small SVG banner that
// visually represents the given pipeline. The returned SVG is sanitized.
// Errors are non-fatal — callers should treat a failure as "no thumbnail".
func (g *Generator) GeneratePipelineThumbnail(ctx context.Context, p *upal.Pipeline) (string, error) {
	userPrompt := buildPipelineThumbnailPrompt(p)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(g.skills.GetPrompt("thumbnail"), genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userPrompt, genai.RoleUser),
		},
	}

	ctx = upalmodel.WithEffort(ctx, "low")

	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("generate pipeline thumbnail: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	raw := llmutil.ExtractText(resp)
	svg, err := extractSVG(raw)
	if err != nil {
		return "", fmt.Errorf("extract SVG from response: %w", err)
	}
	return svg, nil
}

// buildPipelineThumbnailPrompt creates the user message for pipeline thumbnail generation.
func buildPipelineThumbnailPrompt(p *upal.Pipeline) string {
	typeCounts := make(map[string]int)
	for _, s := range p.Stages {
		typeCounts[s.Type]++
	}

	var stageDesc []string
	for st, count := range typeCounts {
		stageDesc = append(stageDesc, fmt.Sprintf("%d %s", count, st))
	}

	desc := ""
	if p.Description != "" {
		desc = fmt.Sprintf(" Description: %q.", p.Description)
	}

	return fmt.Sprintf(
		"Create a thumbnail for a pipeline named %q.\nStages: %s.%s\nDesign a visual that captures this pipeline's purpose.",
		p.Name,
		strings.Join(stageDesc, ", "),
		desc,
	)
}

var (
	reScript        = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reForeignObject = regexp.MustCompile(`(?is)<foreignObject[^>]*>.*?</foreignObject>`)
	reMarkdownFence = regexp.MustCompile("(?m)^```[a-z]*\\s*|^```\\s*$")
)

// extractSVG pulls the <svg>...</svg> element out of an LLM response,
// strips unsafe tags, and enforces a size limit.
func extractSVG(text string) (string, error) {
	// Remove markdown code fences
	text = reMarkdownFence.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	start := strings.Index(strings.ToLower(text), "<svg")
	end := strings.LastIndex(strings.ToLower(text), "</svg>")
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("no valid <svg>...</svg> element found")
	}
	svg := text[start : end+6]

	// Remove unsafe tags
	svg = reScript.ReplaceAllString(svg, "")
	svg = reForeignObject.ReplaceAllString(svg, "")

	// Size guard
	if len(svg) > 50_000 {
		return "", fmt.Errorf("SVG too large (%d bytes)", len(svg))
	}
	if len(svg) < 30 {
		return "", fmt.Errorf("SVG too small, likely invalid")
	}

	return svg, nil
}
