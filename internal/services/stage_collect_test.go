package services_test

import (
	"context"
	"fmt"
	"strings"
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

func TestCollectStageExecutor_EmptySources(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	stage := upal.Stage{
		ID:     "s1",
		Type:   "collect",
		Config: upal.StageConfig{Sources: nil},
	}
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Output["text"] != "" {
		t.Errorf("expected empty text, got %q", result.Output["text"])
	}
	if result.Status != "completed" {
		t.Errorf("expected status=completed, got %q", result.Status)
	}
}

func TestCollectStageExecutor_MultiSource(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	exec.RegisterFetcher(&stubFetcher{typ: "mock", text: "hello", data: "d1"})

	stage := upal.Stage{
		ID:   "s1",
		Type: "collect",
		Config: upal.StageConfig{
			Sources: []upal.CollectSource{
				{ID: "a", Type: "mock", URL: "http://x"},
				{ID: "b", Type: "mock", URL: "http://y"},
			},
		},
	}
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text, _ := result.Output["text"].(string)
	sources, _ := result.Output["sources"].(map[string]any)
	if _, ok := sources["a"]; !ok {
		t.Error("expected source 'a' in sources map")
	}
	if _, ok := sources["b"]; !ok {
		t.Error("expected source 'b' in sources map")
	}
	if !strings.Contains(text, "hello") {
		t.Errorf("expected combined text to contain 'hello', got %q", text)
	}
}

func TestCollectStageExecutor_PartialFailure(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	exec.RegisterFetcher(&stubFetcher{typ: "ok", text: "good", data: "d"})
	exec.RegisterFetcher(&stubFetcher{typ: "bad", err: fmt.Errorf("fetch failed")})

	stage := upal.Stage{
		ID:   "s1",
		Type: "collect",
		Config: upal.StageConfig{
			Sources: []upal.CollectSource{
				{ID: "good-src", Type: "ok", URL: "http://x"},
				{ID: "bad-src", Type: "bad", URL: "http://y"},
			},
		},
	}
	result, err := exec.Execute(context.Background(), stage, nil)
	if err != nil {
		t.Fatalf("Execute should not return error on partial failure: %v", err)
	}
	text, _ := result.Output["text"].(string)
	if !strings.Contains(text, "good") {
		t.Error("expected successful source text in output")
	}
	if !strings.Contains(text, "fetch failed") {
		t.Error("expected error text from failed source in output")
	}
}

func TestCollectStageExecutor_FetcherReplacement(t *testing.T) {
	exec := services.NewCollectStageExecutor()
	exec.RegisterFetcher(&stubFetcher{typ: "mock", text: "first"})
	exec.RegisterFetcher(&stubFetcher{typ: "mock", text: "second"})

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
	if result.Output["text"] != "second" {
		t.Errorf("expected second fetcher to win, got %q", result.Output["text"])
	}
}
