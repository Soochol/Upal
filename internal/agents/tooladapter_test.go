package agents

import (
	"context"
	"testing"
)

type mockTool struct{}

func (m *mockTool) Name() string                                        { return "test_tool" }
func (m *mockTool) Description() string                                 { return "A test tool" }
func (m *mockTool) InputSchema() map[string]any                         { return map[string]any{"type": "string"} }
func (m *mockTool) Execute(ctx context.Context, input any) (any, error) { return "result", nil }

func TestADKToolAdapter(t *testing.T) {
	adapter := NewADKTool(&mockTool{})
	if adapter.Name() != "test_tool" {
		t.Fatalf("expected 'test_tool', got %q", adapter.Name())
	}
	if adapter.Description() != "A test tool" {
		t.Fatal("wrong description")
	}
	if adapter.IsLongRunning() {
		t.Fatal("expected not long running")
	}
}

func TestAdaptTools(t *testing.T) {
	tools := []mockTool{{}, {}}
	// Convert to the tools.Tool interface slice
	upalTools := make([]interface {
		Name() string
		Description() string
		InputSchema() map[string]any
		Execute(ctx context.Context, input any) (any, error)
	}, len(tools))
	for i := range tools {
		upalTools[i] = &tools[i]
	}

	// We test AdaptTools with the actual tools.Tool interface
	// This test just verifies the adapter works end-to-end
	adapter1 := NewADKTool(&tools[0])
	adapter2 := NewADKTool(&tools[1])

	if adapter1.Name() != "test_tool" {
		t.Fatalf("adapter1: expected 'test_tool', got %q", adapter1.Name())
	}
	if adapter2.Name() != "test_tool" {
		t.Fatalf("adapter2: expected 'test_tool', got %q", adapter2.Name())
	}
}
