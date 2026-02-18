package provider

import (
	"context"
	"testing"
)

type mockProvider struct{ name string }

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "mock"}, nil
}
func (m *mockProvider) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	return nil, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockProvider{name: "openai"})
	reg.Register(&mockProvider{name: "ollama"})
	p, ok := reg.Get("openai")
	if !ok {
		t.Fatal("openai not found")
	}
	if p.Name() != "openai" {
		t.Errorf("name: got %q", p.Name())
	}
}

func TestRegistry_Resolve(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockProvider{name: "openai"})
	p, model, err := reg.Resolve("openai/gpt-4o")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Name() != "openai" {
		t.Errorf("provider: got %q", p.Name())
	}
	if model != "gpt-4o" {
		t.Errorf("model: got %q", model)
	}
}

func TestRegistry_Resolve_Unknown(t *testing.T) {
	reg := NewRegistry()
	_, _, err := reg.Resolve("unknown/model")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
