package agents

import (
	"context"
	"iter"
	"testing"

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
