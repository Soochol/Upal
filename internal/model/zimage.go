package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"
)

// imageParamsKey is the context key for ImageParams.
type imageParamsKey struct{}

// ImageParams holds configurable image generation parameters.
type ImageParams struct {
	Width  int
	Height int
	Steps  int
}

// WithImageParams returns a context carrying image generation parameters.
func WithImageParams(ctx context.Context, p ImageParams) context.Context {
	return context.WithValue(ctx, imageParamsKey{}, p)
}

var _ adkmodel.LLM = (*ZImageLLM)(nil)

// ZImageLLM calls a local Python inference server for Z-IMAGE text-to-image generation.
type ZImageLLM struct {
	serverURL string
	name      string
	client    *http.Client
}

// ZImageOption configures a ZImageLLM instance.
type ZImageOption func(*ZImageLLM)

// WithZImageName sets a custom name for the LLM instance.
func WithZImageName(name string) ZImageOption {
	return func(z *ZImageLLM) { z.name = name }
}

// NewZImageLLM creates a Z-IMAGE adapter that calls the given server URL.
func NewZImageLLM(serverURL string, opts ...ZImageOption) *ZImageLLM {
	z := &ZImageLLM{
		serverURL: serverURL,
		name:      "zimage",
		client:    &http.Client{},
	}
	for _, opt := range opts {
		opt(z)
	}
	return z
}

func (z *ZImageLLM) Name() string { return z.name }

func (z *ZImageLLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		resp, err := z.generate(ctx, req)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(resp, nil)
	}
}

// zimageRequest is the JSON body sent to the Python inference server.
type zimageRequest struct {
	Prompt string `json:"prompt"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Steps  int    `json:"steps,omitempty"`
}

// zimageResponse is the JSON body returned by the Python inference server.
type zimageResponse struct {
	Image    string `json:"image"`     // base64-encoded image
	MIMEType string `json:"mime_type"` // e.g. "image/png"
}

func (z *ZImageLLM) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	// Extract prompt text from request contents.
	var prompt string
	for _, content := range req.Contents {
		for _, part := range content.Parts {
			if part.Text != "" {
				prompt = part.Text
			}
		}
	}
	if prompt == "" {
		return nil, fmt.Errorf("zimage: no text prompt provided")
	}

	body := zimageRequest{Prompt: prompt}
	if p, ok := ctx.Value(imageParamsKey{}).(ImageParams); ok {
		body.Width = p.Width
		body.Height = p.Height
		body.Steps = p.Steps
	}
	if body.Width == 0 {
		body.Width = 1024
	}
	if body.Height == 0 {
		body.Height = 1024
	}
	if body.Steps == 0 {
		body.Steps = 28
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("zimage: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, z.serverURL+"/generate", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("zimage: create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := z.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("zimage: HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("zimage: read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zimage: server returned %d: %s", httpResp.StatusCode, string(respBody))
	}

	var apiResp zimageResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("zimage: unmarshal response: %w", err)
	}

	imageData, err := base64.StdEncoding.DecodeString(apiResp.Image)
	if err != nil {
		return nil, fmt.Errorf("zimage: decode image base64: %w", err)
	}

	mimeType := apiResp.MIMEType
	if mimeType == "" {
		mimeType = "image/png"
	}

	return &adkmodel.LLMResponse{
		Content: &genai.Content{
			Role: genai.RoleModel,
			Parts: []*genai.Part{
				{InlineData: &genai.Blob{Data: imageData, MIMEType: mimeType}},
			},
		},
		TurnComplete: true,
	}, nil
}
