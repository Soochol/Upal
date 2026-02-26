package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// SessionRepository defines the data access interface for Session entities.
type SessionRepository interface {
	Create(ctx context.Context, s *upal.Session) error
	Get(ctx context.Context, id string) (*upal.Session, error)
	List(ctx context.Context) ([]*upal.Session, error)
	Update(ctx context.Context, s *upal.Session) error
	Delete(ctx context.Context, id string) error
}
