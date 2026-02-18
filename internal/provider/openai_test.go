package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProvider_ChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "gpt-4o" {
			t.Errorf("unexpected model: %v", reqBody["model"])
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Hello! How can I help?"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("openai", server.URL+"/v1", "test-key")
	resp, err := p.ChatCompletion(context.Background(), &ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Content != "Hello! How can I help?" {
		t.Errorf("content: got %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("finish_reason: got %q", resp.FinishReason)
	}
}

func TestOpenAIProvider_ChatCompletion_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role": "assistant", "content": "",
						"tool_calls": []map[string]any{
							{"id": "call_1", "type": "function", "function": map[string]any{"name": "web_search", "arguments": `{"query":"AI trends"}`}},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("openai", server.URL+"/v1", "test-key")
	resp, err := p.ChatCompletion(context.Background(), &ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "Search for AI trends"}},
		Tools:    []ToolDefinition{{Name: "web_search", Description: "Search the web"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool calls: got %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "web_search" {
		t.Errorf("tool name: got %q", resp.ToolCalls[0].Name)
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider("ollama", "http://localhost:11434/v1", "")
	if p.Name() != "ollama" {
		t.Errorf("name: got %q, want ollama", p.Name())
	}
}
