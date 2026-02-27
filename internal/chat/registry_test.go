package chat

import (
	"context"
	"fmt"
	"testing"
)

func TestResolve_BaseTools(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ChatTool{
		Name:        "list_workflows",
		Description: "List all workflows",
	})
	reg.Register(&ChatTool{
		Name:        "create_workflow",
		Description: "Create a new workflow",
	})
	reg.AddRule(Rule{
		Page:  "workflows",
		Tools: []string{"list_workflows", "create_workflow"},
	})

	tools := reg.Resolve("workflows", nil)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["list_workflows"] {
		t.Error("expected list_workflows in resolved tools")
	}
	if !names["create_workflow"] {
		t.Error("expected create_workflow in resolved tools")
	}
}

func TestResolve_ConditionalTools(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ChatTool{
		Name:        "base_tool",
		Description: "Always available",
	})
	reg.Register(&ChatTool{
		Name:        "node_tool",
		Description: "Only when a node is selected",
	})

	// Base rule — always available on "workflows" page.
	reg.AddRule(Rule{
		Page:  "workflows",
		Tools: []string{"base_tool"},
	})
	// Conditional rule — only when selected_node_id is present.
	reg.AddRule(Rule{
		Page: "workflows",
		Condition: func(ctx map[string]any) bool {
			_, ok := ctx["selected_node_id"]
			return ok
		},
		Tools: []string{"node_tool"},
	})

	// Without selected_node_id: only base_tool.
	tools := reg.Resolve("workflows", map[string]any{})
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool without context, got %d", len(tools))
	}
	if tools[0].Name != "base_tool" {
		t.Errorf("expected base_tool, got %q", tools[0].Name)
	}

	// With selected_node_id: both tools.
	tools = reg.Resolve("workflows", map[string]any{"selected_node_id": "abc"})
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools with context, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["base_tool"] {
		t.Error("expected base_tool in resolved tools")
	}
	if !names["node_tool"] {
		t.Error("expected node_tool in resolved tools")
	}
}

func TestResolve_UnknownPage(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ChatTool{
		Name:        "some_tool",
		Description: "A tool",
	})
	reg.AddRule(Rule{
		Page:  "workflows",
		Tools: []string{"some_tool"},
	})

	tools := reg.Resolve("unknown", nil)
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools for unknown page, got %d", len(tools))
	}
}

func TestExecuteToolCall_Success(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ChatTool{
		Name:        "greet",
		Description: "Greet someone",
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			name, _ := args["name"].(string)
			return map[string]any{"greeting": "Hello, " + name + "!"}, nil
		},
	})

	result, err := reg.ExecuteToolCall(context.Background(), "greet", map[string]any{"name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["greeting"] != "Hello, World!" {
		t.Errorf("expected greeting 'Hello, World!', got %q", m["greeting"])
	}
}

func TestExecuteToolCall_UnknownTool(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.ExecuteToolCall(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if want := "unknown chat tool: nonexistent"; err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestExecuteToolCall_ExecuteError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ChatTool{
		Name:        "failing",
		Description: "A tool that fails",
		Execute: func(ctx context.Context, args map[string]any) (any, error) {
			return nil, fmt.Errorf("something went wrong")
		},
	})

	_, err := reg.ExecuteToolCall(context.Background(), "failing", nil)
	if err == nil {
		t.Fatal("expected error from failing tool")
	}
	if err.Error() != "something went wrong" {
		t.Errorf("error = %q, want %q", err.Error(), "something went wrong")
	}
}

func TestResolve_DedupesToolNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ChatTool{
		Name:        "shared_tool",
		Description: "Shared",
	})
	// Two rules for the same page both reference the same tool.
	reg.AddRule(Rule{Page: "workflows", Tools: []string{"shared_tool"}})
	reg.AddRule(Rule{Page: "workflows", Tools: []string{"shared_tool"}})

	tools := reg.Resolve("workflows", nil)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool (deduplicated), got %d", len(tools))
	}
}

func TestResolve_RuleReferencesUnregisteredTool(t *testing.T) {
	reg := NewRegistry()
	// Rule references a tool that was never registered.
	reg.AddRule(Rule{Page: "workflows", Tools: []string{"ghost_tool"}})

	tools := reg.Resolve("workflows", nil)
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools (unregistered tool skipped), got %d", len(tools))
	}
}
