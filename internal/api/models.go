package api

import (
	"encoding/json"
	"net/http"

	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	var models []upal.ModelInfo

	// Collect Ollama providers separately for dynamic discovery.
	staticConfigs := make(map[string]struct{})
	for name, pc := range s.providerConfigs {
		if upalmodel.IsOllama(pc) {
			cat, opts := upalmodel.OptionsForType(pc.Type)
			ollamaModels := upalmodel.DiscoverOllamaModels(name, pc.URL, cat, opts)
			models = append(models, ollamaModels...)
		} else {
			staticConfigs[name] = struct{}{}
		}
	}

	// Add statically known models for non-Ollama providers.
	for _, m := range upalmodel.AllStaticModels(s.providerConfigs) {
		if _, ok := staticConfigs[m.Provider]; ok {
			models = append(models, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
