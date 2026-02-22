package model

import (
	"github.com/soochol/upal/internal/config"
	adkmodel "google.golang.org/adk/model"
)

// LLMFactory creates an adkmodel.LLM for a given provider name and config.
type LLMFactory func(providerName string, cfg config.ProviderConfig) adkmodel.LLM

var factories = map[string]LLMFactory{}

// RegisterProvider registers a factory for the given provider type string.
// Called from init() in each model implementation file.
func RegisterProvider(typeName string, factory LLMFactory) {
	factories[typeName] = factory
}

// BuildLLM looks up a registered factory for cfg.Type and calls it.
// If no factory is found but cfg.URL is set, falls back to OpenAI-compat.
// Returns (nil, false) if the type is unknown and no URL fallback is available.
func BuildLLM(providerName string, cfg config.ProviderConfig) (adkmodel.LLM, bool) {
	if factory, ok := factories[cfg.Type]; ok {
		return factory(providerName, cfg), true
	}
	// Fallback: any provider with a URL is treated as OpenAI-compatible.
	if cfg.URL != "" {
		return NewOpenAILLM(cfg.APIKey,
			WithOpenAIBaseURL(cfg.URL),
			WithOpenAIName(providerName)), true
	}
	return nil, false
}
