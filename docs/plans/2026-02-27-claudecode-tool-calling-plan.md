# ClaudeCodeLLM Tool Calling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make ClaudeCodeLLM support custom function calling so chat, generation, and workflow execution all work with the Claude Code CLI model.

**Architecture:** Intercept custom FunctionDeclarations from the request, inject them as text in the system prompt with a strict output format, parse `<tool_call>` blocks from CLI text output back into `genai.FunctionCall` Parts, and convert incoming `FunctionResponse` Parts into text for multi-turn conversations. The handler/executor layer requires zero changes.

**Tech Stack:** Go, regexp, encoding/json, google.golang.org/genai, ADK model interface

**Design doc:** `docs/plans/2026-02-27-claudecode-tool-calling-design.md`

---

### Task 1: Extract custom FunctionDeclarations from request

**Files:**
- Modify: `internal/model/claudecode.go`
- Create: `internal/model/claudecode_test.go`

**Step 1: Write the failing test**

```go
// internal/model/claudecode_test.go
package model

import (
	"testing"

	"google.golang.org/genai"
	adkmodel "google.golang.org/adk/model"
)

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/... -v -race -run TestExtractCustomToolDefs`
Expected: FAIL — `extractCustomToolDefs` undefined

**Step 3: Write minimal implementation**

Add to `claudecode.go`:

```go
// extractCustomToolDefs returns all FunctionDeclarations from the request's tools,
// excluding native tools (GoogleSearch, etc.) that are handled by CLI tool mappings.
func extractCustomToolDefs(req *adkmodel.LLMRequest) []*genai.FunctionDeclaration {
	if req.Config == nil {
		return nil
	}
	var defs []*genai.FunctionDeclaration
	for _, tool := range req.Config.Tools {
		defs = append(defs, tool.FunctionDeclarations...)
	}
	return defs
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model/... -v -race -run TestExtractCustomToolDefs`
Expected: PASS (all 4 tests)

**Step 5: Commit**

```bash
git add internal/model/claudecode.go internal/model/claudecode_test.go
git commit -m "feat(claudecode): extract custom FunctionDeclarations from request"
```

---

### Task 2: Build tool definitions prompt text

**Files:**
- Modify: `internal/model/claudecode.go`
- Modify: `internal/model/claudecode_test.go`

