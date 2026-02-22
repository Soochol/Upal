package agents

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// LLMNodeBuilder creates agents that call an LLM with optional tool-use loop.
type LLMNodeBuilder struct{}

func (b *LLMNodeBuilder) NodeType() upal.NodeType { return upal.NodeTypeAgent }

func (b *LLMNodeBuilder) Build(nd *upal.NodeDefinition, deps BuildDeps) (agent.Agent, error) {
	nodeID := nd.ID
	outputDir := deps.OutputDir

	modelID, _ := nd.Config["model"].(string)
	systemPrompt, _ := nd.Config["system_prompt"].(string)
	promptTpl, _ := nd.Config["prompt"].(string)
	outputFmt, _ := nd.Config["output"].(string)

	var temperature *float32
	if v, ok := nd.Config["temperature"].(float64); ok {
		t := float32(v)
		temperature = &t
	}
	var maxTokens int32
	if v, ok := nd.Config["max_tokens"].(float64); ok {
		maxTokens = int32(v)
	}
	var topP *float32
	if v, ok := nd.Config["top_p"].(float64); ok {
		t := float32(v)
		topP = &t
	}

	var imageParams *upalmodel.ImageParams
	if ratio, ok := nd.Config["aspect_ratio"].(string); ok {
		imageParams = &upalmodel.ImageParams{}
		imageParams.Width, imageParams.Height = aspectRatioToSize(ratio)
	}
	if v, ok := nd.Config["steps"].(float64); ok {
		if imageParams == nil {
			imageParams = &upalmodel.ImageParams{}
		}
		imageParams.Steps = int(v)
	}

	if outputFmt != "" {
		systemPrompt += "\n\n" + outputFmt
	}

	llm, modelName := resolveLLM(modelID, deps.LLMs)
	if llm == nil {
		return nil, fmt.Errorf("no LLM found for model %q in node %q", modelID, nodeID)
	}

	var funcDecls []*genai.FunctionDeclaration
	var nativeTools []*genai.Tool
	upalTools := make(map[string]tools.Tool)
	if toolNames, ok := nd.Config["tools"].([]any); ok {
		for _, tn := range toolNames {
			name, ok := tn.(string)
			if !ok {
				continue
			}
			if deps.ToolReg != nil && deps.ToolReg.IsNative(name) {
				switch name {
				case tools.WebSearch.Name():
					nativeTools = append(nativeTools, &genai.Tool{
						GoogleSearch: &genai.GoogleSearch{},
					})
				}
				continue
			}
			if name == tools.WebSearch.Name() {
				nativeTools = append(nativeTools, &genai.Tool{
					GoogleSearch: &genai.GoogleSearch{},
				})
				continue
			}
			if deps.ToolReg == nil {
				return nil, fmt.Errorf("node %q references tool %q but no tool registry is configured", nd.ID, name)
			}
			t, found := deps.ToolReg.Get(name)
			if !found {
				return nil, fmt.Errorf("node %q references unknown tool %q", nd.ID, name)
			}
			upalTools[name] = t
			funcDecls = append(funcDecls, &genai.FunctionDeclaration{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  toGenaiSchema(t.InputSchema()),
			})
		}
	}

	maxTurns := 1
	if len(funcDecls) > 0 {
		maxTurns = 10
	}

	return agent.New(agent.Config{
		Name:        nodeID,
		Description: fmt.Sprintf("LLM agent node %s", nodeID),
		Run: func(ctx agent.InvocationContext) iter.Seq2[*session.Event, error] {
			return func(yield func(*session.Event, error) bool) {
				state := ctx.Session().State()

				resolvedPrompt := resolveTemplateFromState(promptTpl, state)

				contents := []*genai.Content{
					{Role: genai.RoleUser, Parts: buildPromptParts(resolvedPrompt)},
				}

				genCfg := &genai.GenerateContentConfig{
					SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
				}
				if temperature != nil {
					genCfg.Temperature = temperature
				}
				if maxTokens > 0 {
					genCfg.MaxOutputTokens = maxTokens
				}
				if topP != nil {
					genCfg.TopP = topP
				}
				var allTools []*genai.Tool
				allTools = append(allTools, nativeTools...)
				if len(funcDecls) > 0 {
					allTools = append(allTools, &genai.Tool{FunctionDeclarations: funcDecls})
				}
				if len(allTools) > 0 {
					genCfg.Tools = allTools
				}

				var llmCtx context.Context = ctx
				if imageParams != nil {
					llmCtx = upalmodel.WithImageParams(llmCtx, *imageParams)
				}
				if nodeLogFn := nodeLogFuncFromContext(ctx); nodeLogFn != nil {
					llmCtx = upalmodel.WithLogFunc(llmCtx, upalmodel.LogFunc(func(msg string) {
						nodeLogFn(nodeID, msg)
					}))
				}

				for turn := 0; turn < maxTurns; turn++ {
					req := &adkmodel.LLMRequest{
						Model:    modelName,
						Config:   genCfg,
						Contents: contents,
					}

					var resp *adkmodel.LLMResponse
					for r, err := range llm.GenerateContent(llmCtx, req, false) {
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

					var toolCalls []*genai.FunctionCall
					for _, p := range resp.Content.Parts {
						if p.FunctionCall != nil {
							toolCalls = append(toolCalls, p.FunctionCall)
						}
					}

					if len(toolCalls) == 0 {
						result := strings.TrimSpace(llmutil.ExtractContentSavingAudio(resp, outputDir))
						_ = state.Set(nodeID, result)

						event := session.NewEvent(ctx.InvocationID())
						event.Author = nodeID
						event.Branch = ctx.Branch()
						event.LLMResponse = adkmodel.LLMResponse{
							Content:       resp.Content,
							TurnComplete:  true,
							FinishReason:  resp.FinishReason,
							UsageMetadata: resp.UsageMetadata,
						}
						event.Actions.StateDelta[nodeID] = result
						yield(event, nil)
						return
					}

					toolCallEvent := session.NewEvent(ctx.InvocationID())
					toolCallEvent.Author = nodeID
					toolCallEvent.Branch = ctx.Branch()
					toolCallEvent.LLMResponse = adkmodel.LLMResponse{Content: resp.Content}
					if !yield(toolCallEvent, nil) {
						return
					}

					contents = append(contents, resp.Content)
					toolRespContent := executeToolCalls(ctx, toolCalls, upalTools)
					contents = append(contents, toolRespContent)

					toolRespEvent := session.NewEvent(ctx.InvocationID())
					toolRespEvent.Author = nodeID
					toolRespEvent.Branch = ctx.Branch()
					toolRespEvent.LLMResponse = adkmodel.LLMResponse{Content: toolRespContent}
					if !yield(toolRespEvent, nil) {
						return
					}
				}

				yield(nil, fmt.Errorf("node %q exceeded max_turns (%d)", nodeID, maxTurns))
			}
		},
	})
}

// executeToolCalls executes a list of function calls against the tool registry
// and returns a Content with FunctionResponse parts for feeding back to the LLM.
func executeToolCalls(ctx agent.InvocationContext, calls []*genai.FunctionCall, upalTools map[string]tools.Tool) *genai.Content {
	var toolResults []*genai.Part
	for _, fc := range calls {
		output := executeSingleTool(ctx, fc, upalTools)
		toolResults = append(toolResults, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     fc.Name,
				Response: output,
			},
		})
	}
	return &genai.Content{
		Role:  genai.RoleUser,
		Parts: toolResults,
	}
}

// executeSingleTool runs a single tool call with panic recovery.
func executeSingleTool(ctx agent.InvocationContext, fc *genai.FunctionCall, upalTools map[string]tools.Tool) (output map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			output = map[string]any{"error": fmt.Sprintf("tool %q panicked: %v", fc.Name, r)}
		}
	}()

	t, ok := upalTools[fc.Name]
	if !ok {
		return map[string]any{"error": fmt.Sprintf("unknown tool %q", fc.Name)}
	}
	result, err := t.Execute(ctx, fc.Args)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return map[string]any{"result": fmt.Sprintf("%v", result)}
}
