package model

import (
	"strings"
	"testing"

	"google.golang.org/genai"
	adkmodel "google.golang.org/adk/model"
)

// ---------------------------------------------------------------------------
// extractCustomToolDefs tests
// ---------------------------------------------------------------------------

func TestExtractCustomToolDefs_NoTools(t *testing.T) {
	req := &adkmodel.LLMRequest{}
	defs := extractCustomToolDefs(req)
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
}

func TestExtractCustomToolDefs_OnlyNative(t *testing.T) {
	req := &adkmodel.LLMRequest{
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{GoogleSearch: &genai.GoogleSearch{}}},
		},
	}
	defs := extractCustomToolDefs(req)
	if len(defs) != 0 {
		t.Errorf("expected 0 custom defs for native-only tools, got %d", len(defs))
	}
}

func TestExtractCustomToolDefs_CustomFunctions(t *testing.T) {
	req := &adkmodel.LLMRequest{
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{FunctionDeclarations: []*genai.FunctionDeclaration{
					{
						Name:        "generate_workflow",
						Description: "Generate a workflow",
						Parameters: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"description": {Type: genai.TypeString, Description: "What to generate"},
							},
							Required: []string{"description"},
						},
					},
					{
						Name:        "add_node",
						Description: "Add a node",
						Parameters: &genai.Schema{
							Type: genai.TypeObject,
							Properties: map[string]*genai.Schema{
								"node_type": {Type: genai.TypeString},
								"label":     {Type: genai.TypeString},
							},
							Required: []string{"node_type", "label"},
						},
					},
				}},
			},
		},
	}
	defs := extractCustomToolDefs(req)
	if len(defs) != 2 {
		t.Fatalf("expected 2 defs, got %d", len(defs))
	}
	if defs[0].Name != "generate_workflow" {
		t.Errorf("expected first def name 'generate_workflow', got %q", defs[0].Name)
	}
}

func TestExtractCustomToolDefs_MixedNativeAndCustom(t *testing.T) {
	req := &adkmodel.LLMRequest{
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{GoogleSearch: &genai.GoogleSearch{}},
				{FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "my_tool", Description: "A custom tool"},
				}},
			},
		},
	}
	defs := extractCustomToolDefs(req)
	if len(defs) != 1 {
		t.Fatalf("expected 1 custom def, got %d", len(defs))
	}
	if defs[0].Name != "my_tool" {
		t.Errorf("expected 'my_tool', got %q", defs[0].Name)
	}
}

// ---------------------------------------------------------------------------
// buildToolPrompt tests
// ---------------------------------------------------------------------------

func TestBuildToolPrompt_Empty(t *testing.T) {
	result := buildToolPrompt(nil)
	if result != "" {
		t.Errorf("expected empty string for nil defs, got %q", result)
	}
}

func TestBuildToolPrompt_SingleTool(t *testing.T) {
	defs := []*genai.FunctionDeclaration{{
		Name:        "get_weather",
		Description: "Get current weather",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"city": {Type: genai.TypeString, Description: "City name"},
			},
			Required: []string{"city"},
		},
	}}
	result := buildToolPrompt(defs)
	if !strings.Contains(result, "get_weather") {
		t.Error("expected tool name in prompt")
	}
	if !strings.Contains(result, "<tool_call>") {
		t.Error("expected <tool_call> format instruction")
	}
	if !strings.Contains(result, "city") {
		t.Error("expected parameter name in prompt")
	}
	if !strings.Contains(result, "required") {
		t.Error("expected required marker in prompt")
	}
}

func TestBuildToolPrompt_MultipleTools(t *testing.T) {
	defs := []*genai.FunctionDeclaration{
		{Name: "tool_a", Description: "First tool"},
		{Name: "tool_b", Description: "Second tool"},
	}
	result := buildToolPrompt(defs)
	if !strings.Contains(result, "tool_a") || !strings.Contains(result, "tool_b") {
		t.Error("expected both tool names")
	}
}

func TestBuildToolPrompt_NoParameters(t *testing.T) {
	defs := []*genai.FunctionDeclaration{
		{Name: "list_all", Description: "List everything"},
	}
	result := buildToolPrompt(defs)
	if !strings.Contains(result, "list_all") {
		t.Error("expected tool name")
	}
	if strings.Contains(result, "Parameters:") {
		t.Error("should not include Parameters section for tool without params")
	}
}