**Step 1: Write the failing test**

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/... -v -race -run TestBuildToolPrompt`
Expected: FAIL — `buildToolPrompt` undefined

**Step 3: Write minimal implementation**

```go
// buildToolPrompt generates a text-based tool definition section to append to the
// system prompt. It instructs the LLM to output tool calls in <tool_call> XML format.
func buildToolPrompt(defs []*genai.FunctionDeclaration) string {
	if len(defs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n---\n\n## Available Tools\n\n")
	sb.WriteString("You have access to the following tools. When you need to use a tool, output EXACTLY this format:\n\n")
	sb.WriteString("<tool_call>\n{\"name\": \"tool_name\", \"arguments\": {\"key\": \"value\"}}\n</tool_call>\n\n")
	sb.WriteString("You may output multiple <tool_call> blocks. After outputting tool calls, STOP and wait for results. Do NOT guess tool results.\n\n")

	for _, def := range defs {
		sb.WriteString("### ")
		sb.WriteString(def.Name)
		sb.WriteString("\n")
		if def.Description != "" {
			sb.WriteString(def.Description)
			sb.WriteString("\n")
		}
		if def.Parameters != nil && len(def.Parameters.Properties) > 0 {
			sb.WriteString("Parameters:\n")
			required := make(map[string]bool)
			for _, r := range def.Parameters.Required {
				required[r] = true
			}
			for name, prop := range def.Parameters.Properties {
				sb.WriteString("  - ")
				sb.WriteString(name)
				sb.WriteString(" (")
				sb.WriteString(schemaTypeString(prop))
				if required[name] {
					sb.WriteString(", required")
				}
				sb.WriteString(")")
				if prop.Description != "" {
					sb.WriteString(": ")
					sb.WriteString(prop.Description)
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func schemaTypeString(s *genai.Schema) string {
	switch s.Type {
	case genai.TypeString:
		return "string"
	case genai.TypeNumber:
		return "number"
	case genai.TypeInteger:
		return "integer"
	case genai.TypeBoolean:
		return "boolean"
	case genai.TypeArray:
		return "array"
	case genai.TypeObject:
		return "object"
	default:
		return "string"
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model/... -v -race -run TestBuildToolPrompt`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/claudecode.go internal/model/claudecode_test.go
git commit -m "feat(claudecode): build tool definition prompt for system prompt injection"
```

---

### Task 3: Parse `<tool_call>` blocks from text response

**Files:**
- Modify: `internal/model/claudecode.go`
- Modify: `internal/model/claudecode_test.go`

**Step 1: Write the failing test**

```go
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
	// Remaining text should have the tool call blocks removed
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
	// Broken blocks are left in the text as-is
	if !strings.Contains(remaining, "not valid json") {
		t.Error("broken content should remain in text")
	}
}

func TestParseToolCalls_CompactFormat(t *testing.T) {
	// Some LLMs may output without extra whitespace
	text := `<tool_call>{"name":"list_nodes","arguments":{}}</tool_call>`

	calls, _ := parseToolCalls(text)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "list_nodes" {
		t.Errorf("expected 'list_nodes', got %q", calls[0].Name)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/... -v -race -run TestParseToolCalls`
Expected: FAIL — `parseToolCalls` undefined

**Step 3: Write minimal implementation**

```go
import (
	"encoding/json"
	"regexp"
)

var toolCallRe = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)

type rawToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// parseToolCalls extracts <tool_call> blocks from text, returning parsed FunctionCalls
// and the remaining text with successfully parsed blocks removed.
func parseToolCalls(text string) ([]*genai.FunctionCall, string) {
	matches := toolCallRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil, text
	}

	var calls []*genai.FunctionCall
	// Track which match ranges to remove (only successfully parsed ones).
	var removeRanges [][2]int

	for _, match := range matches {
		jsonStr := text[match[2]:match[3]]
		var raw rawToolCall
		if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
			continue // Skip broken JSON, leave in text
		}
		if raw.Name == "" {
			continue
		}
		calls = append(calls, &genai.FunctionCall{
			ID:   fmt.Sprintf("cc_tool_%d", len(calls)),
			Name: raw.Name,
			Args: raw.Arguments,
		})
		removeRanges = append(removeRanges, [2]int{match[0], match[1]})
	}

	// Build remaining text by removing parsed tool call blocks.
	remaining := removeRangesFromText(text, removeRanges)
	remaining = strings.TrimSpace(remaining)

	return calls, remaining
}

func removeRangesFromText(text string, ranges [][2]int) string {
	if len(ranges) == 0 {
		return text
	}
	var sb strings.Builder
	prev := 0
	for _, r := range ranges {
		sb.WriteString(text[prev:r[0]])
		prev = r[1]
	}
	sb.WriteString(text[prev:])
	return sb.String()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model/... -v -race -run TestParseToolCalls`
Expected: PASS (all 5 tests)

**Step 5: Commit**

```bash
git add internal/model/claudecode.go internal/model/claudecode_test.go
git commit -m "feat(claudecode): parse <tool_call> blocks from LLM text output"
```

---

### Task 4: Convert FunctionCall/FunctionResponse Parts in message builder

**Files:**
- Modify: `internal/model/claudecode.go`
- Modify: `internal/model/claudecode_test.go`

**Step 1: Write the failing test**

```go
func TestBuildUserMessage_TextOnly(t *testing.T) {
	contents := []*genai.Content{
		genai.NewContentFromText("Hello", genai.RoleUser),
		genai.NewContentFromText("Hi there", "model"),
		genai.NewContentFromText("Generate a blog workflow", genai.RoleUser),
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
		genai.NewContentFromText("Generate a workflow", genai.RoleUser),
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
}

func TestBuildUserMessage_WithFunctionResponse(t *testing.T) {
	contents := []*genai.Content{
		genai.NewContentFromText("Generate a workflow", genai.RoleUser),
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/... -v -race -run TestBuildUserMessage`
Expected: FAIL — `buildUserMessage` undefined

**Step 3: Refactor message building into a named function and add FunctionCall/FunctionResponse handling**

Extract the existing inline message building logic from `generate()` into `buildUserMessage()`, then add support for FunctionCall and FunctionResponse parts:

```go
// buildUserMessage converts ADK content parts into a text conversation for the CLI.
// Handles Text, FunctionCall, and FunctionResponse parts.
func buildUserMessage(contents []*genai.Content) string {
	var sb strings.Builder
	for _, content := range contents {
		for _, part := range content.Parts {
			switch {
			case part.Text != "":
				switch content.Role {
				case "user":
					sb.WriteString(part.Text)
					sb.WriteString("\n")
				case "model":
					sb.WriteString("[Assistant]: ")
					sb.WriteString(part.Text)
					sb.WriteString("\n\n[User]: ")
				}
			case part.FunctionCall != nil:
				fc := part.FunctionCall
				argsJSON, _ := json.Marshal(fc.Args)
				sb.WriteString("[Assistant called tool: ")
				sb.WriteString(fc.Name)
				sb.WriteString("]\nArguments: ")
				sb.Write(argsJSON)
				sb.WriteString("\n\n")
			case part.FunctionResponse != nil:
				fr := part.FunctionResponse
				resultJSON, _ := json.Marshal(fr.Response)
				sb.WriteString("[Tool result: ")
				sb.WriteString(fr.Name)
				sb.WriteString("]\n")
				sb.Write(resultJSON)
				sb.WriteString("\n\n")
			}
		}
	}
	return sb.String()
}
```

Then update `generate()` to call `buildUserMessage(req.Contents)` instead of the inline loop.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/model/... -v -race -run TestBuildUserMessage`
Expected: PASS (all 3 tests)

**Step 5: Also run existing tests to verify no regression**

Run: `go test ./internal/model/... -v -race`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/model/claudecode.go internal/model/claudecode_test.go
git commit -m "feat(claudecode): handle FunctionCall/FunctionResponse in message builder"
```

---

### Task 5: Wire everything into `generate()` — complete tool calling loop

**Files:**
- Modify: `internal/model/claudecode.go`
- Modify: `internal/model/claudecode_test.go`

**Step 1: Write the integration test**

```go
func TestGenerate_BuildsResponseWithToolCalls(t *testing.T) {
	// Test that when text output contains <tool_call> blocks,
	// generate() returns FunctionCall Parts instead of just text.

	// We can't call the CLI in tests, so we test the response building logic.
	// This tests the parseAndBuildResponse helper.
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

func TestGenerate_PlainTextResponse(t *testing.T) {
	text := "Here is your answer: hello world"
	resp := buildToolResponse(text)
	if len(resp.Content.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(resp.Content.Parts))
	}
	if resp.Content.Parts[0].Text != text {
		t.Errorf("expected plain text passthrough")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/model/... -v -race -run TestGenerate_`
Expected: FAIL — `buildToolResponse` undefined

**Step 3: Implement `buildToolResponse` and wire into `generate()`**

```go
// buildToolResponse parses CLI text output and creates an LLMResponse.
// If <tool_call> blocks are found, they become FunctionCall Parts.
// Any surrounding text becomes Text Parts.
func buildToolResponse(text string) *adkmodel.LLMResponse {
	calls, remaining := parseToolCalls(text)

	var parts []*genai.Part

	if len(calls) > 0 {
		// Add remaining text as a text part (if non-empty)
		if remaining != "" {
			parts = append(parts, genai.NewPartFromText(remaining))
		}
		// Add function call parts
		for _, fc := range calls {
			parts = append(parts, &genai.Part{FunctionCall: fc})
		}
	} else {
		// No tool calls — plain text response
		parts = []*genai.Part{genai.NewPartFromText(text)}
	}

	return &adkmodel.LLMResponse{
		Content: &genai.Content{
			Role:  "model",
			Parts: parts,
		},
		TurnComplete: true,
		FinishReason: genai.FinishReasonStop,
	}
}
```

Then update `generate()` to:
1. Call `extractCustomToolDefs(req)` to get custom tool defs
2. If custom defs exist, append `buildToolPrompt(defs)` to `systemPrompt`
3. Use `buildUserMessage(req.Contents)` instead of inline loop
4. Replace the hardcoded text response with `buildToolResponse(text)`
5. Update effort logic: if custom tools present, don't default to "low"

The key changes in `generate()`:

```go
func (c *ClaudeCodeLLM) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	// ... extract system prompt (unchanged) ...

	// Append custom tool definitions to system prompt.
	customDefs := extractCustomToolDefs(req)
	if len(customDefs) > 0 {
		systemPrompt += buildToolPrompt(customDefs)
	}

	// Build user message (now handles FunctionCall/FunctionResponse parts).
	userMessage := buildUserMessage(req.Contents)

	// ... build CLI args (unchanged) ...

	// Effort: don't default to "low" when custom tools are present.
	if effort := effortFromContext(ctx); effort != "" {
		args = append(args, "--effort", effort)
	} else if len(cliTools) > 0 || len(customDefs) > 0 {
		// Tools need at least medium effort.
	} else if !thinkingFromContext(ctx) {
		args = append(args, "--effort", "low")
	}

	// ... timeout: also extend for custom tools ...
	if len(cliTools) > 0 || len(customDefs) > 0 || effortFromContext(ctx) == "high" {
		timeout = 5 * time.Minute
	}

	// ... execute CLI (unchanged) ...

	text := strings.TrimSpace(stdout.String())
	// ... logging, empty check (unchanged) ...

	// Parse response — may contain FunctionCall parts.
	return buildToolResponse(text), nil
}
```

**Step 4: Run all tests**

Run: `go test ./internal/model/... -v -race`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/model/claudecode.go internal/model/claudecode_test.go
git commit -m "feat(claudecode): wire text-based tool calling into generate()"
```

---

### Task 6: Update GenerateContent docstring and run full test suite

**Files:**
- Modify: `internal/model/claudecode.go` (docstring only)

**Step 1: Update the GenerateContent docstring**

Change lines 90-92:

```go
// GenerateContent runs `claude -p` with the given request and returns the response.
// Native tools (e.g., web_search → WebSearch) are mapped to Claude Code CLI tools.
// Custom FunctionDeclarations are supported via text-based tool calling: definitions
// are injected into the system prompt, and <tool_call> blocks in the response are
// parsed into FunctionCall Parts for the handler's tool execution loop.
```

**Step 2: Run the full backend test suite**

Run: `make test`
Expected: PASS — no regressions in other packages

**Step 3: Commit**

```bash
git add internal/model/claudecode.go
git commit -m "docs(claudecode): update GenerateContent docstring for tool calling support"
```

---

### Task 7: Manual integration test with chat endpoint

**This task is manual verification, not automated.**

**Step 1: Restart the backend**

Run: `make dev-backend` (or restart if already running)

**Step 2: Test chat endpoint with curl**

```bash
TK=$(cat /tmp/jwt_token.txt)
curl -s --max-time 60 -X POST http://localhost:8081/api/chat \
  -H "Authorization: Bearer $TK" \
  -H "Content-Type: application/json" \
  -d '{"message":"블로그 글 쓰는 간단한 워크플로우 만들어줘","page":"workflows","context":{"workflow_id":"blog","nodes":[]},"history":[]}' \
  > /tmp/chat_test.txt 2>&1
cat /tmp/chat_test.txt
```

**Step 3: Verify the response contains actual tool_call SSE events**

Expected output should include:
```
event: tool_call
data: {"id":"cc_tool_0","name":"generate_workflow","args":{...}}

event: tool_result
data: {"id":"cc_tool_0","name":"generate_workflow","success":true,"result":{...}}
```

Instead of the previous behavior where `<tool_call>` appeared as text in `text_delta`.

**Step 4: Test in browser**

Navigate to `http://localhost:5173/workflows?w=blog`, type "블로그 글 쓰는 워크플로우 만들어줘" in the chat bar, and verify the workflow gets generated on the canvas.

---

### Summary

| Task | What | Files |
|------|------|-------|
| 1 | Extract custom FunctionDeclarations | claudecode.go, claudecode_test.go |
| 2 | Build tool prompt text | claudecode.go, claudecode_test.go |
| 3 | Parse `<tool_call>` blocks | claudecode.go, claudecode_test.go |
| 4 | Handle FunctionCall/FunctionResponse in messages | claudecode.go, claudecode_test.go |
| 5 | Wire into generate() | claudecode.go, claudecode_test.go |
| 6 | Docstring + full test suite | claudecode.go |
| 7 | Manual integration test | — |

All changes are in `internal/model/claudecode.go` + new test file. Zero changes to handler/executor code.
