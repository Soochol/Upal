package model_test

import (
	"testing"

	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/model"
)

func TestBuildLLM_KnownType(t *testing.T) {
	llm, ok := model.BuildLLM("myprovider", config.ProviderConfig{
		Type:   "anthropic",
		APIKey: "test-key",
	})
	if !ok {
		t.Fatal("expected ok=true for anthropic")
	}
	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestBuildLLM_UnknownTypeWithURL(t *testing.T) {
	llm, ok := model.BuildLLM("myprovider", config.ProviderConfig{
		Type:   "some-openai-compat",
		URL:    "http://localhost:1234/v1",
		APIKey: "key",
	})
	if !ok {
		t.Fatal("expected ok=true for fallback with URL")
	}
	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestBuildLLM_UnknownTypeNoURL(t *testing.T) {
	_, ok := model.BuildLLM("myprovider", config.ProviderConfig{
		Type: "totally-unknown",
	})
	if ok {
		t.Fatal("expected ok=false for unknown type with no URL")
	}
}
