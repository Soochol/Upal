package upal

import (
	"fmt"
	"slices"
)

// AIProviderCategory classifies AI providers by capability.
type AIProviderCategory string

const (
	AICategoryLLM   AIProviderCategory = "llm"
	AICategoryTTS   AIProviderCategory = "tts"
	AICategoryImage AIProviderCategory = "image"
	AICategoryVideo AIProviderCategory = "video"
)

// ValidProviderTypes lists allowed provider types per category.
// The key set also defines the valid categories.
var ValidProviderTypes = map[AIProviderCategory][]string{
	AICategoryLLM:   {"anthropic", "openai", "gemini", "ollama", "claude-code"},
	AICategoryTTS:   {"openai-tts"},
	AICategoryImage: {"gemini-image", "zimage"},
	AICategoryVideo: {},
}

// AIProvider stores credentials and configuration for an AI model provider.
type AIProvider struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Category  AIProviderCategory `json:"category"`
	Type      string             `json:"type"`
	Model     string             `json:"model"`
	APIKey    string             `json:"api_key,omitempty"`
	IsDefault bool               `json:"is_default"`
}

// Validate checks required fields and that category/type are known values.
func (p *AIProvider) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if p.Category == "" {
		return fmt.Errorf("category is required")
	}
	allowed, ok := ValidProviderTypes[p.Category]
	if !ok {
		return fmt.Errorf("invalid category: %s", p.Category)
	}
	if p.Type == "" {
		return fmt.Errorf("type is required")
	}
	if !slices.Contains(allowed, p.Type) {
		return fmt.Errorf("invalid type %q for category %s", p.Type, p.Category)
	}
	return nil
}

// AIProviderSafe is the API-safe view with secrets stripped.
type AIProviderSafe struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Category  AIProviderCategory `json:"category"`
	Type      string             `json:"type"`
	Model     string             `json:"model"`
	IsDefault bool               `json:"is_default"`
}

// Safe returns an AIProviderSafe view with the API key removed.
func (p *AIProvider) Safe() AIProviderSafe {
	return AIProviderSafe{
		ID:        p.ID,
		Name:      p.Name,
		Category:  p.Category,
		Type:      p.Type,
		Model:     p.Model,
		IsDefault: p.IsDefault,
	}
}
