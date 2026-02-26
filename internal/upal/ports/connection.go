package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// ConnectionPort defines the contract for managing external service connections.
type ConnectionPort interface {
	Create(ctx context.Context, conn *upal.Connection) error
	Get(ctx context.Context, id string) (*upal.Connection, error)
	Resolve(ctx context.Context, id string) (*upal.Connection, error)
	List(ctx context.Context) ([]upal.ConnectionSafe, error)
	Update(ctx context.Context, conn *upal.Connection) error
	Delete(ctx context.Context, id string) error
}
