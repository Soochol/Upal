package model

import (
	"testing"

	"google.golang.org/genai"
)

func TestIsImageCapableModel(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"gemini-2.0-flash-exp", true},
		{"gemini-2.0-flash-preview-image-generation", false},
		{"gemini-2.5-flash-preview-image-generation", true},
		{"gemini-2.5-flash", false},
		{"gemini-2.5-pro", false},
		{"gpt-4o", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := isImageCapableModel(tt.model); got != tt.want {
				t.Errorf("isImageCapableModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestGeminiImageConvertResponse(t *testing.T) {
	llm := NewGeminiImageLLM("test-key")

	t.Run("image response", func(t *testing.T) {
		imgData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Role: "model",
						Parts: []*genai.Part{
							{InlineData: &genai.Blob{Data: imgData, MIMEType: "image/png"}},
						},
					},
					FinishReason: genai.FinishReasonStop,
				},
			},
		}

		llmResp, err := llm.convertResponse(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if llmResp.Content == nil || len(llmResp.Content.Parts) == 0 {
			t.Fatal("expected content with parts")
		}
		part := llmResp.Content.Parts[0]
		if part.InlineData == nil {
			t.Fatal("expected InlineData in part")
		}
		if part.InlineData.MIMEType != "image/png" {
			t.Errorf("MIMEType = %q, want %q", part.InlineData.MIMEType, "image/png")
		}
		if len(part.InlineData.Data) != len(imgData) {
			t.Errorf("Data length = %d, want %d", len(part.InlineData.Data), len(imgData))
		}
		if !llmResp.TurnComplete {
			t.Error("expected TurnComplete = true")
		}
	})

	t.Run("mixed text and image", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Role: "model",
						Parts: []*genai.Part{
							{Text: "Here is your image:"},
							{InlineData: &genai.Blob{Data: []byte{1, 2, 3}, MIMEType: "image/jpeg"}},
						},
					},
					FinishReason: genai.FinishReasonStop,
				},
			},
		}

		llmResp, err := llm.convertResponse(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(llmResp.Content.Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(llmResp.Content.Parts))
		}
		if llmResp.Content.Parts[0].Text != "Here is your image:" {
			t.Errorf("text part mismatch")
		}
		if llmResp.Content.Parts[1].InlineData == nil {
			t.Error("expected InlineData in second part")
		}
	})

	t.Run("no candidates", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{}
		_, err := llm.convertResponse(resp)
		if err == nil {
			t.Error("expected error for no candidates")
		}
	})

	t.Run("nil content", func(t *testing.T) {
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{Content: nil}},
		}
		_, err := llm.convertResponse(resp)
		if err == nil {
			t.Error("expected error for nil content")
		}
	})
}

func TestGeminiImageName(t *testing.T) {
	llm := NewGeminiImageLLM("key")
	if llm.Name() != "gemini-image" {
		t.Errorf("Name() = %q, want %q", llm.Name(), "gemini-image")
	}
}
