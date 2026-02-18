package provider

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *Registry) Resolve(modelID string) (Provider, string, error) {
	providerName, modelName, err := ParseModelID(modelID)
	if err != nil {
		return nil, "", err
	}
	p, ok := r.Get(providerName)
	if !ok {
		return nil, "", fmt.Errorf("unknown provider: %q", providerName)
	}
	return p, modelName, nil
}
