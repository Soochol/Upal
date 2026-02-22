package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

func TestOpenAITTSGenerateContent(t *testing.T) {
	fakeAudio := []byte("fake-mp3-audio-data")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["input"] != "Hello world" {
			t.Errorf("unexpected input: %v", body["input"])
		}
		if body["model"] != "tts-1" {
			t.Errorf("unexpected model: %v", body["model"])
		}
		if body["instructions"] != "밝게 말하세요" {
			t.Errorf("unexpected instructions: %v", body["instructions"])
		}

		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeAudio)
	}))
	defer server.Close()

	tts := NewOpenAITTSModel("test-key", server.URL)

	req := &adkmodel.LLMRequest{
		Model: "tts-1",
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText("밝게 말하세요", genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText("Hello world", genai.RoleUser),
		},
	}

	var resp *adkmodel.LLMResponse
	for r, err := range tts.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		t.Fatal("nil response")
	}
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(resp.Content.Parts))
	}
	p := resp.Content.Parts[0]
	if p.InlineData == nil {
		t.Fatal("expected InlineData, got nil")
	}
	if p.InlineData.MIMEType != "audio/mpeg" {
		t.Errorf("unexpected MIME type: %s", p.InlineData.MIMEType)
	}
	if string(p.InlineData.Data) != string(fakeAudio) {
		t.Errorf("audio data mismatch")
	}
}

func TestOpenAITTSName(t *testing.T) {
	tts := NewOpenAITTSModel("key", "http://localhost")
	if tts.Name() != "openai-tts" {
		t.Errorf("unexpected name: %s", tts.Name())
	}
}
