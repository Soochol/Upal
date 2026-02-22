package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"
)

func TestAnthropicLLM_Name(t *testing.T) {
	llm := NewAnthropicLLM("test-key")
	if llm.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", llm.Name(), "anthropic")
	}
}

func TestAnthropicLLM_GenerateContent(t *testing.T) {
	// Mock Anthropic API server
	var receivedReq map[string]any
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedReq); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Hello from Claude!"},
			},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-api-key-123", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What is Go?")},
			},
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText("You are a helpful assistant.", "system"),
		},
	}

	var responses []*adkmodel.LLMResponse
	var errs []error

	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		responses = append(responses, resp)
	}

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// Verify API key header
	if got := receivedHeaders.Get("x-api-key"); got != "test-api-key-123" {
		t.Errorf("x-api-key header = %q, want %q", got, "test-api-key-123")
	}

	// Verify anthropic-version header
	if got := receivedHeaders.Get("anthropic-version"); got != "2023-06-01" {
		t.Errorf("anthropic-version header = %q, want %q", got, "2023-06-01")
	}

	// Verify content-type header
	if got := receivedHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type header = %q, want %q", got, "application/json")
	}

	// Verify the request body
	if model, ok := receivedReq["model"].(string); !ok || model != "claude-sonnet-4-20250514" {
		t.Errorf("request model = %v, want %q", receivedReq["model"], "claude-sonnet-4-20250514")
	}
	if system, ok := receivedReq["system"].(string); !ok || system != "You are a helpful assistant." {
		t.Errorf("request system = %v, want %q", receivedReq["system"], "You are a helpful assistant.")
	}

	// Verify we got exactly one response
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}

	resp := responses[0]
	if resp.Content == nil {
		t.Fatal("response Content is nil")
	}
	if resp.Content.Role != "model" {
		t.Errorf("response role = %q, want %q", resp.Content.Role, "model")
	}
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(resp.Content.Parts))
	}
	if resp.Content.Parts[0].Text != "Hello from Claude!" {
		t.Errorf("response text = %q, want %q", resp.Content.Parts[0].Text, "Hello from Claude!")
	}
	if !resp.TurnComplete {
		t.Error("expected TurnComplete to be true for non-streaming response")
	}
	if resp.UsageMetadata != nil {
		t.Error("expected UsageMetadata to be nil when usage absent from response")
	}
}

func TestAnthropicLLM_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify tools are sent in the request
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		// Verify tools were included
		tools, ok := reqBody["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Error("expected tools in request body")
		}

		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "I'll look up the weather."},
				{
					"type": "tool_use",
					"id":   "toolu_01A",
					"name": "get_weather",
					"input": map[string]any{
						"location": "San Francisco",
					},
				},
			},
			"stop_reason": "tool_use",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What's the weather in SF?")},
			},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{
					FunctionDeclarations: []*genai.FunctionDeclaration{
						{
							Name:        "get_weather",
							Description: "Get the weather for a location",
							ParametersJsonSchema: map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location": map[string]any{
										"type":        "string",
										"description": "The city name",
									},
								},
								"required": []any{"location"},
							},
						},
					},
				},
			},
		},
	}

	var responses []*adkmodel.LLMResponse
	var errs []error

	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		responses = append(responses, resp)
	}

	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}

	resp := responses[0]
	if resp.Content == nil {
		t.Fatal("response Content is nil")
	}
	if resp.Content.Role != "model" {
		t.Errorf("response role = %q, want %q", resp.Content.Role, "model")
	}

	// Should have 2 parts: text + function call
	if len(resp.Content.Parts) != 2 {
		t.Fatalf("got %d parts, want 2", len(resp.Content.Parts))
	}

	// First part: text
	if resp.Content.Parts[0].Text != "I'll look up the weather." {
		t.Errorf("part[0].Text = %q, want %q", resp.Content.Parts[0].Text, "I'll look up the weather.")
	}

	// Second part: function call
	fc := resp.Content.Parts[1].FunctionCall
	if fc == nil {
		t.Fatal("part[1].FunctionCall is nil")
	}
	if fc.ID != "toolu_01A" {
		t.Errorf("FunctionCall.ID = %q, want %q", fc.ID, "toolu_01A")
	}
	if fc.Name != "get_weather" {
		t.Errorf("FunctionCall.Name = %q, want %q", fc.Name, "get_weather")
	}
	if loc, ok := fc.Args["location"].(string); !ok || loc != "San Francisco" {
		t.Errorf("FunctionCall.Args[location] = %v, want %q", fc.Args["location"], "San Francisco")
	}
}

