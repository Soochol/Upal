package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/soochol/upal/internal/config"
)

type ModelInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
}

// knownModels maps provider type to a curated list of popular model names.
var knownModels = map[string][]string{
	"gemini": {
		"gemini-2.5-flash",
		"gemini-2.5-pro",
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
	},
	"anthropic": {
		"claude-sonnet-4-20250514",
		"claude-haiku-4-20250414",
		"claude-opus-4-20250514",
	},
	"openai": {
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4.1",
		"gpt-4.1-mini",
		"o3-mini",
	},
}

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	var models []ModelInfo

	for name, pc := range s.providerConfigs {
		if isOllama(pc) {
			ollamaModels := discoverOllamaModels(name, pc.URL)
			models = append(models, ollamaModels...)
			continue
		}

		if known, ok := knownModels[pc.Type]; ok {
			for _, m := range known {
				models = append(models, ModelInfo{
					ID:       name + "/" + m,
					Provider: name,
					Name:     m,
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
func discoverOllamaModels(providerName, baseURL string) []ModelInfo {
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
		})
	}
	return models
}
