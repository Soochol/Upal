package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

const defaultOpenAITTSBaseURL = "https://api.openai.com/v1"

var _ adkmodel.LLM = (*OpenAITTSModel)(nil)

// OpenAITTSModel implements adkmodel.LLM for OpenAI's TTS API.
// system_prompt → speaking instructions, prompt → text to speak.
// Returns audio binary as InlineData (audio/mpeg).
type OpenAITTSModel struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAITTSModel creates an OpenAI TTS adapter.
// baseURL defaults to https://api.openai.com/v1 if empty.
func NewOpenAITTSModel(apiKey, baseURL string) *OpenAITTSModel {
	if baseURL == "" {
		baseURL = defaultOpenAITTSBaseURL
	}
	return &OpenAITTSModel{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

func (t *OpenAITTSModel) Name() string { return "openai-tts" }

func (t *OpenAITTSModel) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		resp, err := t.generate(ctx, req)
		yield(resp, err)
	}
}

type ttsRequest struct {
	Model        string `json:"model"`
	Input        string `json:"input"`
	Voice        string `json:"voice,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

func (t *OpenAITTSModel) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	// Extract text to speak from contents.
	var input string
	for _, c := range req.Contents {
		for _, p := range c.Parts {
			if p.Text != "" {
				input += p.Text
			}
		}
	}
	if input == "" {
		return nil, fmt.Errorf("openai-tts: no text to speak")
	}

	// Extract speaking instructions from system prompt.
	var instructions string
	if req.Config != nil && req.Config.SystemInstruction != nil {
		for _, p := range req.Config.SystemInstruction.Parts {
			if p.Text != "" {
				instructions = p.Text
				break
			}
		}
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "tts-1"
	}

	body := ttsRequest{
		Model:        modelName,
		Input:        input,
		Voice:        "alloy",
		Instructions: instructions,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai-tts: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		t.baseURL+"/audio/speech", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("openai-tts: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+t.apiKey)

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai-tts: HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	audioData, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai-tts: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai-tts: server returned %d: %s", httpResp.StatusCode, string(audioData))
	}

	mimeType := httpResp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		mimeType = strings.SplitN(mimeType, ";", 2)[0]
	}

	return &adkmodel.LLMResponse{
		Content: &genai.Content{
			Role: genai.RoleModel,
			Parts: []*genai.Part{
				{InlineData: &genai.Blob{Data: audioData, MIMEType: mimeType}},
			},
		},
		TurnComplete: true,
		FinishReason: genai.FinishReasonStop,
	}, nil
}
