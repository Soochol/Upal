package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

type MemoryAIProviderRepository struct {
	store *memstore.Store[*upal.AIProvider]
}

func NewMemoryAIProviderRepository() *MemoryAIProviderRepository {
	return &MemoryAIProviderRepository{
		store: memstore.New(func(p *upal.AIProvider) string { return p.ID }),
	}
}

func (r *MemoryAIProviderRepository) Create(ctx context.Context, p *upal.AIProvider) error {
	if r.store.Has(ctx, p.ID) {
		return fmt.Errorf("ai provider %q already exists", p.ID)
	}
	return r.store.Set(ctx, p)
}

func (r *MemoryAIProviderRepository) Get(ctx context.Context, id string) (*upal.AIProvider, error) {
	p, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("ai provider %q: %w", id, ErrNotFound)
	}
	return p, err
}

func (r *MemoryAIProviderRepository) List(ctx context.Context) ([]*upal.AIProvider, error) {
	return r.store.All(ctx)
}

func (r *MemoryAIProviderRepository) Update(ctx context.Context, p *upal.AIProvider) error {
	if !r.store.Has(ctx, p.ID) {
		return fmt.Errorf("ai provider %q: %w", p.ID, ErrNotFound)
	}
	return r.store.Set(ctx, p)
}

func (r *MemoryAIProviderRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("ai provider %q: %w", id, ErrNotFound)
	}
	return err
}

func (r *MemoryAIProviderRepository) ClearDefault(ctx context.Context, category upal.AIProviderCategory) error {
	items, err := r.store.All(ctx)
	if err != nil {
		return err
	}
	for _, p := range items {
		if p.Category == category && p.IsDefault {
			p.IsDefault = false
			_ = r.store.Set(ctx, p)
		}
	}
	return nil
}
