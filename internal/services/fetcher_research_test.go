package services

import (
	"context"
	"iter"
	"testing"

	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// stubResolver returns a stubLLM that implements NativeToolProvider.
type stubResolver struct{}

func (r *stubResolver) Resolve(modelID string) (adkmodel.LLM, string, error) {
	return &stubLLM{}, modelID, nil
}

// stubLLM is a minimal LLM mock that supports native tools but never generates content.
type stubLLM struct{}

func (l *stubLLM) Name() string { return "stub" }

func (l *stubLLM) GenerateContent(_ context.Context, _ *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {}
}

// Satisfy NativeToolProvider so research fetcher doesn't reject the model.
var _ upalmodel.NativeToolProvider = (*stubLLM)(nil)

func (l *stubLLM) NativeTool(name string) (*genai.Tool, bool) {
	return &genai.Tool{}, true
}

func TestResearchFetcher_EmptyTopic(t *testing.T) {
	f := &researchFetcher{}
	_, _, err := f.Fetch(context.Background(), upal.CollectSource{Type: "research"})
	if err == nil {
		t.Fatal("expected error for empty topic")
	}
	if err.Error() != "research source requires a topic" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResearchFetcher_InvalidDepth(t *testing.T) {
	// Use a stub resolver that returns a mock LLM supporting native tools.
	f := &researchFetcher{resolver: &stubResolver{}}
	_, _, err := f.Fetch(context.Background(), upal.CollectSource{
		Type:  "research",
		Topic: "test topic",
		Model: "test/model",
		Depth: "ultra-deep",
	})
	if err == nil {
		t.Fatal("expected error for invalid depth")
	}
	if err.Error() != `unknown research depth "ultra-deep"` {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildResearchSystemPrompt_Fallback(t *testing.T) {
	result := buildResearchSystemPrompt("", "light")
	if result == "" {
		t.Error("expected fallback prompt, got empty")
	}
}

func TestBuildResearchSystemPrompt_DeepSection(t *testing.T) {
	content := "intro\n## Deep Mode — System Prompt\nYou are a deep researcher."
	result := buildResearchSystemPrompt(content, "deep")
	if result != "You are a deep researcher." {
		t.Errorf("expected deep prompt, got %q", result)
	}
}

func TestBuildResearchSystemPrompt_LightSection(t *testing.T) {
	content := "intro\n## Light Mode — System Prompt\nYou are a light searcher.\n## Deep Mode — System Prompt\nDeep stuff."
	result := buildResearchSystemPrompt(content, "light")
	if result != "You are a light searcher." {
		t.Errorf("expected light prompt, got %q", result)
	}
}

func TestParseResearchSources(t *testing.T) {
	text := "Check [Google](https://google.com) and [GitHub](https://github.com) and [Google](https://google.com) again."
	sources := parseResearchSources(text)
	if len(sources) != 2 {
		t.Fatalf("expected 2 unique sources, got %d", len(sources))
	}
	if sources[0]["url"] != "https://google.com" {
		t.Errorf("expected google.com, got %v", sources[0]["url"])
	}
	if sources[1]["url"] != "https://github.com" {
		t.Errorf("expected github.com, got %v", sources[1]["url"])
	}
}

func TestParseResearchSources_NoLinks(t *testing.T) {
	sources := parseResearchSources("no links here")
	if sources != nil {
		t.Errorf("expected nil, got %v", sources)
	}
}
