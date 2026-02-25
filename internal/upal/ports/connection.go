package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// ConnectionPort is the interface for managing external service connections.
// Services should depend on this interface rather than *ConnectionService directly.
type ConnectionPort interface {
	Create(ctx context.Context, conn *upal.Connection) error
	Get(ctx context.Context, id string) (*upal.Connection, error)
	Resolve(ctx context.Context, id string) (*upal.Connection, error)
	List(ctx context.Context) ([]upal.ConnectionSafe, error)
	Update(ctx context.Context, conn *upal.Connection) error
	Delete(ctx context.Context, id string) error
}
