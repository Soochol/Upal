package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type ConnectionRepository interface {
	Create(ctx context.Context, conn *upal.Connection) error
	Get(ctx context.Context, id string) (*upal.Connection, error)
	List(ctx context.Context) ([]*upal.Connection, error)
	Update(ctx context.Context, conn *upal.Connection) error
	Delete(ctx context.Context, id string) error
}
