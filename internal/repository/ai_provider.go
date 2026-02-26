package repository

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

type AIProviderRepository interface {
	Create(ctx context.Context, p *upal.AIProvider) error
	Get(ctx context.Context, id string) (*upal.AIProvider, error)
	List(ctx context.Context) ([]*upal.AIProvider, error)
	Update(ctx context.Context, p *upal.AIProvider) error
	Delete(ctx context.Context, id string) error
	ClearDefault(ctx context.Context, category upal.AIProviderCategory) error
}
