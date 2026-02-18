package tools

import (
	"context"
	"testing"
)

type echoTool struct{}

func (e *echoTool) Name() string               { return "echo" }
func (e *echoTool) Description() string         { return "Echoes input" }
func (e *echoTool) InputSchema() map[string]any { return map[string]any{"type": "object"} }
func (e *echoTool) Execute(ctx context.Context, input any) (any, error) { return input, nil }

func TestToolRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&echoTool{})
	tool, ok := reg.Get("echo")
	if !ok {
		t.Fatal("echo tool not found")
	}
	if tool.Name() != "echo" {
		t.Errorf("name: got %q", tool.Name())
	}
}

func TestToolRegistry_Execute(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&echoTool{})
	result, err := reg.Execute(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "hello" {
		t.Errorf("result: got %v, want hello", result)
	}
}

func TestToolRegistry_Execute_Unknown(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Execute(context.Background(), "unknown", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&echoTool{})
	tools := reg.List()
	if len(tools) != 1 {
		t.Fatalf("list: got %d, want 1", len(tools))
	}
}
