package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/soochol/upal/internal/config"
)

// ModelCategory classifies models into groups with shared configuration options.
type ModelCategory string

const (
	ModelCategoryText  ModelCategory = "text"
	ModelCategoryImage ModelCategory = "image"
	ModelCategoryTTS   ModelCategory = "tts"
)

// OptionSchema describes a single configurable option for a model category.
type OptionSchema struct {
	Key     string         `json:"key"`
	Label   string         `json:"label"`
	Type    string         `json:"type"` // "slider", "number", "select"
	Min     *float64       `json:"min,omitempty"`
	Max     *float64       `json:"max,omitempty"`
	Step    *float64       `json:"step,omitempty"`
	Default any            `json:"default,omitempty"`
	Choices []OptionChoice `json:"choices,omitempty"`
}

// OptionChoice represents a single choice in a select-type option.
type OptionChoice struct {
	Label string `json:"label"`
	Value any    `json:"value"`
}

// ModelTier classifies models by capability level for prompt-based selection.
type ModelTier string

const (
	ModelTierHigh ModelTier = "high"
	ModelTierMid  ModelTier = "mid"
	ModelTierLow  ModelTier = "low"
)

type ModelInfo struct {
	ID       string         `json:"id"`
	Provider string         `json:"provider"`
	Name     string         `json:"name"`
	Category ModelCategory  `json:"category"`
	Tier     ModelTier      `json:"tier,omitempty"`
	Hint     string         `json:"hint,omitempty"` // one-line capability hint for LLM selection
	Options  []OptionSchema `json:"options"`
}

// modelCategoryByType maps provider types to their model category.
var modelCategoryByType = map[string]ModelCategory{
	"anthropic":    ModelCategoryText,
	"gemini":       ModelCategoryText,
	"openai":       ModelCategoryText,
	"claude-code":  ModelCategoryText,
	"gemini-image": ModelCategoryImage,
	"zimage":       ModelCategoryImage,
	"openai-tts":   ModelCategoryTTS,
}

func ptr(f float64) *float64 { return &f }

// categoryOptions defines the configurable options per model category.
var categoryOptions = map[ModelCategory][]OptionSchema{
	ModelCategoryText: {
		{Key: "temperature", Label: "Temperature", Type: "slider", Min: ptr(0), Max: ptr(2), Step: ptr(0.1)},
		{Key: "max_tokens", Label: "Max Tokens", Type: "number", Min: ptr(1), Max: ptr(128000), Step: ptr(1)},
		{Key: "top_p", Label: "Top P", Type: "slider", Min: ptr(0), Max: ptr(1), Step: ptr(0.05)},
	},
	ModelCategoryTTS: {
		{Key: "voice", Label: "Voice", Type: "select", Default: "alloy", Choices: []OptionChoice{
			{Label: "Alloy", Value: "alloy"},
			{Label: "Echo", Value: "echo"},
			{Label: "Fable", Value: "fable"},
			{Label: "Onyx", Value: "onyx"},
			{Label: "Nova", Value: "nova"},
			{Label: "Shimmer", Value: "shimmer"},
		}},
	},
	ModelCategoryImage: {
		{Key: "aspect_ratio", Label: "Aspect Ratio", Type: "select", Default: "1:1", Choices: []OptionChoice{
			{Label: "1:1", Value: "1:1"},
			{Label: "16:9", Value: "16:9"},
			{Label: "9:16", Value: "9:16"},
			{Label: "4:3", Value: "4:3"},
			{Label: "3:4", Value: "3:4"},
			{Label: "3:2", Value: "3:2"},
			{Label: "2:3", Value: "2:3"},
		}},
		{Key: "steps", Label: "Steps", Type: "number", Min: ptr(1), Max: ptr(100), Step: ptr(1), Default: float64(28)},
		{Key: "quality", Label: "Quality", Type: "select", Default: "standard", Choices: []OptionChoice{
			{Label: "Standard", Value: "standard"},
			{Label: "HD", Value: "hd"},
		}},
	},
}

// optionsForType returns the option schema for the given provider type.
func optionsForType(providerType string) (ModelCategory, []OptionSchema) {
	cat, ok := modelCategoryByType[providerType]
	if !ok {
		cat = ModelCategoryText
	}
	return cat, categoryOptions[cat]
}

// modelEntry holds metadata for a known model within a provider type.
type modelEntry struct {
	Name string
	Tier ModelTier
	Hint string // one-line description for LLM model selection
}

