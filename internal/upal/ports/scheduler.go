package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// RetryExecutor runs a workflow with configurable retry and backoff.
type RetryExecutor interface {
	ExecuteWithRetry(
		ctx context.Context,
		wf *upal.WorkflowDefinition,
		inputs map[string]any,
		policy upal.RetryPolicy,
		triggerType, triggerRef string,
	) (<-chan upal.WorkflowEvent, <-chan upal.RunResult, error)
}

// ConcurrencyControl limits concurrent workflow executions per-workflow.
type ConcurrencyControl interface {
	Acquire(ctx context.Context, workflowName string) error
	Release(workflowName string)
}

// PipelineRunner starts and resumes pipeline executions.
type PipelineRunner interface {
	Start(ctx context.Context, pipeline *upal.Pipeline) (*upal.PipelineRun, error)
	Resume(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun) error
}

// PipelineRegistry looks up pipeline definitions by ID.
type PipelineRegistry interface {
	Get(ctx context.Context, id string) (*upal.Pipeline, error)
}

// SchedulerPort is the interface for pipeline-schedule synchronization.
// The api layer depends on this rather than *scheduler.SchedulerService directly.
type SchedulerPort interface {
	SyncPipelineSchedules(ctx context.Context, pipeline *upal.Pipeline) error
	RemovePipelineSchedules(ctx context.Context, pipelineID string) error
}
