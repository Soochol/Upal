package services_test

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
)

// stubFetcher implements SourceFetcher for testing.
type stubFetcher struct {
	typ  string
	text string
	data any
	err  error
}

func (s *stubFetcher) Type() string { return s.typ }
func (s *stubFetcher) Fetch(_ context.Context, _ upal.CollectSource) (string, any, error) {
	return s.text, s.data, s.err
}

func TestCollectStageExecutor_CustomFetcher(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	exec.RegisterFetcher(&stubFetcher{typ: "mock", text: "hello", data: "data"})

	stage := upal.Stage{
		ID:   "s1",
		Type: "collect",
		Config: upal.StageConfig{
			Sources: []upal.CollectSource{{ID: "src1", Type: "mock", URL: "http://x"}},
		},
	}
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["text"] != "hello" {
		t.Errorf("expected text='hello', got %q", result.Output["text"])
	}
}

func TestCollectStageExecutor_UnknownSourceType(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	stage := upal.Stage{
		ID:   "s1",
		Type: "collect",
		Config: upal.StageConfig{
			Sources: []upal.CollectSource{{ID: "src1", Type: "nonexistent", URL: "http://x"}},
		},
	}
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text, _ := result.Output["text"].(string)
	if text == "" {
		t.Error("expected error text for unknown source type")
	}
}
