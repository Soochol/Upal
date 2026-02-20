// Package model provides LLM interface implementations for various providers.
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

var _ adkmodel.LLM = (*OpenAILLM)(nil)

const openaiDefaultBaseURL = "https://api.openai.com/v1"

// OpenAIOption configures an OpenAILLM instance.
type OpenAIOption func(*OpenAILLM)

// WithOpenAIBaseURL sets a custom base URL for the API endpoint.
// This is useful for OpenAI-compatible APIs like Ollama and LM Studio.
func WithOpenAIBaseURL(url string) OpenAIOption {
	return func(o *OpenAILLM) {
		o.baseURL = url
	}
}

// WithOpenAIName sets a custom name for the LLM instance.
func WithOpenAIName(name string) OpenAIOption {
	return func(o *OpenAILLM) {
		o.name = name
	}
}

// OpenAILLM implements the ADK model.LLM interface for the OpenAI Chat Completions API.
// It also works with OpenAI-compatible APIs such as Ollama and LM Studio.
type OpenAILLM struct {
	apiKey  string
	baseURL string
	name    string
	client  *http.Client
}

// NewOpenAILLM creates a new OpenAI LLM adapter.
func NewOpenAILLM(apiKey string, opts ...OpenAIOption) *OpenAILLM {
	llm := &OpenAILLM{
		apiKey:  apiKey,
		baseURL: openaiDefaultBaseURL,
		name:    "openai",
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(llm)
	}
	return llm
}

// Name returns the configured name of this LLM (default "openai").
func (o *OpenAILLM) Name() string {
	return o.name
}

// GenerateContent sends a chat completion request to the OpenAI API and returns
// an iterator that yields exactly one LLMResponse for non-streaming requests.
func (o *OpenAILLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		// Build the OpenAI request body.
		body, err := o.buildRequestBody(req)
		if err != nil {
			yield(nil, fmt.Errorf("openai: failed to build request: %w", err))
			return
		}

		encoded, err := json.Marshal(body)
		if err != nil {
			yield(nil, fmt.Errorf("openai: failed to marshal request: %w", err))
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(encoded))
		if err != nil {
			yield(nil, fmt.Errorf("openai: failed to create HTTP request: %w", err))
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		if o.apiKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)
		}

		httpResp, err := o.client.Do(httpReq)
		if err != nil {
			yield(nil, fmt.Errorf("openai: HTTP request failed: %w", err))
			return
		}
		defer httpResp.Body.Close()

		respBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			yield(nil, fmt.Errorf("openai: failed to read response body: %w", err))
			return
		}

		if httpResp.StatusCode != http.StatusOK {
			yield(nil, fmt.Errorf("openai: API returned status %d: %s", httpResp.StatusCode, string(respBody)))
			return
		}

		var apiResp openaiChatResponse
		if err := json.Unmarshal(respBody, &apiResp); err != nil {
			yield(nil, fmt.Errorf("openai: failed to unmarshal response: %w", err))
			return
		}

		llmResp, err := o.convertResponse(&apiResp)
		if err != nil {
			yield(nil, fmt.Errorf("openai: failed to convert response: %w", err))
			return
		}

		yield(llmResp, nil)
	}
}

// buildRequestBody converts an LLMRequest into an OpenAI chat completions request body.
func (o *OpenAILLM) buildRequestBody(req *adkmodel.LLMRequest) (map[string]any, error) {
	body := map[string]any{
		"model":  req.Model,
		"stream": false,
	}

	var messages []map[string]any

	// Add system instruction as a system message if present.
	if req.Config != nil && req.Config.SystemInstruction != nil {
		text := extractText(req.Config.SystemInstruction)
		if text != "" {
			messages = append(messages, map[string]any{
				"role":    "system",
				"content": text,
			})
		}
	}

	// Convert each Content to OpenAI messages.
	for _, content := range req.Contents {
		msgs, err := o.convertContent(content)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msgs...)
	}

	body["messages"] = messages

	// Convert tools if present.
	if req.Config != nil && len(req.Config.Tools) > 0 {
		tools := o.convertTools(req.Config.Tools)
		if len(tools) > 0 {
			body["tools"] = tools
		}
	}

	// Pass through optional generation parameters.
	if req.Config != nil {
		if req.Config.Temperature != nil {
			body["temperature"] = *req.Config.Temperature
		}
		if req.Config.TopP != nil {
			body["top_p"] = *req.Config.TopP
		}
		if req.Config.MaxOutputTokens > 0 {
			body["max_tokens"] = req.Config.MaxOutputTokens
		}
		if len(req.Config.StopSequences) > 0 {
			body["stop"] = req.Config.StopSequences
		}
		if req.Config.FrequencyPenalty != nil {
			body["frequency_penalty"] = *req.Config.FrequencyPenalty
		}
		if req.Config.PresencePenalty != nil {
			body["presence_penalty"] = *req.Config.PresencePenalty
		}
	}

	return body, nil
}

