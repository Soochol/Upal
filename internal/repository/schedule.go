package repository

import (
	"context"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// ScheduleRepository abstracts persistence for cron schedules.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *upal.Schedule) error
	Get(ctx context.Context, id string) (*upal.Schedule, error)
	Update(ctx context.Context, schedule *upal.Schedule) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*upal.Schedule, error)
	ListDue(ctx context.Context, now time.Time) ([]*upal.Schedule, error)
	ListByPipeline(ctx context.Context, pipelineID string) ([]*upal.Schedule, error)
}
