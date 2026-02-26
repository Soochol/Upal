package upal

// AIProviderCategory classifies AI providers by capability.
type AIProviderCategory string

const (
	AICategoryLLM   AIProviderCategory = "llm"
	AICategoryTTS   AIProviderCategory = "tts"
	AICategoryImage AIProviderCategory = "image"
	AICategoryVideo AIProviderCategory = "video"
)

// AIProvider stores credentials and configuration for an AI model provider.
type AIProvider struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Category  AIProviderCategory `json:"category"`
	Type      string             `json:"type"`
	APIKey    string             `json:"api_key,omitempty"`
	IsDefault bool               `json:"is_default"`
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