func TestAnthropicLLM_FunctionResponseConversion(t *testing.T) {
	// Test that FunctionResponse parts in request are converted to
	// Anthropic tool_result format in user messages
	var receivedReq map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedReq)

		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "The weather is sunny."},
			},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("What's the weather?")},
			},
			{
				Role: "model",
				Parts: []*genai.Part{
					{
						FunctionCall: &genai.FunctionCall{
							ID:   "toolu_01A",
							Name: "get_weather",
							Args: map[string]any{"location": "SF"},
						},
					},
				},
			},
			{
				Role: "user",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							ID:       "toolu_01A",
							Name:     "get_weather",
							Response: map[string]any{"output": "sunny, 72F"},
						},
					},
				},
			},
		},
	}

	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content.Parts[0].Text != "The weather is sunny." {
			t.Errorf("unexpected response text: %q", resp.Content.Parts[0].Text)
		}
	}

	// Verify the messages in the request
	messages, ok := receivedReq["messages"].([]any)
	if !ok {
		t.Fatal("no messages in request")
	}

	// Should have 3 messages: user, assistant (with tool_use), user (with tool_result)
	if len(messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(messages))
	}

	// Verify the assistant message has tool_use block
	assistantMsg := messages[1].(map[string]any)
	if assistantMsg["role"] != "assistant" {
		t.Errorf("message[1].role = %v, want %q", assistantMsg["role"], "assistant")
	}

	// Verify the user message has tool_result block
	toolResultMsg := messages[2].(map[string]any)
	if toolResultMsg["role"] != "user" {
		t.Errorf("message[2].role = %v, want %q", toolResultMsg["role"], "user")
	}
	content, ok := toolResultMsg["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array in tool_result message")
	}
	block := content[0].(map[string]any)
	if block["type"] != "tool_result" {
		t.Errorf("content[0].type = %v, want %q", block["type"], "tool_result")
	}
	if block["tool_use_id"] != "toolu_01A" {
		t.Errorf("content[0].tool_use_id = %v, want %q", block["tool_use_id"], "toolu_01A")
	}
}

func TestAnthropicLLM_WebSearchTool(t *testing.T) {
	var receivedReq map[string]any
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedReq)

		// Simulate Anthropic response with server-managed search results.
		// server_tool_use and web_search_tool_result blocks should be ignored by convertResponse.
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "I'll search for that."},
				{"type": "server_tool_use", "id": "srvtoolu_01", "name": "web_search", "input": map[string]any{"query": "Go programming"}},
				{"type": "web_search_tool_result", "tool_use_id": "srvtoolu_01", "content": []map[string]any{{"type": "web_search_result", "url": "https://go.dev"}}},
				{"type": "text", "text": "Based on my search, Go is a programming language."},
			},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Tell me about Go")}},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearch: &genai.GoogleSearch{}},
			},
		},
	}

	var responses []*adkmodel.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		responses = append(responses, resp)
	}

	// Verify beta header is set.
	if got := receivedHeaders.Get("anthropic-beta"); got != "web-search-2025-03-05" {
		t.Errorf("anthropic-beta header = %q, want %q", got, "web-search-2025-03-05")
	}

	// Verify web_search tool in request body.
	tools, ok := receivedReq["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected tools in request body")
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "web_search_20250305" {
		t.Errorf("tool type = %v, want web_search_20250305", tool["type"])
	}
	if tool["name"] != "web_search" {
		t.Errorf("tool name = %v, want web_search", tool["name"])
	}

	// Verify response: only text blocks extracted, server_tool_use/web_search_tool_result ignored.
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(responses))
	}
	resp := responses[0]
	if len(resp.Content.Parts) != 2 {
		t.Fatalf("got %d parts, want 2 (two text blocks)", len(resp.Content.Parts))
	}
	if resp.Content.Parts[0].Text != "I'll search for that." {
		t.Errorf("part[0] = %q, want %q", resp.Content.Parts[0].Text, "I'll search for that.")
	}
	if resp.Content.Parts[1].Text != "Based on my search, Go is a programming language." {
		t.Errorf("part[1] = %q, want %q", resp.Content.Parts[1].Text, "Based on my search, Go is a programming language.")
	}
	// No FunctionCall parts — server-managed tool doesn't produce them.
	for i, p := range resp.Content.Parts {
		if p.FunctionCall != nil {
			t.Errorf("part[%d] has unexpected FunctionCall", i)
		}
	}
}

