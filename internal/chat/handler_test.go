package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// --- mocks ---

// mockSkillsProvider satisfies skills.Provider for testing.
type mockSkillsProvider struct{}

func (m *mockSkillsProvider) Get(name string) string      { return "" }
func (m *mockSkillsProvider) GetPrompt(name string) string { return "You are a test assistant." }

// mockLLM satisfies adkmodel.LLM, returning pre-configured responses per turn.
type mockLLM struct {
	responses []*adkmodel.LLMResponse
	callIndex int
}

func (m *mockLLM) Name() string { return "mock" }

func (m *mockLLM) GenerateContent(_ context.Context, _ *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		idx := m.callIndex
		m.callIndex++
		if idx < len(m.responses) {
			yield(m.responses[idx], nil)
		}
	}
}

// sseEvent represents a parsed SSE event.
type sseEvent struct {
	Event string
	Data  string
}

// parseSSEEvents splits an SSE body into individual events.
func parseSSEEvents(body string) []sseEvent {
	var events []sseEvent
	blocks := strings.Split(body, "\n\n")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var ev sseEvent
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "event: ") {
				ev.Event = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				ev.Data = strings.TrimPrefix(line, "data: ")
			}
			// id: lines are parsed but not stored — tests don't assert on them.
		}
		if ev.Event != "" || ev.Data != "" {
			events = append(events, ev)
		}
	}
	return events
}

// newTestHandler creates a Handler with the given mock LLM.
func newTestHandler(llm adkmodel.LLM, registry *ChatRegistry) *Handler {
	if registry == nil {
		registry = NewRegistry()
	}
	return NewHandler(
		registry,
		&mockSkillsProvider{},
		nil,
		func(ctx context.Context) (adkmodel.LLM, string, error) {
			return llm, "mock-model", nil
		},
	)
}

// postChat sends a POST request with the given ChatRequest body.
func postChat(handler *Handler, req ChatRequest) *httptest.ResponseRecorder {
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

// --- tests ---

func TestHandler_MissingMessage(t *testing.T) {
	handler := newTestHandler(&mockLLM{}, nil)
	w := postChat(handler, ChatRequest{Page: "workflows"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "message is required") {
		t.Errorf("expected 'message is required' in body, got %q", w.Body.String())
	}
}

func TestHandler_MissingPage(t *testing.T) {
	handler := newTestHandler(&mockLLM{}, nil)
	w := postChat(handler, ChatRequest{Message: "hello"})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "page is required") {
		t.Errorf("expected 'page is required' in body, got %q", w.Body.String())
	}
}

func TestHandler_SSEHeaders(t *testing.T) {
	llm := &mockLLM{
		responses: []*adkmodel.LLMResponse{
			{
				Content: &genai.Content{
					Role:  "model",
					Parts: []*genai.Part{genai.NewPartFromText("hi")},
				},
				TurnComplete: true,
			},
		},
	}
	handler := newTestHandler(llm, nil)
	w := postChat(handler, ChatRequest{Message: "hello", Page: "workflows"})

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
	if conn := w.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("Connection = %q, want keep-alive", conn)
	}
}

func TestHandler_TextResponse(t *testing.T) {
	llm := &mockLLM{
		responses: []*adkmodel.LLMResponse{
			{
				Content: &genai.Content{
					Role:  "model",
					Parts: []*genai.Part{genai.NewPartFromText("Hello! How can I help you?")},
				},
				TurnComplete: true,
			},
		},
	}
	handler := newTestHandler(llm, nil)
	w := postChat(handler, ChatRequest{Message: "hello", Page: "workflows"})

	events := parseSSEEvents(w.Body.String())

	// Expect text_delta and done events.
	var hasTextDelta, hasDone bool
	for _, ev := range events {
		switch ev.Event {
		case "text_delta":
			hasTextDelta = true
			var data map[string]any
			if err := json.Unmarshal([]byte(ev.Data), &data); err != nil {
				t.Fatalf("failed to parse text_delta data: %v", err)
			}
			text, _ := data["text"].(string)
			if !strings.Contains(text, "Hello! How can I help you?") {
				t.Errorf("text_delta text = %q, want to contain response text", text)
			}
		case "done":
			hasDone = true
			var data map[string]any
			if err := json.Unmarshal([]byte(ev.Data), &data); err != nil {
				t.Fatalf("failed to parse done data: %v", err)
			}
			content, _ := data["content"].(string)
			if !strings.Contains(content, "Hello! How can I help you?") {
				t.Errorf("done content = %q, want to contain response text", content)
			}
		}
	}
	if !hasTextDelta {
		t.Error("expected text_delta event in SSE stream")
	}
	if !hasDone {
		t.Error("expected done event in SSE stream")
	}
}

