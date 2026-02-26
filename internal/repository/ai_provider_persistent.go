package repository

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

type PersistentAIProviderRepository struct {
	mem *MemoryAIProviderRepository
	db  *db.DB
}

func NewPersistentAIProviderRepository(mem *MemoryAIProviderRepository, database *db.DB) *PersistentAIProviderRepository {
	return &PersistentAIProviderRepository{mem: mem, db: database}
}

func (r *PersistentAIProviderRepository) Create(ctx context.Context, p *upal.AIProvider) error {
	_ = r.mem.Create(ctx, p)
	if err := r.db.CreateAIProvider(ctx, p); err != nil {
		slog.Warn("db create ai provider failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentAIProviderRepository) Get(ctx context.Context, id string) (*upal.AIProvider, error) {
	if p, err := r.mem.Get(ctx, id); err == nil {
		return p, nil
	}
	p, err := r.db.GetAIProvider(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, p)
	return p, nil
}

func (r *PersistentAIProviderRepository) List(ctx context.Context) ([]*upal.AIProvider, error) {
	providers, err := r.db.ListAIProviders(ctx)
	if err == nil {
		return providers, nil
	}
	slog.Warn("db list ai providers failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentAIProviderRepository) Update(ctx context.Context, p *upal.AIProvider) error {
	_ = r.mem.Update(ctx, p)
	if err := r.db.UpdateAIProvider(ctx, p); err != nil {
		slog.Warn("db update ai provider failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentAIProviderRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeleteAIProvider(ctx, id); err != nil {
		slog.Warn("db delete ai provider failed", "err", err)
	}
	return nil
}

func (r *PersistentAIProviderRepository) ClearDefault(ctx context.Context, category upal.AIProviderCategory) error {
	_ = r.mem.ClearDefault(ctx, category)
	if err := r.db.ClearAIProviderDefault(ctx, string(category)); err != nil {
		slog.Warn("db clear ai provider default failed", "err", err)
	}
	return nil
}