func TestAnthropicLLM_WebSearchWithFunctionTools(t *testing.T) {
	var receivedReq map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedReq)

		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Hello")}},
		},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearch: &genai.GoogleSearch{}},
				{FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "get_weather", Description: "Get weather"},
				}},
			},
		},
	}

	for _, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Verify both native and custom tools coexist.
	tools, ok := receivedReq["tools"].([]any)
	if !ok {
		t.Fatal("expected tools in request body")
	}
	if len(tools) != 2 {
		t.Fatalf("got %d tools, want 2 (web_search + get_weather)", len(tools))
	}

	// First: web_search
	t0 := tools[0].(map[string]any)
	if t0["type"] != "web_search_20250305" {
		t.Errorf("tools[0].type = %v, want web_search_20250305", t0["type"])
	}
	// Second: get_weather
	t1 := tools[1].(map[string]any)
	if t1["name"] != "get_weather" {
		t.Errorf("tools[1].name = %v, want get_weather", t1["name"])
	}
}

func TestAnthropicLLM_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "model not found",
			},
		})
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "bad-model",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
		},
	}

	var gotError bool
	for _, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			gotError = true
		}
	}
	if !gotError {
		t.Error("expected an error for API error response")
	}
}

func TestAnthropicLLM_TokenUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "Hello!"}},
			"stop_reason": "end_turn",
			"usage": map[string]any{
				"input_tokens":  42,
				"output_tokens": 17,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))
	req := &adkmodel.LLMRequest{
		Model:    "claude-sonnet-4-20250514",
		Contents: []*genai.Content{{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("hi")}}},
	}

	var got []*adkmodel.LLMResponse
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, resp)
	}

	if len(got) != 1 {
		t.Fatalf("got %d responses, want 1", len(got))
	}
	u := got[0].UsageMetadata
	if u == nil {
		t.Fatal("UsageMetadata is nil, expected token counts")
	}
	if u.PromptTokenCount != 42 {
		t.Errorf("PromptTokenCount = %d, want 42", u.PromptTokenCount)
	}
	if u.CandidatesTokenCount != 17 {
		t.Errorf("CandidatesTokenCount = %d, want 17", u.CandidatesTokenCount)
	}
	if u.TotalTokenCount != 59 {
		t.Errorf("TotalTokenCount = %d, want 59", u.TotalTokenCount)
	}
}

func TestAnthropicLLM_MaxTokensFromConfig(t *testing.T) {
	var receivedReq map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedReq)

		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "ok"}},
			"stop_reason": "end_turn",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewAnthropicLLM("test-key", WithAnthropicBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "claude-sonnet-4-20250514",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
		},
		Config: &genai.GenerateContentConfig{
			MaxOutputTokens: 2048,
		},
	}

	for _, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Verify max_tokens was set from config
	maxTokens, ok := receivedReq["max_tokens"].(float64)
	if !ok {
		t.Fatal("max_tokens not in request")
	}
	if maxTokens != 2048 {
		t.Errorf("max_tokens = %v, want 2048", maxTokens)
	}
}
