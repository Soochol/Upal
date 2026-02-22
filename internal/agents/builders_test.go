package agents

import (
	"context"
	"iter"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// mockLLMForBuild satisfies adkmodel.LLM for build-time tests (no actual calls).
type mockLLMForBuild struct{}

func (m *mockLLMForBuild) Name() string { return "mock" }
func (m *mockLLMForBuild) GenerateContent(_ context.Context, _ *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		yield(&adkmodel.LLMResponse{
			Content:      &genai.Content{Role: "model", Parts: []*genai.Part{genai.NewPartFromText("mock")}},
			TurnComplete: true,
		}, nil)
	}
}

func TestBuildAgent_Input(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "input1", Type: upal.NodeTypeInput}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "input1" {
		t.Fatalf("expected 'input1', got %q", a.Name())
	}
}

func TestBuildAgent_Output(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "output1", Type: upal.NodeTypeOutput}
	a, err := BuildAgent(nd, nil, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if a.Name() != "output1" {
		t.Fatalf("expected 'output1', got %q", a.Name())
	}
}

func TestBuildAgent_AgentWithWebSearch(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "searcher",
		Type: upal.NodeTypeAgent,
		Config: map[string]any{
			"model":         "anthropic/claude-sonnet-4-6",
			"system_prompt": "You are a researcher.",
			"prompt":        "Search for {{topic}}",
			"tools":         []any{"web_search"},
		},
	}

	// Use a mock LLM — we only need to verify the agent builds without error.
	mockLLM := &mockLLMForBuild{}
	llms := map[string]adkmodel.LLM{"anthropic": mockLLM}

	a, err := BuildAgent(nd, llms, nil)
	if err != nil {
		t.Fatalf("BuildAgent with web_search tool should succeed: %v", err)
	}
	if a.Name() != "searcher" {
		t.Errorf("agent name = %q, want %q", a.Name(), "searcher")
	}
}

func TestBuildAgent_AgentWithWebSearchAndNilToolReg(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "searcher",
		Type: upal.NodeTypeAgent,
		Config: map[string]any{
			"model":         "anthropic/claude-sonnet-4-6",
			"system_prompt": "You are a researcher.",
			"prompt":        "Search",
			"tools":         []any{"web_search"},
		},
	}

	mockLLM := &mockLLMForBuild{}
	llms := map[string]adkmodel.LLM{"anthropic": mockLLM}

	// toolReg is nil — native tools should still work.
	a, err := BuildAgent(nd, llms, nil)
	if err != nil {
		t.Fatalf("BuildAgent with nil toolReg should succeed for native tools: %v", err)
	}
	if a.Name() != "searcher" {
		t.Errorf("agent name = %q, want %q", a.Name(), "searcher")
	}
}

