// internal/services/stage_transform.go
package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/soochol/upal/internal/upal"
)

type TransformStageExecutor struct{}

func (e *TransformStageExecutor) Type() string { return "transform" }

func (e *TransformStageExecutor) Execute(_ context.Context, _ *upal.Pipeline, stage upal.Stage, prevResult *upal.StageResult) (*upal.StageResult, error) {
	output := make(map[string]any)

	if prevResult != nil {
		if stage.Config.InputMapping != nil {
			for destKey, srcKey := range stage.Config.InputMapping {
				if val, ok := prevResult.Output[srcKey]; ok {
					output[destKey] = val
				}
			}
		} else {
			for k, v := range prevResult.Output {
				output[k] = v
			}
		}
	}

	if stage.Config.Expression != "" {
		var parsed any
		if err := json.Unmarshal([]byte(stage.Config.Expression), &parsed); err != nil {
			return nil, fmt.Errorf("transform expression error: %w", err)
		}
		output["expression_result"] = parsed
	}

	return &upal.StageResult{
		StageID: stage.ID,
		Status:  upal.StageStatusCompleted,
		Output:  output,
	}, nil
}
