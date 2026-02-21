// internal/services/stage_passthrough.go
package services

import (
	"context"

	"github.com/soochol/upal/internal/upal"
)

// PassthroughStageExecutor is a no-op executor for declarative stages (schedule, trigger)
// that configure when a pipeline runs but require no processing during execution.
type PassthroughStageExecutor struct {
	stageType string
}

func NewPassthroughStageExecutor(stageType string) *PassthroughStageExecutor {
	return &PassthroughStageExecutor{stageType: stageType}
}

func (e *PassthroughStageExecutor) Type() string { return e.stageType }

func (e *PassthroughStageExecutor) Execute(_ context.Context, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	output := make(map[string]any)
	if prevResult != nil {
		for k, v := range prevResult.Output {
			output[k] = v
		}
	}
	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output:  output,
	}, nil
}
