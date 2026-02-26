package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/soochol/upal/internal/upal"
)

// SessionDB defines the database methods used by persistent session repositories.
type SessionDB interface {
	CreateSession(ctx context.Context, userID string, s *upal.Session) error
	GetSession(ctx context.Context, userID string, id string) (*upal.Session, error)
	ListSessions(ctx context.Context, userID string) ([]*upal.Session, error)
	UpdateSession(ctx context.Context, userID string, s *upal.Session) error
	DeleteSession(ctx context.Context, userID string, id string) error
}

type PersistentSessionRepository struct {
	mem *MemorySessionRepository
	db  SessionDB
}

func NewPersistentSessionRepository(mem *MemorySessionRepository, db SessionDB) *PersistentSessionRepository {
	return &PersistentSessionRepository{mem: mem, db: db}
}

func (r *PersistentSessionRepository) Create(ctx context.Context, s *upal.Session) error {
	_ = r.mem.Create(ctx, s)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.CreateSession(ctx, userID, s); err != nil {
		return fmt.Errorf("db create session: %w", err)
	}
	return nil
}

func (r *PersistentSessionRepository) Get(ctx context.Context, id string) (*upal.Session, error) {
	if s, err := r.mem.Get(ctx, id); err == nil {
		return s, nil
	}
	userID := upal.UserIDFromContext(ctx)
	s, err := r.db.GetSession(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, s)
	return s, nil
}

func (r *PersistentSessionRepository) List(ctx context.Context) ([]*upal.Session, error) {
	userID := upal.UserIDFromContext(ctx)
	sessions, err := r.db.ListSessions(ctx, userID)
	if err == nil {
		return sessions, nil
	}
	slog.Warn("db list sessions failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentSessionRepository) Update(ctx context.Context, s *upal.Session) error {
	_ = r.mem.Update(ctx, s)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.UpdateSession(ctx, userID, s); err != nil {
		return fmt.Errorf("db update session: %w", err)
	}
	return nil
}

func (r *PersistentSessionRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	userID := upal.UserIDFromContext(ctx)
	if err := r.db.DeleteSession(ctx, userID, id); err != nil {
		return fmt.Errorf("db delete session: %w", err)
	}
	return nil
}