// knownModels maps provider type to a curated list of popular models with metadata.
var knownModels = map[string][]modelEntry{
	"gemini": {
		{"gemini-2.5-pro", ModelTierHigh, "high capability, complex reasoning and analysis"},
		{"gemini-2.5-flash", ModelTierMid, "balanced speed/quality, general-purpose tasks"},
		{"gemini-2.0-flash", ModelTierLow, "fast and cheap, simple straightforward tasks"},
		{"gemini-2.0-flash-lite", ModelTierLow, "fastest and cheapest, trivial tasks only"},
	},
	"anthropic": {
		{"claude-opus-4-20250514", ModelTierHigh, "highest capability, complex multi-step reasoning"},
		{"claude-sonnet-4-6", ModelTierHigh, "high capability, strong default for most tasks"},
		{"claude-sonnet-4-20250514", ModelTierMid, "balanced capability, general-purpose tasks"},
		{"claude-haiku-4-20250414", ModelTierLow, "fast and cheap, simple tasks"},
	},
	"openai": {
		{"gpt-4.1", ModelTierHigh, "high capability, complex reasoning tasks"},
		{"gpt-4o", ModelTierMid, "balanced speed/quality, general-purpose tasks"},
		{"gpt-4.1-mini", ModelTierMid, "good capability at lower cost"},
		{"gpt-4o-mini", ModelTierLow, "fast and cheap, simple tasks"},
		{"o3-mini", ModelTierMid, "strong reasoning, math and logic tasks"},
	},
	"claude-code": {
		{"opus", ModelTierHigh, "highest capability, complex multi-step reasoning"},
		{"sonnet", ModelTierMid, "balanced capability, strong default for most tasks"},
		{"haiku", ModelTierLow, "fast and cheap, simple tasks"},
	},
	"gemini-image": {
		{"gemini-2.5-flash-image", ModelTierMid, "image generation"},
		{"gemini-2.0-flash-exp-image-generation", ModelTierMid, "image generation (experimental)"},
		{"gemini-3-pro-image-preview", ModelTierHigh, "image generation (preview)"},
	},
	"zimage": {
		{"z-image", ModelTierMid, "local image generation"},
	},
	"openai-tts": {
		{"tts-1", ModelTierLow, "fast speech synthesis, standard quality"},
		{"tts-1-hd", ModelTierMid, "high-definition speech synthesis"},
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
func KnownModelsGrouped(configs map[string]config.ProviderConfig) []ModelInfo {
	var models []ModelInfo
	for name, pc := range configs {
		cat := modelCategoryByType[pc.Type]
		if known, ok := knownModels[pc.Type]; ok {
			for _, m := range known {
				models = append(models, ModelInfo{
					ID:       name + "/" + m.Name,
					Provider: name,
					Name:     m.Name,
					Category: cat,
					Tier:     m.Tier,
					Hint:     m.Hint,
				})
			}
		}
	}
	return models
}

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	var models []ModelInfo

	for name, pc := range s.providerConfigs {
		cat, opts := optionsForType(pc.Type)

		if isOllama(pc) {
			ollamaModels := discoverOllamaModels(name, pc.URL, cat, opts)
			models = append(models, ollamaModels...)
			continue
		}

		if known, ok := knownModels[pc.Type]; ok {
			for _, m := range known {
				models = append(models, ModelInfo{
					ID:       name + "/" + m.Name,
					Provider: name,
					Name:     m.Name,
					Category: cat,
					Tier:     m.Tier,
					Hint:     m.Hint,
					Options:  opts,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// isOllama detects if a provider config points to a local Ollama instance.
func isOllama(pc config.ProviderConfig) bool {
	return strings.Contains(pc.URL, "11434")
}

// discoverOllamaModels queries the Ollama API to list locally installed models.
func discoverOllamaModels(providerName, baseURL string, cat ModelCategory, opts []OptionSchema) []ModelInfo {
	// Ollama's native API is at /api/tags (not the OpenAI-compat /v1 path)
	apiURL := strings.TrimSuffix(baseURL, "/v1") + "/api/tags"

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}

	var models []ModelInfo
	for _, m := range result.Models {
		// Ollama model names may include ":latest" tag — strip it for cleaner display
		name := strings.TrimSuffix(m.Name, ":latest")
		models = append(models, ModelInfo{
			ID:       providerName + "/" + name,
			Provider: providerName,
			Name:     name,
			Category: cat,
			Options:  opts,
		})
	}
	return models
}