// ---------------------------------------------------------------------------
// schemaTypeString tests
// ---------------------------------------------------------------------------

func TestSchemaTypeString(t *testing.T) {
	tests := []struct {
		schema genai.Type
		want   string
	}{
		{genai.TypeString, "string"},
		{genai.TypeNumber, "number"},
		{genai.TypeInteger, "integer"},
		{genai.TypeBoolean, "boolean"},
		{genai.TypeArray, "array"},
		{genai.TypeObject, "object"},
	}
	for _, tt := range tests {
		got := schemaTypeString(&genai.Schema{Type: tt.schema})
		if got != tt.want {
			t.Errorf("schemaTypeString(%v) = %q, want %q", tt.schema, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// parseToolCalls tests
// ---------------------------------------------------------------------------

func TestParseToolCalls_NoToolCalls(t *testing.T) {
	text := "Here is a simple text response with no tool calls."
	calls, remaining := parseToolCalls(text)
	if len(calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(calls))
	}
	if remaining != text {
		t.Errorf("expected unchanged text, got %q", remaining)
	}
}

func TestParseToolCalls_SingleCall(t *testing.T) {
	text := `I'll generate that for you.

<tool_call>
{"name": "generate_workflow", "arguments": {"description": "blog writer"}}
</tool_call>

Please wait.`

	calls, remaining := parseToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "generate_workflow" {
		t.Errorf("expected name 'generate_workflow', got %q", calls[0].Name)
	}
	args := calls[0].Args
	if args["description"] != "blog writer" {
		t.Errorf("expected arg 'blog writer', got %v", args["description"])
	}
	if calls[0].ID != "cc_tool_0" {
		t.Errorf("expected ID 'cc_tool_0', got %q", calls[0].ID)
	}
	// Remaining text should have the tool call blocks removed.
	if strings.Contains(remaining, "<tool_call>") {
		t.Error("remaining text should not contain <tool_call> blocks")
	}
}

func TestParseToolCalls_MultipleCallsAndText(t *testing.T) {
	text := `Let me do two things.

<tool_call>
{"name": "add_node", "arguments": {"node_type": "input", "label": "Topic"}}
</tool_call>

<tool_call>
{"name": "add_node", "arguments": {"node_type": "agent", "label": "Writer"}}
</tool_call>`

	calls, _ := parseToolCalls(text)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].ID != "cc_tool_0" || calls[1].ID != "cc_tool_1" {
		t.Errorf("expected sequential IDs, got %q and %q", calls[0].ID, calls[1].ID)
	}
}

func TestParseToolCalls_BrokenJSON(t *testing.T) {
	text := `<tool_call>
{not valid json}
</tool_call>`

	calls, remaining := parseToolCalls(text)
	if len(calls) != 0 {
		t.Errorf("expected 0 calls for broken JSON, got %d", len(calls))
	}
	// Broken blocks are left in the text as-is.
	if !strings.Contains(remaining, "not valid json") {
		t.Error("broken content should remain in text")
	}
}

func TestParseToolCalls_CompactFormat(t *testing.T) {
	// Some LLMs may output without extra whitespace.
	text := `<tool_call>{"name":"list_nodes","arguments":{}}</tool_call>`

	calls, _ := parseToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "list_nodes" {
		t.Errorf("expected 'list_nodes', got %q", calls[0].Name)
	}
}

func TestParseToolCalls_EmptyName(t *testing.T) {
	text := `<tool_call>
{"name": "", "arguments": {}}
</tool_call>`

	calls, _ := parseToolCalls(text)
	if len(calls) != 0 {
		t.Errorf("expected 0 calls for empty name, got %d", len(calls))
	}
}

// ---------------------------------------------------------------------------
// buildUserMessage tests
// ---------------------------------------------------------------------------

func TestBuildUserMessage_TextOnly(t *testing.T) {
	contents := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Hello")}},
		{Role: "model", Parts: []*genai.Part{genai.NewPartFromText("Hi there")}},
		{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Generate a blog workflow")}},
	}
	msg := buildUserMessage(contents)
	if !strings.Contains(msg, "Hello") {
		t.Error("expected user message in output")
	}
	if !strings.Contains(msg, "[Assistant]:") {
		t.Error("expected assistant prefix")
	}
}

func TestBuildUserMessage_WithFunctionCall(t *testing.T) {
	contents := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Generate a workflow")}},
		{Role: "model", Parts: []*genai.Part{
			genai.NewPartFromText("I'll generate that."),
			{FunctionCall: &genai.FunctionCall{
				ID:   "cc_tool_0",
				Name: "generate_workflow",
				Args: map[string]any{"description": "blog writer"},
			}},
		}},
	}
	msg := buildUserMessage(contents)
	if !strings.Contains(msg, "[Assistant called tool: generate_workflow]") {
		t.Error("expected FunctionCall text representation")
	}
	if !strings.Contains(msg, "blog writer") {
		t.Error("expected function call arguments in output")
	}
}

func TestBuildUserMessage_WithFunctionResponse(t *testing.T) {
	contents := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("Generate a workflow")}},
		{Role: "model", Parts: []*genai.Part{
			{FunctionCall: &genai.FunctionCall{
				ID:   "cc_tool_0",
				Name: "generate_workflow",
				Args: map[string]any{"description": "blog"},
			}},
		}},
		{Role: "user", Parts: []*genai.Part{
			{FunctionResponse: &genai.FunctionResponse{
				ID:       "cc_tool_0",
				Name:     "generate_workflow",
				Response: map[string]any{"name": "my-blog", "nodes": []any{}},
			}},
		}},
	}
	msg := buildUserMessage(contents)
	if !strings.Contains(msg, "[Tool result: generate_workflow]") {
		t.Error("expected FunctionResponse text representation")
	}
	if !strings.Contains(msg, "my-blog") {
		t.Error("expected tool result content")
	}
}

