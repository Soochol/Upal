// Package model implements ADK model.LLM interfaces for various LLM providers.
package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"
)

// Compile-time interface compliance check.
var _ adkmodel.LLM = (*AnthropicLLM)(nil)

const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicVersion        = "2023-06-01"
	defaultMaxTokens        = 4096
)

// AnthropicOption configures an AnthropicLLM.
type AnthropicOption func(*AnthropicLLM)

// WithAnthropicBaseURL sets the base URL for the Anthropic API.
// Useful for testing with httptest.
func WithAnthropicBaseURL(url string) AnthropicOption {
	return func(a *AnthropicLLM) {
		a.baseURL = url
	}
}

// AnthropicLLM implements the ADK model.LLM interface for the Anthropic Messages API.
type AnthropicLLM struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewAnthropicLLM creates a new AnthropicLLM with the given API key and options.
func NewAnthropicLLM(apiKey string, opts ...AnthropicOption) *AnthropicLLM {
	a := &AnthropicLLM{
		apiKey:  apiKey,
		baseURL: defaultAnthropicBaseURL,
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns "anthropic".
func (a *AnthropicLLM) Name() string {
	return "anthropic"
}

// GenerateContent converts genai.Content to the Anthropic API format, calls the API,
// and converts the response back to genai.Content. The stream parameter is accepted
// for interface compliance but streaming is not yet implemented.
func (a *AnthropicLLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		resp, err := a.generate(ctx, req)
		yield(resp, err)
	}
}

// generate performs a synchronous call to the Anthropic Messages API.
func (a *AnthropicLLM) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	body := a.buildRequestBody(req)

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/v1/messages", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	if beta := anthropicBetaFeatures(req); beta != "" {
		httpReq.Header.Set("anthropic-beta", beta)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return a.convertResponse(&apiResp), nil
}

// buildRequestBody converts an LLMRequest into the Anthropic API request body.
func (a *AnthropicLLM) buildRequestBody(req *adkmodel.LLMRequest) map[string]any {
	var systemPrompt string
	var messages []map[string]any

	// Extract system instruction from config
	if req.Config != nil && req.Config.SystemInstruction != nil {
		for _, part := range req.Config.SystemInstruction.Parts {
			if part.Text != "" {
				systemPrompt = part.Text
				break
			}
		}
	}

	// Convert contents to Anthropic messages
	for _, content := range req.Contents {
		switch content.Role {
		case "model":
			// Convert model role to assistant
			messages = append(messages, a.convertModelContent(content))
		case "user":
			messages = append(messages, a.convertUserContent(content))
		}
	}

	maxTokens := int32(defaultMaxTokens)
	if req.Config != nil && req.Config.MaxOutputTokens > 0 {
		maxTokens = req.Config.MaxOutputTokens
	}

	body := map[string]any{
		"model":      req.Model,
		"messages":   messages,
		"max_tokens": maxTokens,
	}

	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	if req.Config != nil && req.Config.Temperature != nil {
		body["temperature"] = *req.Config.Temperature
	}

	// Convert tools
	if req.Config != nil && len(req.Config.Tools) > 0 {
		var tools []map[string]any
		for _, tool := range req.Config.Tools {
			// Native server-managed tools.
			if tool.GoogleSearch != nil {
				tools = append(tools, map[string]any{
					"type": "web_search_20250305",
					"name": "web_search",
				})
			}
			// Custom function declarations.
			for _, fd := range tool.FunctionDeclarations {
				t := map[string]any{
					"name":        fd.Name,
					"description": fd.Description,
				}
				// Prefer ParametersJsonSchema if set, fall back to Parameters
				if fd.ParametersJsonSchema != nil {
					t["input_schema"] = fd.ParametersJsonSchema
				} else if fd.Parameters != nil {
					t["input_schema"] = fd.Parameters
				}
				tools = append(tools, t)
			}
		}
		if len(tools) > 0 {
			body["tools"] = tools
		}
	}

	return body
}

