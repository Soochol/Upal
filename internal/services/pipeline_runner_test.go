// internal/services/pipeline_runner_test.go
package services

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// mockStageExecutor records calls and returns canned results.
type mockStageExecutor struct {
	stageType string
	calls     []string
	output    map[string]any
	err       error
}

func (m *mockStageExecutor) Type() string { return m.stageType }
func (m *mockStageExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	m.calls = append(m.calls, stage.ID)
	if m.err != nil {
		return nil, m.err
	}
	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "completed",
		Output:  m.output,
	}, nil
}

func TestPipelineRunner_ExecuteSequential(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	wfExec := &mockStageExecutor{stageType: "workflow", output: map[string]any{"result": "ok"}}
	transformExec := &mockStageExecutor{stageType: "transform", output: map[string]any{"transformed": true}}

	runner := NewPipelineRunner(runRepo)
	runner.RegisterExecutor(wfExec)
	runner.RegisterExecutor(transformExec)

	pipeline := &upal.Pipeline{
		ID:   "pipe-1",
		Name: "Test",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Collect", Type: "workflow"},
			{ID: "s2", Name: "Transform", Type: "transform"},
		},
	}

	run, err := runner.Start(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	if run.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", run.Status)
	}
	if len(wfExec.calls) != 1 || wfExec.calls[0] != "s1" {
		t.Errorf("expected workflow executor called with s1, got %v", wfExec.calls)
	}
	if len(transformExec.calls) != 1 || transformExec.calls[0] != "s2" {
		t.Errorf("expected transform executor called with s2, got %v", transformExec.calls)
	}
}

func TestPipelineRunner_StageFailure(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	failExec := &mockStageExecutor{
		stageType: "workflow",
		err:       context.DeadlineExceeded,
	}

	runner := NewPipelineRunner(runRepo)
	runner.RegisterExecutor(failExec)

	pipeline := &upal.Pipeline{
		ID:   "pipe-2",
		Name: "Fail Test",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Broken", Type: "workflow"},
		},
	}

	run, err := runner.Start(context.Background(), pipeline)
	if err == nil {
		t.Fatal("expected error from failed stage")
	}
	if run.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", run.Status)
	}
}

func TestPipelineRunner_UnknownStageType(t *testing.T) {
	runner := NewPipelineRunner(repository.NewMemoryPipelineRunRepository())

	pipeline := &upal.Pipeline{
		ID:   "pipe-3",
		Name: "Unknown",
		Stages: []upal.Stage{
			{ID: "s1", Type: "quantum_computer"},
		},
	}

	_, err := runner.Start(context.Background(), pipeline)
	if err == nil {
		t.Fatal("expected error for unknown stage type")
	}
}
