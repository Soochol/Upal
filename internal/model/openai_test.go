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

func TestOpenAILLM_Name(t *testing.T) {
	llm := NewOpenAILLM("test-key")
	if got := llm.Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

func TestOpenAILLM_CustomName(t *testing.T) {
	llm := NewOpenAILLM("test-key", WithOpenAIName("ollama"))
	if got := llm.Name(); got != "ollama" {
		t.Errorf("Name() = %q, want %q", got, "ollama")
	}
}

func TestOpenAILLM_GenerateContent(t *testing.T) {
	// Set up a mock OpenAI server that returns a simple text response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path.
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}

		// Verify authorization header.
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		// Parse the request body to verify conversion.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		// Verify model is passed through.
		if reqBody["model"] != "gpt-4o" {
			t.Errorf("model = %v, want gpt-4o", reqBody["model"])
		}

		// Verify stream is false.
		if reqBody["stream"] != false {
			t.Errorf("stream = %v, want false", reqBody["stream"])
		}

		// Verify messages were converted.
		messages, ok := reqBody["messages"].([]any)
		if !ok {
			t.Fatalf("messages is not a slice: %T", reqBody["messages"])
		}
		if len(messages) != 2 {
			t.Fatalf("expected 2 messages (system + user), got %d", len(messages))
		}

		// Verify system message.
		sysMsg := messages[0].(map[string]any)
		if sysMsg["role"] != "system" {
			t.Errorf("first message role = %v, want system", sysMsg["role"])
		}
		if sysMsg["content"] != "You are helpful." {
			t.Errorf("system content = %v, want 'You are helpful.'", sysMsg["content"])
		}

		// Verify user message.
		userMsg := messages[1].(map[string]any)
		if userMsg["role"] != "user" {
			t.Errorf("second message role = %v, want user", userMsg["role"])
		}

		// Return a mock response.
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello! How can I help?",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("test-key", WithOpenAIBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "gpt-4o",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hello")},
			},
		},
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{genai.NewPartFromText("You are helpful.")},
			},
		},
	}

	var responses []*adkmodel.LLMResponse
	var lastErr error
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			lastErr = err
			break
		}
		responses = append(responses, resp)
	}

	if lastErr != nil {
		t.Fatalf("GenerateContent returned error: %v", lastErr)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Content == nil {
		t.Fatal("response Content is nil")
	}
	if resp.Content.Role != "model" {
		t.Errorf("response role = %q, want %q", resp.Content.Role, "model")
	}
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(resp.Content.Parts))
	}
	if resp.Content.Parts[0].Text != "Hello! How can I help?" {
		t.Errorf("text = %q, want %q", resp.Content.Parts[0].Text, "Hello! How can I help?")
	}
	if !resp.TurnComplete {
		t.Error("expected TurnComplete to be true")
	}
}

func TestOpenAILLM_ToolCalls(t *testing.T) {
	// Set up a mock OpenAI server that returns tool calls.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request to verify tools were sent.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		// Verify tools were converted from genai format.
		tools, ok := reqBody["tools"].([]any)
		if !ok {
			t.Fatalf("tools is not a slice: %T", reqBody["tools"])
		}
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		tool := tools[0].(map[string]any)
		if tool["type"] != "function" {
			t.Errorf("tool type = %v, want function", tool["type"])
		}
		fn := tool["function"].(map[string]any)
		if fn["name"] != "get_weather" {
			t.Errorf("function name = %v, want get_weather", fn["name"])
		}

		// Return a tool call response.
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": nil,
						"tool_calls": []map[string]any{
							{
								"id":   "call_abc123",
								"type": "function",
								"function": map[string]any{
									"name":      "get_weather",
									"arguments": `{"location":"San Francisco"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("test-key", WithOpenAIBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "gpt-4o",
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
							Description: "Get weather for a location",
							Parameters: &genai.Schema{
								Type: "OBJECT",
								Properties: map[string]*genai.Schema{
									"location": {Type: "STRING", Description: "City name"},
								},
								Required: []string{"location"},
							},
						},
					},
				},
			},
		},
	}

	var responses []*adkmodel.LLMResponse
	var lastErr error
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			lastErr = err
			break
		}
		responses = append(responses, resp)
	}

	if lastErr != nil {
		t.Fatalf("GenerateContent returned error: %v", lastErr)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Content == nil {
		t.Fatal("response Content is nil")
	}
	if resp.Content.Role != "model" {
		t.Errorf("response role = %q, want %q", resp.Content.Role, "model")
	}
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(resp.Content.Parts))
	}

	part := resp.Content.Parts[0]
	if part.FunctionCall == nil {
		t.Fatal("expected FunctionCall part, got nil")
	}
	if part.FunctionCall.Name != "get_weather" {
		t.Errorf("function name = %q, want %q", part.FunctionCall.Name, "get_weather")
	}
	if part.FunctionCall.ID != "call_abc123" {
		t.Errorf("function call ID = %q, want %q", part.FunctionCall.ID, "call_abc123")
	}
	if loc, ok := part.FunctionCall.Args["location"]; !ok || loc != "San Francisco" {
		t.Errorf("function args = %v, want location=San Francisco", part.FunctionCall.Args)
	}
	if !resp.TurnComplete {
		t.Error("expected TurnComplete to be true")
	}
}

func TestOpenAILLM_FunctionResponseConversion(t *testing.T) {
	// Verify that FunctionResponse parts are converted to tool role messages.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}

		messages := reqBody["messages"].([]any)
		// Expect: user, assistant (with tool_calls), tool
		if len(messages) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(messages))
		}

		// Verify the tool response message.
		toolMsg := messages[2].(map[string]any)
		if toolMsg["role"] != "tool" {
			t.Errorf("tool message role = %v, want tool", toolMsg["role"])
		}
		if toolMsg["tool_call_id"] != "call_abc123" {
			t.Errorf("tool_call_id = %v, want call_abc123", toolMsg["tool_call_id"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "The weather is sunny.",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("test-key", WithOpenAIBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "gpt-4o",
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
							ID:   "call_abc123",
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
							ID:       "call_abc123",
							Name:     "get_weather",
							Response: map[string]any{"weather": "sunny"},
						},
					},
				},
			},
		},
	}

	var responses []*adkmodel.LLMResponse
	var lastErr error
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			lastErr = err
			break
		}
		responses = append(responses, resp)
	}

	if lastErr != nil {
		t.Fatalf("GenerateContent returned error: %v", lastErr)
	}
	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Content.Parts[0].Text != "The weather is sunny." {
		t.Errorf("text = %q, want %q", responses[0].Content.Parts[0].Text, "The weather is sunny.")
	}
}

func TestOpenAILLM_NoAuthHeader(t *testing.T) {
	// When API key is empty, no Authorization header should be sent (e.g., Ollama).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("expected no Authorization header, got %q", got)
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello from Ollama",
					},
					"finish_reason": "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("", WithOpenAIBaseURL(server.URL))

	req := &adkmodel.LLMRequest{
		Model: "llama3.2",
		Contents: []*genai.Content{
			{
				Role:  "user",
				Parts: []*genai.Part{genai.NewPartFromText("Hi")},
			},
		},
	}

	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("GenerateContent returned error: %v", err)
		}
		if resp.Content.Parts[0].Text != "Hello from Ollama" {
			t.Errorf("text = %q, want %q", resp.Content.Parts[0].Text, "Hello from Ollama")
		}
	}
}
