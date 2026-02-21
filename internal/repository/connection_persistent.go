package repository

import (
	"context"
	"log/slog"

	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/upal"
)

// PersistentConnectionRepository wraps MemoryConnectionRepository with PostgreSQL.
type PersistentConnectionRepository struct {
	mem *MemoryConnectionRepository
	db  *db.DB
}

func NewPersistentConnectionRepository(mem *MemoryConnectionRepository, database *db.DB) *PersistentConnectionRepository {
	return &PersistentConnectionRepository{mem: mem, db: database}
}

func (r *PersistentConnectionRepository) Create(ctx context.Context, conn *upal.Connection) error {
	_ = r.mem.Create(ctx, conn)
	if err := r.db.CreateConnection(ctx, conn); err != nil {
		slog.Warn("db create connection failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentConnectionRepository) Get(ctx context.Context, id string) (*upal.Connection, error) {
	if c, err := r.mem.Get(ctx, id); err == nil {
		return c, nil
	}
	c, err := r.db.GetConnection(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = r.mem.Create(ctx, c)
	return c, nil
}

func (r *PersistentConnectionRepository) List(ctx context.Context) ([]*upal.Connection, error) {
	conns, err := r.db.ListConnections(ctx)
	if err == nil {
		return conns, nil
	}
	slog.Warn("db list connections failed, falling back to in-memory", "err", err)
	return r.mem.List(ctx)
}

func (r *PersistentConnectionRepository) Update(ctx context.Context, conn *upal.Connection) error {
	_ = r.mem.Update(ctx, conn)
	if err := r.db.UpdateConnection(ctx, conn); err != nil {
		slog.Warn("db update connection failed, in-memory only", "err", err)
	}
	return nil
}

func (r *PersistentConnectionRepository) Delete(ctx context.Context, id string) error {
	_ = r.mem.Delete(ctx, id)
	if err := r.db.DeleteConnection(ctx, id); err != nil {
		slog.Warn("db delete connection failed", "err", err)
	}
	return nil
}
