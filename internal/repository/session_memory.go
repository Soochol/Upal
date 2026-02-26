package repository

import (
	"context"
	"errors"
	"fmt"

	memstore "github.com/soochol/upal/internal/repository/memory"
	"github.com/soochol/upal/internal/upal"
)

// MemorySessionRepository is an in-memory implementation of SessionRepository.
type MemorySessionRepository struct {
	store *memstore.Store[*upal.Session]
}

// NewMemorySessionRepository creates a new in-memory session repository.
func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		store: memstore.New(func(s *upal.Session) string { return s.ID }),
	}
}

func (r *MemorySessionRepository) Create(ctx context.Context, s *upal.Session) error {
	if r.store.Has(ctx, s.ID) {
		return fmt.Errorf("session %q already exists", s.ID)
	}
	return r.store.Set(ctx, s)
}

func (r *MemorySessionRepository) Get(ctx context.Context, id string) (*upal.Session, error) {
	s, err := r.store.Get(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return nil, fmt.Errorf("session %q: %w", id, ErrNotFound)
	}
	return s, err
}

func (r *MemorySessionRepository) List(ctx context.Context) ([]*upal.Session, error) {
	return r.store.All(ctx)
}

func (r *MemorySessionRepository) Update(ctx context.Context, s *upal.Session) error {
	if !r.store.Has(ctx, s.ID) {
		return fmt.Errorf("session %q: %w", s.ID, ErrNotFound)
	}
	return r.store.Set(ctx, s)
}

func (r *MemorySessionRepository) Delete(ctx context.Context, id string) error {
	err := r.store.Delete(ctx, id)
	if errors.Is(err, memstore.ErrNotFound) {
		return fmt.Errorf("session %q: %w", id, ErrNotFound)
	}
	return err
}
