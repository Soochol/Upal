package agents

import (
	"fmt"
	"iter"
	"regexp"
	"sort"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/remoteagent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// BuildAgent creates an ADK Agent from a NodeDefinition.
func BuildAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	switch nd.Type {
	case upal.NodeTypeInput:
		return buildInputAgent(nd)
	case upal.NodeTypeOutput:
		return buildOutputAgent(nd, llms)
	case upal.NodeTypeTool:
		return buildToolAgent(nd, toolReg)
	case upal.NodeTypeAgent:
		return buildLLMAgent(nd, llms, toolReg)
	case upal.NodeTypeExternal:
		return buildRemoteAgent(nd)
	default:
		return nil, fmt.Errorf("unknown node type %q for node %q", nd.Type, nd.ID)
	}
}

// buildInputAgent creates a custom Agent that reads user input from session state.
// It reads __user_input__{nodeID} from state, stores it under the node ID, and yields an event.
func buildInputAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Input node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()
				key := "__user_input__" + nodeID
				val, err := state.Get(key)
				if err != nil || val == nil {
					val = ""
				}

				_ = state.Set(nodeID, val)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(fmt.Sprintf("%v", val))},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = val
				yield(event, nil)
			}
		},
	})
}

// buildOutputAgent creates a custom Agent that collects all non-__ prefixed state keys,
// sorts them, joins their string values, and stores the result under the node ID.
// autoLayoutSystemPrompt instructs the LLM to generate a premium styled HTML page from workflow output.
var autoLayoutSystemPrompt = `You are an AI Web Developer. Your task is to generate a single, self-contained HTML document for rendering in an iframe, based on the provided content data from a workflow.

**Visual Aesthetic:**
* Aesthetics are crucial. Make the page look amazing, especially on mobile.
* **CRITICAL: Aim for premium, state-of-the-art designs. Avoid simple minimum viable products.**
* **Use Rich Aesthetics**: The viewer should be wowed at first glance by the design. Use best practices in modern web design (e.g. vibrant colors, dark modes, glassmorphism, and dynamic animations) to create a stunning first impression.
* **Prioritize Visual Excellence**: Implement designs that feel extremely premium:
    - Avoid generic colors (plain red, blue, green). Use curated, harmonious color palettes (e.g., HSL tailored colors, sleek dark modes).
    - Use modern typography (e.g., from Google Fonts like Inter, Roboto, or Outfit) instead of browser defaults.
    - Use smooth gradients.
    - Add subtle micro-animations for enhanced user experience.
* **Use a Dynamic Design**: An interface that feels responsive and alive encourages interaction. Achieve this with hover effects and interactive elements. Micro-animations are highly effective for improving user engagement.
* **Thematic Specificity**: Do not just create a generic layout. Define a clear "vibe" or theme based on the content. Use specific aesthetic keywords (e.g., "Glassmorphism", "Neobrutalism", "Minimalist") to guide the design.
* **Typography Hierarchy**: Explicitly import and use font pairings. Use a distinct Display Font for headers and a highly readable Body Font for text.
* **Readability**: Pay extra attention to readability. Ensure the text is always readable with sufficient contrast against the background. Choose fonts and colors that enhance legibility.

**Design and Functionality:**
* **Component-Based Design**: Do not just dump text into blocks. Semanticize the content into distinct UI components.
* **Layout Dynamics**: Break the grid. Avoid strict, identical grid columns. Use asymmetrical layouts, Bento grids, or responsive flexbox layouts where some elements span full width to create visual interest and emphasize key content.
* **Tailwind Configuration**: Extend the Tailwind configuration within a ` + "`<script>`" + ` block to define custom font families and color palettes that match the theme.
* Thoroughly analyze the content to determine the most compelling layout or visualization. For example, if data has multiple sections, consider cards, carousels, or tabbed views.
* If requirements are underspecified, make reasonable assumptions. Your goal is to deliver a polished product with no placeholder content.
* The output must be a complete and valid HTML document with no placeholder content.

**Libraries:**
* Use Tailwind for CSS via CDN: ` + "`<script src=\"https://cdn.tailwindcss.com\"></script>`" + `
* Google Fonts are allowed for typography imports.

**Constraints:**
* **Media Restriction:** ONLY use media URLs that are explicitly present in the input data. Do NOT generate or hallucinate any media URLs.
* **Render All Media:** You MUST render ALL media (images, videos, audio) that are present in the data. Every provided media URL must appear in the final HTML output.
* **Navigation Restriction:** Do NOT generate unneeded fake links or buttons to sub-pages (e.g. "About", "Contact", "Learn More") unless the data explicitly calls for them.
* **Footer Restriction:** NEVER generate any footer content, including legal footers like "All rights reserved" or "Copyright".
* Output ONLY the HTML document, no explanation or markdown fences.`

