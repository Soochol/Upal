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

func TestParseToolCalls(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		wantCalls      int
		wantFirstName  string
		wantFirstID    string
		wantFirstArg   string // key=value to check in first call's Args
		remainContains string // substring that must be in remaining text
		remainExcludes string // substring that must NOT be in remaining text
	}{
		{
			name:           "no tool calls",
			text:           "Here is a simple text response with no tool calls.",
			wantCalls:      0,
			remainContains: "simple text response",
		},
		{
			name: "single call with surrounding text",
			text: "I'll generate that for you.\n\n<tool_call>\n" +
				`{"name": "generate_workflow", "arguments": {"description": "blog writer"}}` +
				"\n</tool_call>\n\nPlease wait.",
			wantCalls:      1,
			wantFirstName:  "generate_workflow",
			wantFirstID:    "cc_tool_0",
			wantFirstArg:   "description=blog writer",
			remainExcludes: "<tool_call>",
		},
		{
			name: "multiple calls with sequential IDs",
			text: "Let me do two things.\n\n<tool_call>\n" +
				`{"name": "add_node", "arguments": {"node_type": "input", "label": "Topic"}}` +
				"\n</tool_call>\n\n<tool_call>\n" +
				`{"name": "add_node", "arguments": {"node_type": "agent", "label": "Writer"}}` +
				"\n</tool_call>",
			wantCalls:     2,
			wantFirstName: "add_node",
			wantFirstID:   "cc_tool_0",
		},
		{
			name:           "broken JSON left in text",
			text:           "<tool_call>\n{not valid json}\n</tool_call>",
			wantCalls:      0,
			remainContains: "not valid json",
		},
		{
			name:          "compact format without whitespace",
			text:          `<tool_call>{"name":"list_nodes","arguments":{}}</tool_call>`,
			wantCalls:     1,
			wantFirstName: "list_nodes",
		},
		{
			name:      "empty name is skipped",
			text:      "<tool_call>\n{\"name\": \"\", \"arguments\": {}}\n</tool_call>",
			wantCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, remaining := parseToolCalls(tt.text)
			if len(calls) != tt.wantCalls {
				t.Fatalf("expected %d calls, got %d", tt.wantCalls, len(calls))
			}
			if tt.wantCalls > 0 {
				if tt.wantFirstName != "" && calls[0].Name != tt.wantFirstName {
					t.Errorf("expected first name %q, got %q", tt.wantFirstName, calls[0].Name)
				}
				if tt.wantFirstID != "" && calls[0].ID != tt.wantFirstID {
					t.Errorf("expected first ID %q, got %q", tt.wantFirstID, calls[0].ID)
				}
				if tt.wantFirstArg != "" {
					parts := strings.SplitN(tt.wantFirstArg, "=", 2)
					if v, _ := calls[0].Args[parts[0]].(string); v != parts[1] {
						t.Errorf("expected arg %s=%s, got %v", parts[0], parts[1], calls[0].Args[parts[0]])
					}
				}
			}
			if tt.wantCalls == 2 && calls[1].ID != "cc_tool_1" {
				t.Errorf("expected second ID 'cc_tool_1', got %q", calls[1].ID)
			}
			if tt.remainContains != "" && !strings.Contains(remaining, tt.remainContains) {
				t.Errorf("remaining should contain %q, got %q", tt.remainContains, remaining)
			}
			if tt.remainExcludes != "" && strings.Contains(remaining, tt.remainExcludes) {
				t.Errorf("remaining should not contain %q", tt.remainExcludes)
			}
		})
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

func TestBuildToolResponse(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantTexts int
		wantCalls int
		wantName  string // expected name of first FunctionCall (if any)
	}{
		{
			name: "text with tool call",
			text: "I'll generate that workflow.\n\n<tool_call>\n" +
				`{"name": "generate_workflow", "arguments": {"description": "a blog writer"}}` +
				"\n</tool_call>",
			wantTexts: 1,
			wantCalls: 1,
			wantName:  "generate_workflow",
		},
		{
			name:      "plain text",
			text:      "Here is your answer: hello world",
			wantTexts: 1,
			wantCalls: 0,
		},
		{
			name: "only tool call, no surrounding text",
			text: "<tool_call>\n" +
				`{"name": "list_nodes", "arguments": {}}` +
				"\n</tool_call>",
			wantTexts: 0,
			wantCalls: 1,
			wantName:  "list_nodes",
		},
		{
			name: "multiple tool calls with text",
			text: "Let me add nodes.\n\n<tool_call>\n" +
				`{"name": "add_node", "arguments": {"type": "input"}}` +
				"\n</tool_call>\n\n<tool_call>\n" +
				`{"name": "add_node", "arguments": {"type": "agent"}}` +
				"\n</tool_call>",
			wantTexts: 1,
			wantCalls: 2,
			wantName:  "add_node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := buildToolResponse(tt.text)
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
			if textCount != tt.wantTexts {
				t.Errorf("expected %d text parts, got %d", tt.wantTexts, textCount)
			}
			if callCount != tt.wantCalls {
				t.Errorf("expected %d FunctionCall parts, got %d", tt.wantCalls, callCount)
			}
			if tt.wantName != "" {
				for _, p := range resp.Content.Parts {
					if p.FunctionCall != nil {
						if p.FunctionCall.Name != tt.wantName {
							t.Errorf("expected tool name %q, got %q", tt.wantName, p.FunctionCall.Name)
						}
						break
					}
				}
			}
		})
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