func TestHandler_ToolCallFlow(t *testing.T) {
	// Set up a registry with a test tool.
	registry := NewRegistry()
	registry.Register(&ChatTool{
		Name:        "get_info",
		Description: "Get some info",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}},
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			return map[string]any{"info": "result for " + args["query"].(string)}, nil
		},
	})
	registry.AddRule(Rule{
		Page:  "workflows",
		Tools: []string{"get_info"},
	})

	// Turn 1: LLM returns a tool call.
	// Turn 2: LLM returns text after receiving tool results.
	llm := &mockLLM{
		responses: []*adkmodel.LLMResponse{
			{
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{
							FunctionCall: &genai.FunctionCall{
								ID:   "call_1",
								Name: "get_info",
								Args: map[string]any{"query": "test"},
							},
						},
					},
				},
				TurnComplete: true,
			},
			{
				Content: &genai.Content{
					Role:  "model",
					Parts: []*genai.Part{genai.NewPartFromText("Based on the info, here is your answer.")},
				},
				TurnComplete: true,
			},
		},
	}

	handler := newTestHandler(llm, registry)
	w := postChat(handler, ChatRequest{Message: "use tool", Page: "workflows"})

	events := parseSSEEvents(w.Body.String())

	// Expected order: tool_call → tool_result → text_delta → done.
	expectedOrder := []string{"tool_call", "tool_result", "text_delta", "done"}
	var actualOrder []string
	for _, ev := range events {
		actualOrder = append(actualOrder, ev.Event)
	}

	if len(actualOrder) != len(expectedOrder) {
		t.Fatalf("expected %d events %v, got %d events %v", len(expectedOrder), expectedOrder, len(actualOrder), actualOrder)
	}
	for i, want := range expectedOrder {
		if actualOrder[i] != want {
			t.Errorf("event[%d] = %q, want %q (full: %v)", i, actualOrder[i], want, actualOrder)
		}
	}

	// Verify tool_call data.
	var toolCallData map[string]any
	if err := json.Unmarshal([]byte(events[0].Data), &toolCallData); err != nil {
		t.Fatalf("failed to parse tool_call data: %v", err)
	}
	if toolCallData["name"] != "get_info" {
		t.Errorf("tool_call name = %v, want get_info", toolCallData["name"])
	}

	// Verify tool_result data.
	var toolResultData map[string]any
	if err := json.Unmarshal([]byte(events[1].Data), &toolResultData); err != nil {
		t.Fatalf("failed to parse tool_result data: %v", err)
	}
	if toolResultData["success"] != true {
		t.Errorf("tool_result success = %v, want true", toolResultData["success"])
	}
	if toolResultData["name"] != "get_info" {
		t.Errorf("tool_result name = %v, want get_info", toolResultData["name"])
	}
}

func TestHandler_InvalidBody(t *testing.T) {
	handler := newTestHandler(&mockLLM{}, nil)

	r := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandler_HistoryIsPassedToLLM(t *testing.T) {
	// Verify that conversation history is included in the LLM request.
	var capturedReq *adkmodel.LLMRequest
	llm := &capturingLLM{
		response: &adkmodel.LLMResponse{
			Content: &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText("response")},
			},
			TurnComplete: true,
		},
		onCall: func(req *adkmodel.LLMRequest) {
			capturedReq = req
		},
	}

	handler := newTestHandler(llm, nil)
	postChat(handler, ChatRequest{
		Message: "new message",
		Page:    "workflows",
		History: []ChatMessage{
			{Role: "user", Content: "previous question"},
			{Role: "assistant", Content: "previous answer"},
		},
	})

	if capturedReq == nil {
		t.Fatal("LLM was not called")
	}
	// History (2 messages) + current message = 3 content entries.
	if len(capturedReq.Contents) != 3 {
		t.Fatalf("expected 3 contents (2 history + 1 new), got %d", len(capturedReq.Contents))
	}
	if capturedReq.Contents[0].Parts[0].Text != "previous question" {
		t.Errorf("first content = %q, want 'previous question'", capturedReq.Contents[0].Parts[0].Text)
	}
	if capturedReq.Contents[1].Parts[0].Text != "previous answer" {
		t.Errorf("second content = %q, want 'previous answer'", capturedReq.Contents[1].Parts[0].Text)
	}
	if capturedReq.Contents[2].Parts[0].Text != "new message" {
		t.Errorf("third content = %q, want 'new message'", capturedReq.Contents[2].Parts[0].Text)
	}
}

// capturingLLM captures the request and returns a fixed response.
type capturingLLM struct {
	response *adkmodel.LLMResponse
	onCall   func(req *adkmodel.LLMRequest)
}

func (m *capturingLLM) Name() string { return "capturing" }

func (m *capturingLLM) GenerateContent(_ context.Context, req *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		if m.onCall != nil {
			m.onCall(req)
		}
		yield(m.response, nil)
	}
}
