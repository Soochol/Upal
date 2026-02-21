package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// MemoryConnectionRepository is a thread-safe in-memory connection store.
type MemoryConnectionRepository struct {
	store *memstore.Store[*upal.Connection]
}

func NewMemoryConnectionRepository() *MemoryConnectionRepository {
	return &MemoryConnectionRepository{
		store: memstore.New(func(c *upal.Connection) string { return c.ID }),
	}
}

func (r *MemoryConnectionRepository) Create(ctx context.Context, conn *upal.Connection) error {
	if r.store.Has(ctx, conn.ID) {
		return fmt.Errorf("connection %q already exists", conn.ID)
	}
	return r.store.Set(ctx, conn)
}

func (r *MemoryConnectionRepository) Get(ctx context.Context, id string) (*upal.Connection, error) {
	c, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("connection %q not found", id)
	}
	return c, err
}

func (r *MemoryConnectionRepository) List(ctx context.Context) ([]*upal.Connection, error) {
	return r.store.All(ctx)
}

func (r *MemoryConnectionRepository) Update(ctx context.Context, conn *upal.Connection) error {
	if !r.store.Has(ctx, conn.ID) {
		return fmt.Errorf("connection %q not found", conn.ID)
	}
	return r.store.Set(ctx, conn)
}

func (r *MemoryConnectionRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("connection %q not found", id)
	}
	return err
}
