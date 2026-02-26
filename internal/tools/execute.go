package tools

import (
	"context"
	"fmt"

	upalmodel "github.com/soochol/upal/internal/model"
	"google.golang.org/genai"
)

// ToGenaiSchema converts a map[string]any JSON schema (from Tool.InputSchema)
// to a *genai.Schema for use in genai.FunctionDeclaration.
func ToGenaiSchema(schema map[string]any) *genai.Schema {
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
					if items, ok := prop["items"].(map[string]any); ok {
						itemSchema := &genai.Schema{Type: genai.TypeString}
						if it, ok := items["type"].(string); ok {
							switch it {
							case "number":
								itemSchema.Type = genai.TypeNumber
							case "integer":
								itemSchema.Type = genai.TypeInteger
							case "boolean":
								itemSchema.Type = genai.TypeBoolean
							}
						}
						ps.Items = itemSchema
					} else {
						// Gemini requires Items for array types; default to string.
						ps.Items = &genai.Schema{Type: genai.TypeString}
					}
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

// ExecuteToolCalls runs function calls against custom tools and returns a
// Content with FunctionResponse parts for feeding back to the LLM.
// Native tool calls (e.g. web_search handled by the provider) are skipped.
func ExecuteToolCalls(ctx context.Context, calls []*genai.FunctionCall, customTools map[string]Tool) *genai.Content {
	var parts []*genai.Part
	for _, fc := range calls {
		t, ok := customTools[fc.Name]
		if !ok {
			// Native tool call — provider handles it; skip.
			continue
		}
		output := executeSingleToolSafe(ctx, fc, t)
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

func executeSingleToolSafe(ctx context.Context, fc *genai.FunctionCall, t Tool) (output map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			output = map[string]any{"error": fmt.Sprintf("tool %q panicked: %v", fc.Name, r)}
		}
	}()

	result, err := t.Execute(ctx, fc.Args)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return map[string]any{"result": fmt.Sprintf("%v", result)}
}

// ResolveToolSet resolves tool names into native genai.Tool specs and custom
// Tool instances using the registry and the global native tool specs.
// The llm parameter should implement model.NativeToolProvider for
// model-specific native tool support.
func ResolveToolSet(reg *Registry, llm any, toolNames []string) (
	nativeTools []*genai.Tool,
	customTools map[string]Tool,
	funcDecls []*genai.FunctionDeclaration,
	err error,
) {
	customTools = make(map[string]Tool)

	for _, name := range toolNames {
		// 1. Check global native tools (web_search, etc.).
		if spec, isGlobal := upalmodel.LookupNativeTool(name); isGlobal {
			if provider, ok := llm.(upalmodel.NativeToolProvider); ok {
				if modelSpec, supported := provider.NativeTool(name); supported {
					nativeTools = append(nativeTools, modelSpec)
				}
				// Model doesn't support — skip silently.
			} else {
				nativeTools = append(nativeTools, spec)
			}
			continue
		}

		// 2. Check registry native tools.
		if reg != nil && reg.IsNative(name) {
			if provider, ok := llm.(upalmodel.NativeToolProvider); ok {
				if modelSpec, supported := provider.NativeTool(name); supported {
					nativeTools = append(nativeTools, modelSpec)
				}
			}
			continue
		}

		// 3. Custom tool from registry.
		if reg == nil {
			err = fmt.Errorf("tool %q requested but no tool registry configured", name)
			return
		}
		t, found := reg.Get(name)
		if !found {
			err = fmt.Errorf("unknown tool %q", name)
			return
		}
		customTools[name] = t
		funcDecls = append(funcDecls, &genai.FunctionDeclaration{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  ToGenaiSchema(t.InputSchema()),
		})
	}
	return
}
