package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

type MemoryPublishChannelRepository struct {
	store *memstore.Store[*upal.PublishChannel]
}

func NewMemoryPublishChannelRepository() *MemoryPublishChannelRepository {
	return &MemoryPublishChannelRepository{
		store: memstore.New(func(ch *upal.PublishChannel) string { return ch.ID }),
	}
}

func (r *MemoryPublishChannelRepository) Create(ctx context.Context, ch *upal.PublishChannel) error {
	if r.store.Has(ctx, ch.ID) {
		return fmt.Errorf("publish channel %q already exists", ch.ID)
	}
	return r.store.Set(ctx, ch)
}

func (r *MemoryPublishChannelRepository) Get(ctx context.Context, id string) (*upal.PublishChannel, error) {
	ch, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("publish channel %q: %w", id, ErrNotFound)
	}
	return ch, err
}

func (r *MemoryPublishChannelRepository) List(ctx context.Context) ([]*upal.PublishChannel, error) {
	return r.store.All(ctx)
}

func (r *MemoryPublishChannelRepository) Update(ctx context.Context, ch *upal.PublishChannel) error {
	if !r.store.Has(ctx, ch.ID) {
		return fmt.Errorf("publish channel %q: %w", ch.ID, ErrNotFound)
	}
	return r.store.Set(ctx, ch)
}

func (r *MemoryPublishChannelRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("publish channel %q: %w", id, ErrNotFound)
	}
	return err
}
