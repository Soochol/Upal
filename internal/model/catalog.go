package model

import (
	"strings"

	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/upal"
)

func ptr(f float64) *float64 { return &f }

// modelCategoryByType maps provider types to their model category.
var modelCategoryByType = map[string]upal.ModelCategory{
	"anthropic":    upal.ModelCategoryText,
	"gemini":       upal.ModelCategoryText,
	"openai":       upal.ModelCategoryText,
	"claude-code":  upal.ModelCategoryText,
	"gemini-image": upal.ModelCategoryImage,
	"zimage":       upal.ModelCategoryImage,
	"openai-tts":   upal.ModelCategoryTTS,
}

// categoryOptions defines the configurable options per model category.
var categoryOptions = map[upal.ModelCategory][]upal.OptionSchema{
	upal.ModelCategoryText: {
		{Key: "temperature", Label: "Temperature", Type: "slider", Min: ptr(0), Max: ptr(2), Step: ptr(0.1)},
		{Key: "max_tokens", Label: "Max Tokens", Type: "number", Min: ptr(1), Max: ptr(128000), Step: ptr(1)},
		{Key: "top_p", Label: "Top P", Type: "slider", Min: ptr(0), Max: ptr(1), Step: ptr(0.05)},
	},
	upal.ModelCategoryTTS: {
		{Key: "voice", Label: "Voice", Type: "select", Default: "alloy", Choices: []upal.OptionChoice{
			{Label: "Alloy", Value: "alloy"},
			{Label: "Echo", Value: "echo"},
			{Label: "Fable", Value: "fable"},
			{Label: "Onyx", Value: "onyx"},
			{Label: "Nova", Value: "nova"},
			{Label: "Shimmer", Value: "shimmer"},
		}},
	},
	upal.ModelCategoryImage: {
		{Key: "aspect_ratio", Label: "Aspect Ratio", Type: "select", Default: "1:1", Choices: []upal.OptionChoice{
			{Label: "1:1", Value: "1:1"},
			{Label: "16:9", Value: "16:9"},
			{Label: "9:16", Value: "9:16"},
			{Label: "4:3", Value: "4:3"},
			{Label: "3:4", Value: "3:4"},
			{Label: "3:2", Value: "3:2"},
			{Label: "2:3", Value: "2:3"},
		}},
		{Key: "steps", Label: "Steps", Type: "number", Min: ptr(1), Max: ptr(100), Step: ptr(1), Default: float64(28)},
		{Key: "quality", Label: "Quality", Type: "select", Default: "standard", Choices: []upal.OptionChoice{
			{Label: "Standard", Value: "standard"},
			{Label: "HD", Value: "hd"},
		}},
	},
}

// OptionsForType returns the category and option schema for the given provider type.
func OptionsForType(providerType string) (upal.ModelCategory, []upal.OptionSchema) {
	cat, ok := modelCategoryByType[providerType]
	if !ok {
		cat = upal.ModelCategoryText
	}
	return cat, categoryOptions[cat]
}

// CategorySupportsTools reports whether models in a category support function calling.
func CategorySupportsTools(cat upal.ModelCategory) bool {
	return cat == upal.ModelCategoryText
}

// modelEntry holds metadata for a known model within a provider type.
type modelEntry struct {
	Name string
	Tier upal.ModelTier
	Hint string // one-line description for LLM model selection
}

