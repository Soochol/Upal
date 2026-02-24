package llmutil

import (
	"fmt"
	"strings"

	adkmodel "google.golang.org/adk/model"
)

// MapResolver implements ports.LLMResolver using a map of provider name → LLM.
type MapResolver struct {
	llms         map[string]adkmodel.LLM
	defaultLLM   adkmodel.LLM
	defaultModel string
}

// NewMapResolver creates a resolver. llms keys are provider names (e.g. "anthropic").
func NewMapResolver(llms map[string]adkmodel.LLM, defaultLLM adkmodel.LLM, defaultModel string) *MapResolver {
	return &MapResolver{llms: llms, defaultLLM: defaultLLM, defaultModel: defaultModel}
}

// Resolve parses "provider/model" and returns the matching LLM + model name.
// Empty modelID returns the system default.
func (r *MapResolver) Resolve(modelID string) (adkmodel.LLM, string, error) {
	if modelID == "" {
		return r.defaultLLM, r.defaultModel, nil
	}
	provider, modelName, ok := strings.Cut(modelID, "/")
	if !ok {
		return nil, "", fmt.Errorf("invalid model ID %q: expected provider/model format", modelID)
	}
	llm, found := r.llms[provider]
	if !found {
		return nil, "", fmt.Errorf("unknown provider %q in model ID %q", provider, modelID)
	}
	return llm, modelName, nil
}