func TestBuildUserMessage_Empty(t *testing.T) {
	msg := buildUserMessage(nil)
	if msg != "" {
		t.Errorf("expected empty string for nil contents, got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// buildToolResponse tests
// ---------------------------------------------------------------------------

func TestBuildToolResponse_WithToolCalls(t *testing.T) {
	text := `I'll generate that workflow.

<tool_call>
{"name": "generate_workflow", "arguments": {"description": "a blog writer"}}
</tool_call>`

	resp := buildToolResponse(text)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	var hasText, hasCall bool
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			hasText = true
		}
		if p.FunctionCall != nil {
			hasCall = true
			if p.FunctionCall.Name != "generate_workflow" {
				t.Errorf("expected tool name 'generate_workflow', got %q", p.FunctionCall.Name)
			}
		}
	}
	if !hasText {
		t.Error("expected text part for prose before tool call")
	}
	if !hasCall {
		t.Error("expected FunctionCall part")
	}
}

func TestBuildToolResponse_PlainText(t *testing.T) {
	text := "Here is your answer: hello world"
	resp := buildToolResponse(text)
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(resp.Content.Parts))
	}
	if resp.Content.Parts[0].Text != text {
		t.Errorf("expected plain text passthrough")
	}
}

func TestBuildToolResponse_OnlyToolCall(t *testing.T) {
	text := `<tool_call>
{"name": "list_nodes", "arguments": {}}
</tool_call>`

	resp := buildToolResponse(text)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	// Only a tool call, no surrounding text — should have just the FunctionCall part.
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("expected 1 part (FunctionCall only), got %d", len(resp.Content.Parts))
	}
	if resp.Content.Parts[0].FunctionCall == nil {
		t.Error("expected FunctionCall part")
	}
}

func TestBuildToolResponse_MultipleToolCalls(t *testing.T) {
	text := `Let me add nodes.

<tool_call>
{"name": "add_node", "arguments": {"type": "input"}}
</tool_call>

<tool_call>
{"name": "add_node", "arguments": {"type": "agent"}}
</tool_call>`

	resp := buildToolResponse(text)
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	var textCount, callCount int
	for _, p := range resp.Content.Parts {
		if p.Text != "" {
			textCount++
		}
		if p.FunctionCall != nil {
			callCount++
		}
	}
	if textCount != 1 {
		t.Errorf("expected 1 text part, got %d", textCount)
	}
	if callCount != 2 {
		t.Errorf("expected 2 FunctionCall parts, got %d", callCount)
	}
}

func TestBuildToolResponse_TurnComplete(t *testing.T) {
	resp := buildToolResponse("hello")
	if !resp.TurnComplete {
		t.Error("expected TurnComplete to be true")
	}
	if resp.FinishReason != genai.FinishReasonStop {
		t.Errorf("expected FinishReasonStop, got %v", resp.FinishReason)
	}
	if resp.Content.Role != "model" {
		t.Errorf("expected role 'model', got %q", resp.Content.Role)
	}
}