// knownModels maps provider type to a curated list of popular models with metadata.
var knownModels = map[string][]modelEntry{
	"gemini": {
		{"gemini-3.1-pro-preview", upal.ModelTierHigh, "latest high capability, complex reasoning and analysis"},
		{"gemini-3-flash-preview", upal.ModelTierMid, "latest balanced speed and quality, general-purpose tasks"},
	},
	"anthropic": {
		{"claude-opus-4-20250514", upal.ModelTierHigh, "highest capability, complex multi-step reasoning"},
		{"claude-sonnet-4-6", upal.ModelTierHigh, "high capability, strong default for most tasks"},
		{"claude-sonnet-4-20250514", upal.ModelTierMid, "balanced capability, general-purpose tasks"},
		{"claude-haiku-4-20250414", upal.ModelTierLow, "fast and cheap, simple tasks"},
	},
	"openai": {
		{"gpt-4.1", upal.ModelTierHigh, "high capability, complex reasoning tasks"},
		{"gpt-4o", upal.ModelTierMid, "balanced speed/quality, general-purpose tasks"},
		{"gpt-4.1-mini", upal.ModelTierMid, "good capability at lower cost"},
		{"gpt-4o-mini", upal.ModelTierLow, "fast and cheap, simple tasks"},
		{"o3-mini", upal.ModelTierMid, "strong reasoning, math and logic tasks"},
	},
	"claude-code": {
		{"opus", upal.ModelTierHigh, "highest capability, complex multi-step reasoning"},
		{"sonnet", upal.ModelTierMid, "balanced capability, strong default for most tasks"},
		{"haiku", upal.ModelTierLow, "fast and cheap, simple tasks"},
	},
	"gemini-image": {
		{"gemini-2.5-flash-image", upal.ModelTierMid, "image generation"},
		{"gemini-2.0-flash-exp-image-generation", upal.ModelTierMid, "image generation (experimental)"},
		{"gemini-3-pro-image-preview", upal.ModelTierHigh, "image generation (preview)"},
	},
	"zimage": {
		{"z-image", upal.ModelTierMid, "local image generation"},
	},
	"openai-tts": {
		{"tts-1", upal.ModelTierLow, "fast speech synthesis, standard quality"},
		{"tts-1-hd", upal.ModelTierMid, "high-definition speech synthesis"},
	},
}

// KnownModelIDs returns the list of available model IDs (provider/model format)
// derived from provider configs. Used to inject into LLM prompts so the generator
// and configurator can select from actually available models.
func KnownModelIDs(configs map[string]config.ProviderConfig) []string {
	var ids []string
	for name, pc := range configs {
		if known, ok := knownModels[pc.Type]; ok {
			for _, m := range known {
				ids = append(ids, name+"/"+m.Name)
			}
		}
	}
	return ids
}

// KnownModelsGrouped returns all ModelInfo entries with category/tier metadata.
// Used by the generator to inject categorized model guidance into prompts.
func KnownModelsGrouped(configs map[string]config.ProviderConfig) []upal.ModelInfo {
	var models []upal.ModelInfo
	for name, pc := range configs {
		cat := modelCategoryByType[pc.Type]
		if known, ok := knownModels[pc.Type]; ok {
			for _, m := range known {
				models = append(models, upal.ModelInfo{
					ID:            name + "/" + m.Name,
					Provider:      name,
					Name:          m.Name,
					Category:      cat,
					Tier:          m.Tier,
					Hint:          m.Hint,
					SupportsTools: CategorySupportsTools(cat),
				})
			}
		}
	}
	return models
}

// AllStaticModels returns the full list of statically known models with options populated.
func AllStaticModels(configs map[string]config.ProviderConfig) []upal.ModelInfo {
	var models []upal.ModelInfo
	for name, pc := range configs {
		cat, opts := OptionsForType(pc.Type)
		if known, ok := knownModels[pc.Type]; ok {
			for _, m := range known {
				models = append(models, upal.ModelInfo{
					ID:            name + "/" + m.Name,
					Provider:      name,
					Name:          m.Name,
					Category:      cat,
					Tier:          m.Tier,
					Hint:          m.Hint,
					Options:       opts,
					SupportsTools: CategorySupportsTools(cat),
				})
			}
		}
	}
	return models
}

// FirstModelForType returns the first known model name for the given provider type.
func FirstModelForType(providerType string) (string, bool) {
	if models, ok := knownModels[providerType]; ok && len(models) > 0 {
		return models[0].Name, true
	}
	return "", false
}

// DefaultURLForType returns the default API URL for a provider type.
func DefaultURLForType(providerType string) string {
	switch providerType {
	case "ollama":
		return "http://localhost:11434"
	default:
		return ""
	}
}

// IsOllama detects if a provider config points to a local Ollama instance.
// It returns true if the type is explicitly "ollama", or if the type is "openai"
// and the URL contains the default Ollama port (11434) for backward compatibility.
func IsOllama(pc config.ProviderConfig) bool {
	if pc.Type == "ollama" {
		return true
	}
	return pc.Type == "openai" && strings.Contains(pc.URL, "11434")
}
