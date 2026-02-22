package model

import (
	"context"
	"fmt"
	"iter"
	"sync"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"

	"github.com/soochol/upal/internal/config"
)

var _ adkmodel.LLM = (*GeminiLLM)(nil)
var _ NativeToolProvider = (*GeminiLLM)(nil)

// GeminiLLM uses the google.golang.org/genai Go SDK directly for text
// generation. Unlike the OpenAI-compat path, this supports native Gemini
// features such as Google Search grounding via genai.Tool{GoogleSearch}.
type GeminiLLM struct {
	apiKey  string
	name    string
	once    sync.Once
	client  *genai.Client
	initErr error
}

// NewGeminiLLM creates a native Gemini text adapter for the given provider name.
func NewGeminiLLM(providerName, apiKey string) *GeminiLLM {
	return &GeminiLLM{
		name:   providerName,
		apiKey: apiKey,
	}
}

func (g *GeminiLLM) Name() string { return g.name }

func (g *GeminiLLM) ensureClient(ctx context.Context) error {
	g.once.Do(func() {
		g.client, g.initErr = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  g.apiKey,
			Backend: genai.BackendGeminiAPI,
		})
	})
	return g.initErr
}

// NativeTool implements NativeToolProvider.
// Returns the genai.Tool spec for well-known Upal native tools.
func (g *GeminiLLM) NativeTool(name string) (*genai.Tool, bool) {
	switch name {
	case "web_search":
		return &genai.Tool{GoogleSearch: &genai.GoogleSearch{}}, true
	}
	return nil, false
}

func (g *GeminiLLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		if err := g.ensureClient(ctx); err != nil {
			yield(nil, fmt.Errorf("gemini: client init failed: %w", err))
			return
		}

		cfg := req.Config
		if cfg == nil {
			cfg = &genai.GenerateContentConfig{}
		}

		emitLog(ctx, fmt.Sprintf("gemini: calling model %s", req.Model))

		if stream {
			for resp, err := range g.client.Models.GenerateContentStream(ctx, req.Model, req.Contents, cfg) {
				if err != nil {
					emitLog(ctx, fmt.Sprintf("gemini error: %s", err))
					yield(nil, fmt.Errorf("gemini: %w", err))
					return
				}
				if !yield(convertGeminiResponse(resp), nil) {
					return
				}
			}
		} else {
			resp, err := g.client.Models.GenerateContent(ctx, req.Model, req.Contents, cfg)
			if err != nil {
				emitLog(ctx, fmt.Sprintf("gemini error: %s", err))
				yield(nil, fmt.Errorf("gemini: %w", err))
				return
			}
			emitLog(ctx, "gemini: response received")
			yield(convertGeminiResponse(resp), nil)
		}
	}
}

func init() {
	RegisterProvider("gemini", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewGeminiLLM(name, cfg.APIKey)
	})
}

func convertGeminiResponse(resp *genai.GenerateContentResponse) *adkmodel.LLMResponse {
	if resp == nil || len(resp.Candidates) == 0 {
		return &adkmodel.LLMResponse{TurnComplete: true}
	}
	c := resp.Candidates[0]
	turnComplete := c.FinishReason != "" && c.FinishReason != genai.FinishReasonUnspecified
	r := &adkmodel.LLMResponse{
		Content:      c.Content,
		TurnComplete: turnComplete,
		FinishReason: c.FinishReason,
	}
	if resp.UsageMetadata != nil {
		r.UsageMetadata = resp.UsageMetadata
	}
	return r
}
