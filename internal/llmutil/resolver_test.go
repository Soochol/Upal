package llmutil

import (
	"testing"

	adkmodel "google.golang.org/adk/model"
)

func TestMapResolver_EmptyReturnsDefault(t *testing.T) {
	r := NewMapResolver(nil, nil, "default-model")
	_, model, err := r.Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "default-model" {
		t.Errorf("expected default-model, got %s", model)
	}
}

func TestMapResolver_InvalidFormat(t *testing.T) {
	r := NewMapResolver(nil, nil, "")
	_, _, err := r.Resolve("no-slash")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestMapResolver_UnknownProvider(t *testing.T) {
	r := NewMapResolver(map[string]adkmodel.LLM{}, nil, "")
	_, _, err := r.Resolve("unknown/model")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

type stubLLM struct{ adkmodel.LLM }

func TestMapResolver_ValidProviderModel(t *testing.T) {
	fake := &stubLLM{}
	r := NewMapResolver(map[string]adkmodel.LLM{"anthropic": fake}, nil, "")
	llm, model, err := r.Resolve("anthropic/claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llm != fake {
		t.Error("expected the anthropic LLM instance")
	}
	if model != "claude-sonnet-4-6" {
		t.Errorf("expected claude-sonnet-4-6, got %s", model)
	}
}
