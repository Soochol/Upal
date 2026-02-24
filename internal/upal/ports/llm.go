package ports

import adkmodel "google.golang.org/adk/model"

// LLMResolver resolves a "provider/model" ID string to an LLM instance
// and the model name to use in LLMRequest. Empty modelID returns the
// system default.
type LLMResolver interface {
	Resolve(modelID string) (adkmodel.LLM, string, error)
}
