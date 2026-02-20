package model

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestZImageGenerateContent(t *testing.T) {
	// Small 1x1 red PNG for testing.
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	t.Run("successful image generation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/generate" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			var req zimageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.Prompt != "a cat" {
				t.Errorf("prompt = %q, want %q", req.Prompt, "a cat")
			}

			resp := zimageResponse{
				Image:    base64.StdEncoding.EncodeToString(pngData),
				MIMEType: "image/png",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		llm := NewZImageLLM(server.URL)

		req := &adkmodel.LLMRequest{
			Model: "z-image",
			Contents: []*genai.Content{
				genai.NewContentFromText("a cat", genai.RoleUser),
			},
		}

		var result *adkmodel.LLMResponse
		for r, err := range llm.GenerateContent(context.Background(), req, false) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			result = r
		}

		if result == nil || result.Content == nil {
			t.Fatal("expected non-nil response content")
		}
		if len(result.Content.Parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(result.Content.Parts))
		}
		part := result.Content.Parts[0]
		if part.InlineData == nil {
			t.Fatal("expected InlineData")
		}
		if part.InlineData.MIMEType != "image/png" {
			t.Errorf("MIMEType = %q, want %q", part.InlineData.MIMEType, "image/png")
		}
		if len(part.InlineData.Data) != len(pngData) {
			t.Errorf("Data length = %d, want %d", len(part.InlineData.Data), len(pngData))
		}
		if !result.TurnComplete {
			t.Error("expected TurnComplete = true")
		}
	})

	t.Run("empty prompt", func(t *testing.T) {
		llm := NewZImageLLM("http://localhost:9999")
		req := &adkmodel.LLMRequest{
			Model: "z-image",
			Contents: []*genai.Content{
				genai.NewContentFromText("", genai.RoleUser),
			},
		}

		for _, err := range llm.GenerateContent(context.Background(), req, false) {
			if err == nil {
				t.Error("expected error for empty prompt")
			}
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "model not loaded", http.StatusServiceUnavailable)
		}))
		defer server.Close()

		llm := NewZImageLLM(server.URL)
		req := &adkmodel.LLMRequest{
			Model: "z-image",
			Contents: []*genai.Content{
				genai.NewContentFromText("a cat", genai.RoleUser),
			},
		}

		for _, err := range llm.GenerateContent(context.Background(), req, false) {
			if err == nil {
				t.Error("expected error for server error")
			}
		}
	})

	t.Run("default mime type", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := zimageResponse{
				Image:    base64.StdEncoding.EncodeToString(pngData),
				MIMEType: "", // empty — should default to image/png
			}
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		llm := NewZImageLLM(server.URL)
		req := &adkmodel.LLMRequest{
			Model:    "z-image",
			Contents: []*genai.Content{genai.NewContentFromText("test", genai.RoleUser)},
		}

		for r, err := range llm.GenerateContent(context.Background(), req, false) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.Content.Parts[0].InlineData.MIMEType != "image/png" {
				t.Errorf("expected default image/png, got %q", r.Content.Parts[0].InlineData.MIMEType)
			}
		}
	})
}

func TestZImageName(t *testing.T) {
	llm := NewZImageLLM("http://localhost:8090")
	if llm.Name() != "zimage" {
		t.Errorf("Name() = %q, want %q", llm.Name(), "zimage")
	}

	llm2 := NewZImageLLM("http://localhost:8090", WithZImageName("custom"))
	if llm2.Name() != "custom" {
		t.Errorf("Name() = %q, want %q", llm2.Name(), "custom")
	}
}
