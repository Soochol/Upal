package model

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// DiscoverOllamaModels queries the Ollama API to list locally installed models.
func DiscoverOllamaModels(providerName, baseURL string, cat upal.ModelCategory, opts []upal.OptionSchema) []upal.ModelInfo {
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

	var models []upal.ModelInfo
	for _, m := range result.Models {
		// Ollama model names may include ":latest" tag -- strip it for cleaner display
		name := strings.TrimSuffix(m.Name, ":latest")
		models = append(models, upal.ModelInfo{
			ID:            providerName + "/" + name,
			Provider:      providerName,
			Name:          name,
			Category:      cat,
			Options:       opts,
			SupportsTools: CategorySupportsTools(cat),
		})
	}
	return models
}