// manualLayoutSystemPrompt instructs the LLM to generate HTML from the user's layout instructions.
var manualLayoutSystemPrompt = `You are an AI Web Developer. The user will provide detailed layout instructions describing page structure, style design language, and component guidelines, along with data from upstream workflow nodes.

Your task is to generate a single, self-contained HTML document for rendering in an iframe that follows the user's layout instructions precisely.

**Core Rules:**
* Follow the user's layout organization, style design language, and component guidelines exactly.
* The output must be a complete and valid HTML document with no placeholder content.
* **Prioritize Visual Excellence**: Even within the user's style constraints, aim for premium, polished results with attention to typography, spacing, and responsive design.
* **Readability**: Ensure the text is always readable with sufficient contrast against the background.

**Libraries:**
* Use Tailwind for CSS via CDN: ` + "`<script src=\"https://cdn.tailwindcss.com\"></script>`" + `
* Google Fonts are allowed for typography imports.
* **Tailwind Configuration**: Extend the Tailwind configuration within a ` + "`<script>`" + ` block to define custom font families and color palettes that match the user's specified theme.

**Constraints:**
* **Media Restriction:** ONLY use media URLs that are explicitly present in the input data. Do NOT generate or hallucinate any media URLs.
* **Render All Media:** You MUST render ALL media (images, videos, audio) that are present in the data. Every provided media URL must appear in the final HTML output.
* **Navigation Restriction:** Do NOT generate unneeded fake links or buttons unless the user explicitly requests them.
* **Footer Restriction:** NEVER generate any footer content unless the user explicitly requests it.
* Output ONLY the HTML document, no explanation or markdown fences.`

func buildOutputAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM) (agent.Agent, error) {
	nodeID := nd.ID
	displayMode, _ := nd.Config["display_mode"].(string)
	layoutModel, _ := nd.Config["layout_model"].(string)
	layoutPrompt, _ := nd.Config["layout_prompt"].(string)

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Output node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				var keys []string
				for k := range state.All() {
					if !strings.HasPrefix(k, "__") {
						keys = append(keys, k)
					}
				}
				sort.Strings(keys)

				var parts []string
				for _, k := range keys {
					if k == nodeID {
						continue
					}
					v, err := state.Get(k)
					if err != nil || v == nil {
						continue
					}
					parts = append(parts, fmt.Sprintf("%v", v))
				}

				result := strings.Join(parts, "\n\n")

				// Manual layout with user-provided prompt: resolve templates and call LLM
				if displayMode != "auto-layout" && layoutPrompt != "" && llms != nil {
					resolvedPrompt := resolveTemplateFromState(layoutPrompt, state)
					if html, err := generateLayout(ctx, resolvedPrompt, manualLayoutSystemPrompt, layoutModel, llms); err == nil && html != "" {
						result = html
					}
					// On error, fall back to plain text
				}

				// Auto-layout: use LLM to generate a styled HTML page
				if displayMode == "auto-layout" && llms != nil {
					userContent := fmt.Sprintf("Create a styled HTML page presenting the following content:\n\n%s", result)
					if html, err := generateLayout(ctx, userContent, autoLayoutSystemPrompt, layoutModel, llms); err == nil && html != "" {
						result = html
					}
					// On error, fall back to plain text
				}

				_ = state.Set(nodeID, result)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(result)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = result
				yield(event, nil)
			}
		},
	})
}

