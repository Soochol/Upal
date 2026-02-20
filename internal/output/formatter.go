package output

import (
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// Formatter transforms collected upstream content into the final output format.
type Formatter interface {
	Format(ctx agent.InvocationContext, content string) (string, error)
}

// HTMLFormatter uses an LLM to generate a styled HTML page from upstream content.
type HTMLFormatter struct {
	LLM          adkmodel.LLM
	ModelName    string
	SystemPrompt string // baseLayoutConstraints + user-authored design direction
}

func (f *HTMLFormatter) Format(ctx agent.InvocationContext, content string) (string, error) {
	if f.LLM == nil {
		return "", fmt.Errorf("no LLM available for HTML layout generation")
	}

	req := &adkmodel.LLMRequest{
		Model: f.ModelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(f.SystemPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(content, genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range f.LLM.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("HTML layout LLM call: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	text := strings.TrimSpace(llmutil.ExtractText(resp))
	text = strings.TrimPrefix(text, "```html")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text), nil
}

// PassthroughFormatter returns content unchanged. Used for Markdown output.
type PassthroughFormatter struct{}

func (f *PassthroughFormatter) Format(_ agent.InvocationContext, content string) (string, error) {
	return content, nil
}

// BaseLayoutConstraints contains platform-level constraints prepended to the
// user-authored system_prompt when generating HTML layout.
var BaseLayoutConstraints = `You are an AI Web Developer. Your task is to generate a single, self-contained HTML document for rendering in an iframe, based on the provided content data from a workflow.

**Libraries:**
* Use Tailwind for CSS via CDN: ` + "`<script src=\"https://cdn.tailwindcss.com\"></script>`" + `
* Google Fonts are allowed for typography imports.
* **Tailwind Configuration**: Extend the Tailwind configuration within a ` + "`<script>`" + ` block to define custom font families and color palettes that match the theme.

**Constraints:**
* The output must be a complete and valid HTML document with no placeholder content.
* **Media Restriction:** ONLY use media URLs that are explicitly present in the input data. Do NOT generate or hallucinate any media URLs.
* **Render All Media:** You MUST render ALL media (images, videos, audio) that are present in the data. Every provided media URL must appear in the final HTML output.
* **Navigation Restriction:** Do NOT generate unneeded fake links or buttons to sub-pages (e.g. "About", "Contact", "Learn More") unless the data explicitly calls for them.
* **Footer Restriction:** NEVER generate any footer content, including legal footers like "All rights reserved" or "Copyright".
* Output ONLY the HTML document, no explanation or markdown fences.`

// NewFormatter resolves the appropriate Formatter from node config and available LLMs.
// For "md" output_format, returns a PassthroughFormatter (no LLM call).
// For "html" (or unset), returns an HTMLFormatter if system_prompt is configured,
// otherwise falls back to PassthroughFormatter for backward compatibility.
func NewFormatter(config map[string]any, llms map[string]adkmodel.LLM, resolveLLM func(string, map[string]adkmodel.LLM) (adkmodel.LLM, string)) Formatter {
	format, _ := config["output_format"].(string)

	switch format {
	case "md":
		return &PassthroughFormatter{}
	default: // "html" or legacy (no format set)
		systemPrompt, _ := config["system_prompt"].(string)
		if systemPrompt == "" || llms == nil {
			return &PassthroughFormatter{}
		}
		modelID, _ := config["model"].(string)
		llm, modelName := resolveLLM(modelID, llms)
		if llm == nil {
			return &PassthroughFormatter{}
		}
		return &HTMLFormatter{
			LLM:          llm,
			ModelName:    modelName,
			SystemPrompt: BaseLayoutConstraints + "\n\n" + systemPrompt,
		}
	}
}
