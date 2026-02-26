package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/soochol/upal/internal/config"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	var models []upal.ModelInfo
	configs := s.effectiveProviderConfigs(r.Context())

	staticConfigs := make(map[string]struct{})
	for name, pc := range configs {
		if upalmodel.IsOllama(pc) {
			cat, opts := upalmodel.OptionsForType(pc.Type)
			ollamaModels := upalmodel.DiscoverOllamaModels(name, pc.URL, cat, opts)
			models = append(models, ollamaModels...)
		} else {
			staticConfigs[name] = struct{}{}
		}
	}

	for _, m := range upalmodel.AllStaticModels(configs) {
		if _, ok := staticConfigs[m.Provider]; ok {
			models = append(models, m)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

// effectiveProviderConfigs returns provider configs from DB if available, otherwise from config.yaml.
func (s *Server) effectiveProviderConfigs(ctx context.Context) map[string]config.ProviderConfig {
	if s.aiProviderSvc == nil {
		return s.providerConfigs
	}
	providers, err := s.aiProviderSvc.ListAll(ctx)
	if err != nil {
		return s.providerConfigs
	}
	// When DB is available, only show DB-registered providers (no config.yaml fallback).
	configs := make(map[string]config.ProviderConfig, len(providers))
	for _, p := range providers {
		configs[p.Name] = config.ProviderConfig{
			Type:   p.Type,
			APIKey: p.APIKey,
			URL:    upalmodel.DefaultURLForType(p.Type),
		}
	}
	return configs
}