func TestBuildAgent_UnknownType(t *testing.T) {
	nd := &upal.NodeDefinition{ID: "x", Type: "unknown"}
	_, err := BuildAgent(nd, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveTemplateFromState(t *testing.T) {
	state := &testState{
		data: map[string]any{
			"name":  "World",
			"input1": "Hello",
		},
	}

	tests := []struct {
		template string
		expected string
	}{
		{"Hello {{name}}", "Hello World"},
		{"{{input1}} {{name}}", "Hello World"},
		{"No templates here", "No templates here"},
		{"{{missing}}", "{{missing}}"},
	}

	for _, tt := range tests {
		result := resolveTemplateFromState(tt.template, state)
		if result != tt.expected {
			t.Errorf("resolveTemplateFromState(%q) = %q, want %q", tt.template, result, tt.expected)
		}
	}
}

// testState implements session.State for testing.
type testState struct {
	data map[string]any
}

func (s *testState) Get(key string) (any, error) {
	v, ok := s.data[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (s *testState) Set(key string, val any) error {
	s.data[key] = val
	return nil
}

func (s *testState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

func TestBuildPromptParts_PlainText(t *testing.T) {
	parts := buildPromptParts("hello world")
	if len(parts) != 1 {
		t.Fatalf("want 1 part, got %d", len(parts))
	}
	if parts[0].Text != "hello world" {
		t.Errorf("want text %q, got %q", "hello world", parts[0].Text)
	}
}

func TestBuildPromptParts_WithImage(t *testing.T) {
	// Minimal base64 encoded 1 byte
	prompt := "before data:image/png;base64,AA== after"
	parts := buildPromptParts(prompt)
	if len(parts) != 3 {
		t.Fatalf("want 3 parts (text, image, text), got %d", len(parts))
	}
	if !strings.Contains(parts[0].Text, "before") {
		t.Errorf("first part should contain 'before', got %q", parts[0].Text)
	}
	if parts[1].InlineData == nil {
		t.Error("second part should be inline image data")
	}
	if parts[1].InlineData.MIMEType != "image/png" {
		t.Errorf("want mime image/png, got %q", parts[1].InlineData.MIMEType)
	}
	if !strings.Contains(parts[2].Text, "after") {
		t.Errorf("third part should contain 'after', got %q", parts[2].Text)
	}
}

func TestParseDataURIPart_Valid(t *testing.T) {
	p := parseDataURIPart("data:image/jpeg;base64,/9j/AA==")
	if p == nil {
		t.Fatal("expected non-nil part")
	}
	if p.InlineData.MIMEType != "image/jpeg" {
		t.Errorf("want image/jpeg, got %q", p.InlineData.MIMEType)
	}
}

func TestParseDataURIPart_Invalid(t *testing.T) {
	if parseDataURIPart("not-a-data-uri") != nil {
		t.Error("expected nil for non-data URI")
	}
	if parseDataURIPart("data:image/png;notbase64,abc") != nil {
		t.Error("expected nil for non-base64 encoding")
	}
	if parseDataURIPart("data:image/png") != nil {
		t.Error("expected nil for missing semicolon")
	}
}

func TestBuildAgent_Tool_UnknownTool(t *testing.T) {
	reg := tools.NewRegistry()
	nd := &upal.NodeDefinition{
		ID:   "tts_node",
		Type: upal.NodeTypeTool,
		Config: map[string]any{
			"tool":  "tts",
			"input": map[string]any{"text": "hello"},
		},
	}
	_, err := BuildAgent(nd, nil, reg)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' in error, got: %v", err)
	}
}

func TestBuildAgent_Tool_NoToolReg(t *testing.T) {
	nd := &upal.NodeDefinition{
		ID:   "tts_node",
		Type: upal.NodeTypeTool,
		Config: map[string]any{
			"tool":  "tts",
			"input": map[string]any{"text": "hello"},
		},
	}
	_, err := BuildAgent(nd, nil, nil)
	if err == nil {
		t.Fatal("expected error when toolReg is nil")
	}
}

func TestBuildAgent_Tool_KnownTool(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&mockNamedTool{name: "tts"})

	nd := &upal.NodeDefinition{
		ID:   "tts_node",
		Type: upal.NodeTypeTool,
		Config: map[string]any{
			"tool":  "tts",
			"input": map[string]any{"text": "{{upstream}}", "voice": "Rachel"},
		},
	}
	a, err := BuildAgent(nd, nil, reg)
	if err != nil {
		t.Fatalf("expected no error for known tool: %v", err)
	}
	if a.Name() != "tts_node" {
		t.Errorf("want name 'tts_node', got %q", a.Name())
	}
}

// mockNamedTool is a no-op tool for testing ToolNodeBuilder.
type mockNamedTool struct{ name string }

func (m *mockNamedTool) Name() string                { return m.name }
func (m *mockNamedTool) Description() string         { return "mock" }
func (m *mockNamedTool) InputSchema() map[string]any { return nil }
func (m *mockNamedTool) Execute(_ context.Context, input any) (any, error) {
	return map[string]any{"result": "mock-output", "input": input}, nil
}

func TestResolveInputFromState(t *testing.T) {
	state := &testState{
		data: map[string]any{
			"upstream": "hello world",
		},
	}

	tests := []struct {
		name    string
		input   map[string]any
		wantKey string
		wantVal any
	}{
		{
			name:    "nil input returns nil",
			input:   nil,
			wantKey: "",
			wantVal: nil,
		},
		{
			name:    "string value with template resolved",
			input:   map[string]any{"text": "{{upstream}}"},
			wantKey: "text",
			wantVal: "hello world",
		},
		{
			name:    "string value without template unchanged",
			input:   map[string]any{"voice": "Rachel"},
			wantKey: "voice",
			wantVal: "Rachel",
		},
		{
			name:    "non-string value passed through",
			input:   map[string]any{"count": 42},
			wantKey: "count",
			wantVal: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveInputFromState(tt.input, state)
			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil result for nil input, got %v", result)
				}
				return
			}
			got, ok := result[tt.wantKey]
			if !ok {
				t.Fatalf("key %q not found in result", tt.wantKey)
			}
			if got != tt.wantVal {
				t.Errorf("key %q: got %v (%T), want %v (%T)", tt.wantKey, got, got, tt.wantVal, tt.wantVal)
			}
		})
	}
}

func TestBuildAgent_Tool_MissingToolConfig(t *testing.T) {
	reg := tools.NewRegistry()
	nd := &upal.NodeDefinition{
		ID:   "tts_node",
		Type: upal.NodeTypeTool,
		Config: map[string]any{
			// "tool" key intentionally omitted
			"input": map[string]any{"text": "hello"},
		},
	}
	_, err := BuildAgent(nd, nil, reg)
	if err == nil {
		t.Fatal("expected error when tool config field is missing")
	}
	if !strings.Contains(err.Error(), "missing required config field") {
		t.Errorf("expected 'missing required config field' in error, got: %v", err)
	}
}
