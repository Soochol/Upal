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

// mockWaitingExecutor always returns "waiting" (simulates an approval gate).
type mockWaitingExecutor struct {
	stageType string
	calls     []string
}

func (m *mockWaitingExecutor) Type() string { return m.stageType }
func (m *mockWaitingExecutor) Execute(_ context.Context, stage upal.Stage, _ *upal.StageResult) (*upal.StageResult, error) {
	m.calls = append(m.calls, stage.ID)
	return &upal.StageResult{
		StageID: stage.ID,
		Status:  "waiting",
		Output:  map[string]any{"message": "please approve"},
	}, nil
}

func TestPipelineRunner_Resume(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	approvalExec := &mockWaitingExecutor{stageType: "approval"}
	wfExec := &mockStageExecutor{stageType: "workflow", output: map[string]any{"done": true}}

	runner := NewPipelineRunner(runRepo)
	runner.RegisterExecutor(approvalExec)
	runner.RegisterExecutor(wfExec)

	pipeline := &upal.Pipeline{
		ID:   "pipe-resume",
		Name: "Resume Test",
		Stages: []upal.Stage{
			{ID: "s1", Name: "Collect", Type: "workflow"},
			{ID: "s2", Name: "Approve", Type: "approval"},
			{ID: "s3", Name: "Process", Type: "workflow"},
		},
	}

	// Start: should pause at s2
	run, err := runner.Start(context.Background(), pipeline)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if run.Status != "waiting" {
		t.Fatalf("expected status 'waiting', got %q", run.Status)
	}
	if run.CurrentStage != "s2" {
		t.Errorf("expected current_stage 's2', got %q", run.CurrentStage)
	}
	if len(wfExec.calls) != 1 || wfExec.calls[0] != "s1" {
		t.Errorf("expected s1 executed before pause, got %v", wfExec.calls)
	}

	// Resume: should skip s2 (already done), execute s3
	wfExec.calls = nil
	err = runner.Resume(context.Background(), pipeline, run)
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("expected status 'completed' after resume, got %q", run.Status)
	}
	if len(wfExec.calls) != 1 || wfExec.calls[0] != "s3" {
		t.Errorf("expected only s3 executed after resume, got %v", wfExec.calls)
	}
	// approval executor must NOT be called again
	if len(approvalExec.calls) != 1 {
		t.Errorf("approval executor should have been called exactly once, got %d", len(approvalExec.calls))
	}
}

func TestPipelineRunner_Resume_CurrentStageNotFound(t *testing.T) {
	runRepo := repository.NewMemoryPipelineRunRepository()
	runner := NewPipelineRunner(runRepo)

	pipeline := &upal.Pipeline{
		ID:     "pipe-bad",
		Name:   "Bad",
		Stages: []upal.Stage{{ID: "s1", Type: "workflow"}},
	}
	run := &upal.PipelineRun{
		ID:           "prun-bad",
		PipelineID:   "pipe-bad",
		Status:       "waiting",
		CurrentStage: "nonexistent",
		StageResults: make(map[string]*upal.StageResult),
	}
	if err := runRepo.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	err := runner.Resume(context.Background(), pipeline, run)
	if err == nil {
		t.Fatal("expected error for nonexistent current stage")
	}
}
