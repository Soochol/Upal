package ports

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// RetryExecutor wraps workflow execution with configurable retry and backoff.
type RetryExecutor interface {
	ExecuteWithRetry(
		ctx context.Context,
		wf *upal.WorkflowDefinition,
		inputs map[string]any,
		policy upal.RetryPolicy,
		triggerType, triggerRef string,
	) (<-chan upal.WorkflowEvent, <-chan upal.RunResult, error)
}

// ConcurrencyControl limits concurrent workflow executions per workflow.
type ConcurrencyControl interface {
	Acquire(ctx context.Context, workflowName string) error
	Release(workflowName string)
}

// PipelineRunner starts and resumes pipeline stage execution.
type PipelineRunner interface {
	Start(ctx context.Context, pipeline *upal.Pipeline, inputs map[string]any) (*upal.PipelineRun, error)
	Resume(ctx context.Context, pipeline *upal.Pipeline, run *upal.PipelineRun) error
}

// PipelineRegistry provides read-only pipeline lookup (subset of PipelineServicePort).
type PipelineRegistry interface {
	Get(ctx context.Context, id string) (*upal.Pipeline, error)
}

// SchedulerPort defines the contract for pipeline-schedule synchronization.
type SchedulerPort interface {
	SyncPipelineSchedules(ctx context.Context, pipeline *upal.Pipeline) error
	RemovePipelineSchedules(ctx context.Context, pipelineID string) error
	AddSchedule(ctx context.Context, schedule *upal.Schedule) error
	RemoveSchedule(ctx context.Context, id string) error
}
