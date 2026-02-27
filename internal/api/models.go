package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/soochol/upal/internal/config"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
)

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var models []upal.ModelInfo
	configs := s.effectiveProviderConfigs(ctx)

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

	// Mark models from default providers.
	defaultProviders := s.defaultProviderNames(ctx)
	for i := range models {
		if defaultProviders[models[i].Provider] {
			models[i].IsDefault = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orEmpty(models))
}

// defaultProviderNames returns a set of provider names that are marked as default in DB.
func (s *Server) defaultProviderNames(ctx context.Context) map[string]bool {
	if s.aiProviderSvc == nil {
		return nil
	}
	providers, err := s.aiProviderSvc.List(ctx)
	if err != nil {
		return nil
	}
	defaults := make(map[string]bool)
	for _, p := range providers {
		if p.IsDefault {
			defaults[p.Name] = true
		}
	}
	return defaults
}

// effectiveProviderConfigs returns provider configs from DB if available, otherwise from config.yaml.
func (s *Server) effectiveProviderConfigs(ctx context.Context) map[string]config.ProviderConfig {
	if s.aiProviderSvc == nil {
		return s.providerConfigs
	}
	providers, err := s.aiProviderSvc.ListAll(ctx)
	if err != nil {
		slog.Warn("effectiveProviderConfigs: DB unavailable, falling back to static config", "err", err)
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
