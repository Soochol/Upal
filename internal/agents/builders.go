package agents

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// BuildAgent creates an ADK Agent from a NodeDefinition using the default
// registry. This is a backward-compatible convenience function.
func BuildAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	return DefaultRegistry().Build(nd, BuildDeps{
		LLMs:    llms,
		ToolReg: toolReg,
	})
}

// templatePattern matches {{key}} or {{key.subkey}} placeholders.
var templatePattern = regexp.MustCompile(`\{\{(\w+(?:\.\w+)*)\}\}`)

// namedLLM wraps an LLM to override Name() with a specific model name.
// ADK uses Name() as req.Model in API requests, so each agent node needs
// an LLM whose Name() returns the actual model name (e.g., "qwen3:32b"),
// not the provider name (e.g., "ollama").
type namedLLM struct {
	adkmodel.LLM
	name string
}

func (n *namedLLM) Name() string { return n.name }

// resolveLLM resolves a "provider/model" format model ID into an LLM instance
// and the bare model name. Falls back to the first available LLM if the
// specified provider is not found. Returns (nil, "") if no LLMs are available.
func resolveLLM(modelID string, llms map[string]adkmodel.LLM) (adkmodel.LLM, string) {
	if modelID != "" && llms != nil {
		parts := strings.SplitN(modelID, "/", 2)
		providerName := parts[0]
		if l, ok := llms[providerName]; ok {
			if len(parts) == 2 {
				return &namedLLM{LLM: l, name: parts[1]}, parts[1]
			}
			return l, ""
		}
	}

	// Fallback: first available LLM
	for _, l := range llms {
		return l, ""
	}
	return nil, ""
}

// resolveTemplateFromState replaces {{key}} placeholders in a template string
// with values from session state. Unresolved placeholders are left as-is.
func resolveTemplateFromState(template string, state session.State) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		key := strings.Trim(match, "{}")
		val, err := state.Get(key)
		if err != nil || val == nil {
			return match
		}
		return fmt.Sprintf("%v", val)
	})
}

// toGenaiSchema converts a map[string]any JSON schema (from tools.Tool.InputSchema)
// to a *genai.Schema for use in genai.FunctionDeclaration.
func toGenaiSchema(schema map[string]any) *genai.Schema {
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
				case "array":
					ps.Type = genai.TypeArray
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

// buildPromptParts converts a resolved prompt string into genai Parts.
// Segments that are bare data URIs (from asset image nodes) become inline
// image parts; everything else becomes text parts.
func buildPromptParts(prompt string) []*genai.Part {
	if !strings.Contains(prompt, "data:image/") {
		return []*genai.Part{genai.NewPartFromText(prompt)}
	}

	var parts []*genai.Part
	remaining := prompt
	for {
		idx := strings.Index(remaining, "data:image/")
		if idx == -1 {
			if remaining != "" {
				parts = append(parts, genai.NewPartFromText(remaining))
			}
			break
		}
		if idx > 0 {
			parts = append(parts, genai.NewPartFromText(remaining[:idx]))
		}
		rest := remaining[idx:]
		// Data URIs end at whitespace or end-of-string
		end := strings.IndexAny(rest, " \n\r\t")
		var uri string
		if end == -1 {
			uri = rest
			remaining = ""
		} else {
			uri = rest[:end]
			remaining = strings.TrimLeft(rest[end:], " \n\r\t")
		}
		if p := parseDataURIPart(uri); p != nil {
			parts = append(parts, p)
		} else {
			slog.Warn("buildPromptParts: failed to parse data URI, falling back to text", "uri_prefix", uri[:min(len(uri), 40)])
			parts = append(parts, genai.NewPartFromText(uri))
		}
	}
	return parts
}

// parseDataURIPart parses a data URI string and returns a genai inline image part.
// Returns nil if the URI is not a valid base64 data URI.
func parseDataURIPart(uri string) *genai.Part {
	if !strings.HasPrefix(uri, "data:") {
		return nil
	}
	rest := uri[5:]
	semi := strings.Index(rest, ";")
	if semi == -1 {
		return nil
	}
	mimeType := rest[:semi]
	rest = rest[semi+1:]
	if !strings.HasPrefix(rest, "base64,") {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(rest[7:])
	if err != nil {
		return nil
	}
	return &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: mimeType,
			Data:     data,
		},
	}
}

// aspectRatioToSize converts a ratio string like "16:9" to width/height pixels
// using 1024 as the base dimension.
func aspectRatioToSize(ratio string) (int, int) {
	ratios := map[string][2]int{
		"1:1":  {1024, 1024},
		"16:9": {1024, 576},
		"9:16": {576, 1024},
		"4:3":  {1024, 768},
		"3:4":  {768, 1024},
		"3:2":  {1024, 680},
		"2:3":  {680, 1024},
	}
	if wh, ok := ratios[ratio]; ok {
		return wh[0], wh[1]
	}
	return 1024, 1024
}
