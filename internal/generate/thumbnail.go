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

const thumbnailSystemPrompt = `You are an SVG graphic designer creating card thumbnails for an AI workflow platform.

Generate a beautiful, minimal SVG banner image that visually represents an AI workflow.

STRICT REQUIREMENTS:
- SVG dimensions: width="300" height="68" viewBox="0 0 300 68"
- Use 2-4 harmonious colors with hardcoded hex values only (e.g., #3b82f6) — NO CSS variables, NO CSS classes, NO style attributes with CSS
- Use only basic SVG shapes: rect, circle, path, polygon, line, ellipse, linearGradient, radialGradient, defs
- NO text, NO labels, NO <script> tags, NO <foreignObject> tags, NO external references
- Flat, abstract, or geometric design that visually suggests the workflow's domain or purpose
- The image should look professional and polished as a small card thumbnail
- Make the design feel specific to the described workflow — not generic

Return ONLY the complete SVG element. Start your response with <svg and end with </svg>. No markdown fences, no explanation.`

// GenerateThumbnail asks the LLM to create a small SVG banner that visually
// represents the given workflow. The returned SVG is sanitized. Errors are
// non-fatal — callers should treat a failure as "no thumbnail".
func (g *Generator) GenerateThumbnail(ctx context.Context, wf *upal.WorkflowDefinition) (string, error) {
	userPrompt := buildThumbnailPrompt(wf)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(thumbnailSystemPrompt, genai.RoleUser),
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
