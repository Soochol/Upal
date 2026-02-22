package model

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"sync"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"

	"github.com/soochol/upal/internal/config"
)

var _ adkmodel.LLM = (*GeminiImageLLM)(nil)

// GeminiImageLLM uses the google.golang.org/genai Go SDK directly
// to support image generation via ResponseModalities.
// This is separate from the OpenAI-compat path used for text-only Gemini.
type GeminiImageLLM struct {
	apiKey string
	name   string

	once   sync.Once
	client *genai.Client
	initErr error
}

// NewGeminiImageLLM creates a Gemini image generation adapter.
func NewGeminiImageLLM(apiKey string) *GeminiImageLLM {
	return &GeminiImageLLM{
		apiKey: apiKey,
		name:   "gemini-image",
	}
}

func (g *GeminiImageLLM) Name() string { return g.name }

// ensureClient lazily initializes the genai.Client on first use.
func (g *GeminiImageLLM) ensureClient(ctx context.Context) error {
	g.once.Do(func() {
		g.client, g.initErr = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  g.apiKey,
			Backend: genai.BackendGeminiAPI,
		})
	})
	return g.initErr
}

func (g *GeminiImageLLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		resp, err := g.generate(ctx, req)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(resp, nil)
	}
}

func (g *GeminiImageLLM) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	if err := g.ensureClient(ctx); err != nil {
		return nil, fmt.Errorf("gemini-image: client init failed: %w", err)
	}

	cfg := req.Config
	if cfg == nil {
		cfg = &genai.GenerateContentConfig{}
	}

	// Auto-set ResponseModalities for image-capable models.
	if isImageCapableModel(req.Model) && len(cfg.ResponseModalities) == 0 {
		modalities := []string{"TEXT", "IMAGE"}
		cfg.ResponseModalities = modalities
	}

	emitLog(ctx, fmt.Sprintf("gemini-image: calling model %s", req.Model))

	result, err := g.client.Models.GenerateContent(ctx, req.Model, req.Contents, cfg)
	if err != nil {
		emitLog(ctx, fmt.Sprintf("gemini-image error: %s", err))
		return nil, fmt.Errorf("gemini-image: %w", err)
	}

	emitLog(ctx, "gemini-image: response received")
	return g.convertResponse(result)
}

func (g *GeminiImageLLM) convertResponse(resp *genai.GenerateContentResponse) (*adkmodel.LLMResponse, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini-image: no candidates in response")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return nil, fmt.Errorf("gemini-image: no content in candidate")
	}

	return &adkmodel.LLMResponse{
		Content:      candidate.Content,
		TurnComplete: true,
		FinishReason: candidate.FinishReason,
	}, nil
}

func init() {
	RegisterProvider("gemini-image", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewGeminiImageLLM(cfg.APIKey)
	})
}

// isImageCapableModel returns true for Gemini models that support image output.
func isImageCapableModel(model string) bool {
	imageModels := []string{
		"gemini-2.0-flash-exp-image-generation",
		"gemini-2.5-flash-image",
		"gemini-3-pro-image-preview",
	}
	for _, m := range imageModels {
		if strings.EqualFold(model, m) {
			return true
		}
	}
	return false
}