// generateLayout calls an LLM with the given user content and system prompt to produce
// a styled HTML page. Used by both auto-layout and manual-layout modes.
func generateLayout(ctx agent.InvocationContext, userContent string, systemPrompt string, layoutModel string, llms map[string]adkmodel.LLM) (string, error) {
	llm, modelName := resolveLLM(layoutModel, llms)
	if llm == nil {
		return "", fmt.Errorf("no LLM available for layout generation")
	}

	req := &adkmodel.LLMRequest{
		Model: modelName,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userContent, genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range llm.GenerateContent(ctx, req, false) {
		if err != nil {
			return "", fmt.Errorf("layout LLM call: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	// Strip markdown code fences if present
	text := strings.TrimSpace(llmutil.ExtractText(resp))
	text = strings.TrimPrefix(text, "```html")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	return text, nil
}

// buildToolAgent creates a custom Agent that resolves templates in config["input"],
// calls the tool registry, and stores the result under the node ID.
func buildToolAgent(nd *upal.NodeDefinition, toolReg *tools.Registry) (agent.Agent, error) {
	nodeID := nd.ID
	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("Tool node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				toolName, _ := nd.Config["tool"].(string)
				inputTpl, _ := nd.Config["input"].(string)
				resolvedInput := resolveTemplateFromState(inputTpl, state)

				var result any
				var execErr error
				if toolReg != nil && toolName != "" {
					result, execErr = toolReg.Execute(ctx, toolName, resolvedInput)
				}

				if execErr != nil {
					yield(nil, fmt.Errorf("tool %q execution failed: %w", toolName, execErr))
					return
				}

				resultStr := fmt.Sprintf("%v", result)
				_ = state.Set(nodeID, resultStr)

				event := session.NewEvent(ctx.InvocationID())
				event.Author = nodeID
				event.Branch = ctx.Branch()
				event.LLMResponse = adkmodel.LLMResponse{
					Content: &genai.Content{
						Role:  "model",
						Parts: []*genai.Part{genai.NewPartFromText(resultStr)},
					},
					TurnComplete: true,
				}
				event.Actions.StateDelta[nodeID] = resultStr
				yield(event, nil)
			}
		},
	})
}

// buildLLMAgent creates a custom Agent that resolves {{node_id}} template
// references in the prompt config from session state, then calls the LLM.
// This replaces the previous llmagent.New() approach which could not access
// session state for template resolution — the ADK runner sends "run" as user
// content, so agents built with llmagent.New() never see the actual prompt.
func buildLLMAgent(nd *upal.NodeDefinition, llms map[string]adkmodel.LLM, toolReg *tools.Registry) (agent.Agent, error) {
	nodeID := nd.ID

	modelID, _ := nd.Config["model"].(string)
	systemPrompt, _ := nd.Config["system_prompt"].(string)
	promptTpl, _ := nd.Config["prompt"].(string)
	maxTurns := 1
	if mt, ok := nd.Config["max_turns"].(float64); ok && mt > 0 {
		maxTurns = int(mt)
	}

	// Resolve the LLM from "provider/model" format.
	llm, modelName := resolveLLM(modelID, llms)
	if llm == nil {
		return nil, fmt.Errorf("no LLM found for model %q in node %q", modelID, nodeID)
	}

	// Collect tool definitions for tool-use loop.
	// We store both genai function declarations (for the LLM request) and
	// a name→Tool map (for executing tool calls).
	var funcDecls []*genai.FunctionDeclaration
	upalTools := make(map[string]tools.Tool)
	if toolNames, ok := nd.Config["tools"].([]any); ok && toolReg != nil {
		for _, tn := range toolNames {
			name, ok := tn.(string)
			if !ok {
				continue
			}
			t, found := toolReg.Get(name)
			if found {
				upalTools[name] = t
				funcDecls = append(funcDecls, &genai.FunctionDeclaration{
					Name:        t.Name(),
					Description: t.Description(),
					Parameters:  toGenaiSchema(t.InputSchema()),
				})
			}
		}
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("LLM agent node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				// Resolve {{node_id}} templates in the prompt from session state.
				resolvedPrompt := resolveTemplateFromState(promptTpl, state)

				// Build LLM request with system instruction + resolved user prompt.
				contents := []*genai.Content{
					genai.NewContentFromText(resolvedPrompt, genai.RoleUser),
				}

				genCfg := &genai.GenerateContentConfig{
					SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
				}
				if len(funcDecls) > 0 {
					genCfg.Tools = []*genai.Tool{{FunctionDeclarations: funcDecls}}
				}

				// Tool-use agentic loop: call LLM, execute tool calls, repeat.
				for turn := 0; turn < maxTurns; turn++ {
					req := &adkmodel.LLMRequest{
						Model:    modelName,
						Config:   genCfg,
						Contents: contents,
					}

					var resp *adkmodel.LLMResponse
					for r, err := range llm.GenerateContent(ctx, req, false) {
						if err != nil {
							yield(nil, fmt.Errorf("LLM call failed for node %q: %w", nodeID, err))
							return
						}
						resp = r
					}

					if resp == nil || resp.Content == nil {
						yield(nil, fmt.Errorf("empty LLM response for node %q", nodeID))
						return
					}

					// Check for tool calls in the response.
					var toolCalls []*genai.FunctionCall
					for _, p := range resp.Content.Parts {
						if p.FunctionCall != nil {
							toolCalls = append(toolCalls, p.FunctionCall)
						}
					}

					// No tool calls — extract text result and finish.
					if len(toolCalls) == 0 {
						result := strings.TrimSpace(llmutil.ExtractText(resp))
						_ = state.Set(nodeID, result)

						event := session.NewEvent(ctx.InvocationID())
						event.Author = nodeID
						event.Branch = ctx.Branch()
						event.LLMResponse = adkmodel.LLMResponse{
							Content: &genai.Content{
								Role:  "model",
								Parts: []*genai.Part{genai.NewPartFromText(result)},
							},
							TurnComplete: true,
						}
						event.Actions.StateDelta[nodeID] = result
						yield(event, nil)
						return
					}

					// Execute tool calls and append results to conversation.
					contents = append(contents, resp.Content)
					var toolResults []*genai.Part
					for _, fc := range toolCalls {
						var output map[string]any
						if t, ok := upalTools[fc.Name]; ok {
							result, err := t.Execute(ctx, fc.Args)
							if err != nil {
								output = map[string]any{"error": err.Error()}
							} else {
								// Coerce result to map[string]any for FunctionResponse.
								if m, ok := result.(map[string]any); ok {
									output = m
								} else {
									output = map[string]any{"result": fmt.Sprintf("%v", result)}
								}
							}
						} else {
							output = map[string]any{"error": fmt.Sprintf("unknown tool %q", fc.Name)}
						}
						toolResults = append(toolResults, &genai.Part{
							FunctionResponse: &genai.FunctionResponse{
								Name:     fc.Name,
								Response: output,
							},
						})
					}
					contents = append(contents, &genai.Content{
						Role:  genai.RoleUser,
						Parts: toolResults,
					})
				}

				// Exhausted max_turns — yield error.
				yield(nil, fmt.Errorf("node %q exceeded max_turns (%d)", nodeID, maxTurns))
			}
		},
	})
}

// buildRemoteAgent creates a remote A2A agent using remoteagent.NewA2A().
func buildRemoteAgent(nd *upal.NodeDefinition) (agent.Agent, error) {
	nodeID := nd.ID
	endpointURL, _ := nd.Config["endpoint_url"].(string)

	if endpointURL == "" {
		return nil, fmt.Errorf("external node %q missing endpoint_url", nodeID)
	}

	return remoteagent.NewA2A(remoteagent.A2AConfig{
		Name:            nodeID,
		Description:     fmt.Sprintf("Remote A2A agent node %s", nodeID),
		AgentCardSource: endpointURL,
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
// and the bare model name. If the provider is found, the LLM is wrapped with
// namedLLM so that Name() returns the model name. Falls back to the first
// available LLM if the specified provider is not found. Returns (nil, "") if
// no LLMs are available.
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
