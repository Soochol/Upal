package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// PublishChannelRepository stores and retrieves publish channels.
type PublishChannelRepository interface {
	Create(ctx context.Context, ch *upal.PublishChannel) error
	Get(ctx context.Context, id string) (*upal.PublishChannel, error)
	List(ctx context.Context) ([]*upal.PublishChannel, error)
	Update(ctx context.Context, ch *upal.PublishChannel) error
	Delete(ctx context.Context, id string) error
}