// convertModelContent converts a genai.Content with role "model" to an Anthropic assistant message.
func (a *AnthropicLLM) convertModelContent(content *genai.Content) map[string]any {
	var blocks []map[string]any

	for _, part := range content.Parts {
		if part.Text != "" {
			blocks = append(blocks, map[string]any{
				"type": "text",
				"text": part.Text,
			})
		}
		if part.FunctionCall != nil {
			fc := part.FunctionCall
			block := map[string]any{
				"type":  "tool_use",
				"id":    fc.ID,
				"name":  fc.Name,
				"input": fc.Args,
			}
			blocks = append(blocks, block)
		}
	}

	// If there's only one text block, use the simple string format
	if len(blocks) == 1 && blocks[0]["type"] == "text" {
		return map[string]any{
			"role":    "assistant",
			"content": blocks[0]["text"],
		}
	}

	return map[string]any{
		"role":    "assistant",
		"content": blocks,
	}
}

// convertUserContent converts a genai.Content with role "user" to an Anthropic user message.
// FunctionResponse parts are converted to tool_result content blocks.
func (a *AnthropicLLM) convertUserContent(content *genai.Content) map[string]any {
	var blocks []map[string]any
	hasStructuredContent := false

	for _, part := range content.Parts {
		if part.FunctionResponse != nil {
			hasStructuredContent = true
			fr := part.FunctionResponse
			block := map[string]any{
				"type":        "tool_result",
				"tool_use_id": fr.ID,
			}
			// Serialize the response as content
			if fr.Response != nil {
				contentJSON, err := json.Marshal(fr.Response)
				if err == nil {
					block["content"] = string(contentJSON)
				}
			}
			blocks = append(blocks, block)
		} else if part.Text != "" {
			hasStructuredContent = true
			blocks = append(blocks, map[string]any{
				"type": "text",
				"text": part.Text,
			})
		}
	}

	// If all parts are simple text and there's only one, use the simple string format
	if !hasStructuredContent || (len(blocks) == 1 && blocks[0]["type"] == "text") {
		for _, part := range content.Parts {
			if part.Text != "" {
				return map[string]any{
					"role":    "user",
					"content": part.Text,
				}
			}
		}
	}

	return map[string]any{
		"role":    "user",
		"content": blocks,
	}
}

// convertResponse converts an Anthropic API response to an ADK LLMResponse.
func (a *AnthropicLLM) convertResponse(apiResp *anthropicAPIResponse) *adkmodel.LLMResponse {
	var parts []*genai.Part

	for _, block := range apiResp.Content {
		switch block.Type {
		case "text":
			parts = append(parts, genai.NewPartFromText(block.Text))
		case "tool_use":
			args, ok := block.Input.(map[string]any)
			if !ok {
				args = make(map[string]any)
			}
			parts = append(parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   block.ID,
					Name: block.Name,
					Args: args,
				},
			})
		}
	}

	llmResp := &adkmodel.LLMResponse{
		Content: &genai.Content{
			Role:  "model",
			Parts: parts,
		},
		TurnComplete: true,
	}

	// Map Anthropic stop_reason to genai.FinishReason
	switch apiResp.StopReason {
	case "end_turn":
		llmResp.FinishReason = genai.FinishReasonStop
	case "max_tokens":
		llmResp.FinishReason = genai.FinishReasonMaxTokens
	case "tool_use":
		llmResp.FinishReason = genai.FinishReasonStop
	}

	return llmResp
}

// Anthropic API response types

type anthropicAPIResponse struct {
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                 `json:"stop_reason"`
}

type anthropicContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// anthropicBetaFeatures returns the anthropic-beta header value for native tools.
func anthropicBetaFeatures(req *adkmodel.LLMRequest) string {
	if req.Config == nil {
		return ""
	}
	for _, tool := range req.Config.Tools {
		if tool.GoogleSearch != nil {
			return "web-search-2025-03-05"
		}
	}
	return ""
}