// convertContent converts a single genai.Content into one or more OpenAI message objects.
func (o *OpenAILLM) convertContent(content *genai.Content) ([]map[string]any, error) {
	var messages []map[string]any

	// Determine the OpenAI role.
	role := openaiRole(content.Role)

	// Check if any parts are function calls (assistant with tool_calls).
	var toolCalls []map[string]any
	var textParts []string
	var functionResponses []*genai.FunctionResponse

	for _, part := range content.Parts {
		switch {
		case part.FunctionCall != nil:
			argsJSON, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function call args: %w", err)
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   part.FunctionCall.ID,
				"type": "function",
				"function": map[string]any{
					"name":      part.FunctionCall.Name,
					"arguments": string(argsJSON),
				},
			})
		case part.FunctionResponse != nil:
			functionResponses = append(functionResponses, part.FunctionResponse)
		case part.Text != "":
			textParts = append(textParts, part.Text)
		}
	}

	// If there are tool calls, emit an assistant message with tool_calls.
	if len(toolCalls) > 0 {
		msg := map[string]any{
			"role":       "assistant",
			"tool_calls": toolCalls,
		}
		if len(textParts) > 0 {
			msg["content"] = textParts[0]
		}
		messages = append(messages, msg)
	} else if len(functionResponses) > 0 {
		// Function responses become tool role messages.
		for _, fr := range functionResponses {
			respJSON, err := json.Marshal(fr.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function response: %w", err)
			}
			messages = append(messages, map[string]any{
				"role":         "tool",
				"tool_call_id": fr.ID,
				"content":      string(respJSON),
			})
		}
	} else if len(textParts) > 0 {
		// Plain text message.
		combined := textParts[0]
		for i := 1; i < len(textParts); i++ {
			combined += "\n" + textParts[i]
		}
		messages = append(messages, map[string]any{
			"role":    role,
			"content": combined,
		})
	}

	return messages, nil
}

// convertTools converts genai Tool definitions to OpenAI tool format.
func (o *OpenAILLM) convertTools(tools []*genai.Tool) []map[string]any {
	var result []map[string]any
	for _, tool := range tools {
		// Native server-managed tools.
		if tool.GoogleSearch != nil {
			result = append(result, map[string]any{
				"type": "web_search_preview",
			})
		}
		// Custom function declarations.
		for _, fd := range tool.FunctionDeclarations {
			fn := map[string]any{
				"name": fd.Name,
			}
			if fd.Description != "" {
				fn["description"] = fd.Description
			}
			if fd.Parameters != nil {
				fn["parameters"] = convertSchema(fd.Parameters)
			}
			result = append(result, map[string]any{
				"type":     "function",
				"function": fn,
			})
		}
	}
	return result
}

// convertSchema converts a genai.Schema to a JSON Schema-compatible map for OpenAI.
func convertSchema(s *genai.Schema) map[string]any {
	schema := map[string]any{}
	if s.Type != "" {
		schema["type"] = openaiSchemaType(string(s.Type))
	}
	if s.Description != "" {
		schema["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		schema["enum"] = s.Enum
	}
	if s.Items != nil {
		schema["items"] = convertSchema(s.Items)
	}
	if len(s.Properties) > 0 {
		props := map[string]any{}
		for name, prop := range s.Properties {
			props[name] = convertSchema(prop)
		}
		schema["properties"] = props
	}
	if len(s.Required) > 0 {
		schema["required"] = s.Required
	}
	return schema
}

// openaiSchemaType converts genai schema type strings (e.g., "OBJECT") to
// JSON Schema type strings (e.g., "object").
func openaiSchemaType(t string) string {
	switch t {
	case "STRING":
		return "string"
	case "NUMBER", "FLOAT":
		return "number"
	case "INTEGER", "INT":
		return "integer"
	case "BOOLEAN", "BOOL":
		return "boolean"
	case "ARRAY":
		return "array"
	case "OBJECT":
		return "object"
	default:
		return t
	}
}

// convertResponse converts an OpenAI chat response to an ADK LLMResponse.
func (o *OpenAILLM) convertResponse(resp *openaiChatResponse) (*adkmodel.LLMResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	content := &genai.Content{
		Role: genai.RoleModel,
	}

	// Convert tool calls to FunctionCall parts.
	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]any
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tool call arguments: %w", err)
				}
			}
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: args,
				},
			})
		}
	} else if choice.Message.Content != "" {
		content.Parts = append(content.Parts, genai.NewPartFromText(choice.Message.Content))
	}

	return &adkmodel.LLMResponse{
		Content:      content,
		TurnComplete: true,
	}, nil
}

// extractText concatenates all text parts from a Content.
func extractText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var text string
	for i, part := range content.Parts {
		if part.Text != "" {
			if i > 0 && text != "" {
				text += "\n"
			}
			text += part.Text
		}
	}
	return text
}

// openaiRole converts a genai role string to an OpenAI role string.
func openaiRole(role string) string {
	switch role {
	case genai.RoleModel:
		return "assistant"
	case genai.RoleUser:
		return "user"
	default:
		return role
	}
}

// --- OpenAI API types (self-contained, not shared) ---

type openaiChatResponse struct {
	Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
