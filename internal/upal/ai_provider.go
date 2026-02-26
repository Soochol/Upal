package upal

import "fmt"

// AIProviderCategory classifies AI providers by capability.
type AIProviderCategory string

const (
	AICategoryLLM   AIProviderCategory = "llm"
	AICategoryTTS   AIProviderCategory = "tts"
	AICategoryImage AIProviderCategory = "image"
	AICategoryVideo AIProviderCategory = "video"
)

// ValidAICategories is the set of allowed category values.
var ValidAICategories = map[AIProviderCategory]bool{
	AICategoryLLM:   true,
	AICategoryTTS:   true,
	AICategoryImage: true,
	AICategoryVideo: true,
}

// ValidProviderTypes lists allowed provider types per category.
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
	if !ValidAICategories[p.Category] {
		return fmt.Errorf("invalid category: %s", p.Category)
	}
	if p.Type == "" {
		return fmt.Errorf("type is required")
	}
	valid := false
	for _, t := range ValidProviderTypes[p.Category] {
		if t == p.Type {
			valid = true
			break
		}
	}
	if !valid {
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
	IsDefault bool               `json:"is_default"`
}

// Safe returns an AIProviderSafe view with the API key removed.
func (p *AIProvider) Safe() AIProviderSafe {
	return AIProviderSafe{
		ID:        p.ID,
		Name:      p.Name,
		Category:  p.Category,
		Type:      p.Type,
		IsDefault: p.IsDefault,
	}
}
