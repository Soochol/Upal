package output

import (
	"fmt"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/upal/ports"
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

// NewFormatter resolves the appropriate Formatter from node config and available LLMs.
// For "md" output_format, returns a PassthroughFormatter (no LLM call).
// For "html" (or unset), returns an HTMLFormatter if system_prompt is configured,
// otherwise falls back to PassthroughFormatter for backward compatibility.
// basePrompt contains platform-level constraints prepended to the user-authored system_prompt.
func NewFormatter(config map[string]any, resolver ports.LLMResolver, basePrompt string) Formatter {
	format, _ := config["output_format"].(string)

	switch format {
	case "md":
		return &PassthroughFormatter{}
	default: // "html" or legacy (no format set)
		systemPrompt, _ := config["system_prompt"].(string)
		if systemPrompt == "" || resolver == nil {
			return &PassthroughFormatter{}
		}
		modelID, _ := config["model"].(string)
		llm, modelName, err := resolver.Resolve(modelID)
		if err != nil || llm == nil {
			return &PassthroughFormatter{}
		}
		return &HTMLFormatter{
			LLM:          llm,
			ModelName:    modelName,
			SystemPrompt: basePrompt + "\n\n" + systemPrompt,
		}
	}
}
