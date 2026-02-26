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
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateAIProvider(ctx, userID, p); err != nil {
		return err
	}
	_ = r.mem.Create(ctx, p)
	return nil
}

func (r *PersistentAIProviderRepository) Get(ctx context.Context, id string) (*upal.AIProvider, error) {
	if p, err := r.mem.Get(ctx, id); err == nil {
		return p, nil
	}
	userID := upal.UserIDFromContext(ctx)
	p, err := r.db.GetAIProvider(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, p)
	return p, nil
}

func (r *PersistentAIProviderRepository) List(ctx context.Context) ([]*upal.AIProvider, error) {
	userID := upal.UserIDFromContext(ctx)
	providers, err := r.db.ListAIProviders(ctx, userID)
	if err == nil {
		return providers, nil
	}
	slog.Warn("db list ai providers failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentAIProviderRepository) Update(ctx context.Context, p *upal.AIProvider) error {
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateAIProvider(ctx, userID, p); err != nil {
		return err
	}
	_ = r.mem.Update(ctx, p)
	return nil
}

func (r *PersistentAIProviderRepository) Delete(ctx context.Context, id string) error {
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteAIProvider(ctx, userID, id); err != nil {
		return err
	}
	_ = r.mem.Delete(ctx, id)
	return nil
}

func (r *PersistentAIProviderRepository) ClearDefault(ctx context.Context, category upal.AIProviderCategory) error {
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.ClearAIProviderDefault(ctx, userID, string(category)); err != nil {
		return err
	}
	_ = r.mem.ClearDefault(ctx, category)
	return nil
}
