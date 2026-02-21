package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// RunHistoryPort is the interface for recording and querying workflow run history.
// Services should depend on this interface rather than *RunHistoryService directly.
type RunHistoryPort interface {
	StartRun(ctx context.Context, workflowName, triggerType, triggerRef string, inputs map[string]any) (*upal.RunRecord, error)
	CompleteRun(ctx context.Context, id string, outputs map[string]any) error
	FailRun(ctx context.Context, id string, errMsg string) error
	UpdateRunRetryMeta(ctx context.Context, id string, retryCount int, retryOf *string) error
	UpdateNodeRun(ctx context.Context, runID string, nodeRun upal.NodeRunRecord) error
	GetRun(ctx context.Context, id string) (*upal.RunRecord, error)
	ListRuns(ctx context.Context, workflowName string, limit, offset int) ([]*upal.RunRecord, int, error)
	ListAllRuns(ctx context.Context, limit, offset int, status string) ([]*upal.RunRecord, int, error)
}
