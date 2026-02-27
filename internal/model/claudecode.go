package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"

	"github.com/soochol/upal/internal/config"
)

// Compile-time interface compliance check.
var _ adkmodel.LLM = (*ClaudeCodeLLM)(nil)

type thinkingKey struct{}

// WithThinking returns a context that carries the thinking preference.
func WithThinking(ctx context.Context, thinking bool) context.Context {
	return context.WithValue(ctx, thinkingKey{}, thinking)
}

// thinkingFromContext extracts the thinking preference from context.
func thinkingFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(thinkingKey{}).(bool)
	return v
}

type effortKey struct{}

// WithEffort returns a context that carries the effort level for ClaudeCodeLLM.
// Valid values: "low", "high". Empty string means use the default behavior.
func WithEffort(ctx context.Context, effort string) context.Context {
	return context.WithValue(ctx, effortKey{}, effort)
}

// effortFromContext extracts the effort level from context.
func effortFromContext(ctx context.Context) string {
	v, _ := ctx.Value(effortKey{}).(string)
	return v
}

const defaultClaudeBinary = "claude"

// ClaudeCodeLLM implements the ADK model.LLM interface by shelling out to the
// Claude Code CLI (`claude -p`). This allows using a Claude Code subscription
// without an API key.
type ClaudeCodeLLM struct {
	binaryPath string
}

// NewClaudeCodeLLM creates a new ClaudeCodeLLM. It locates the claude binary
// on PATH or uses the provided path. If "claude" is not on PATH, it checks
// common installation locations.
func NewClaudeCodeLLM(binaryPath ...string) *ClaudeCodeLLM {
	bin := defaultClaudeBinary
	if len(binaryPath) > 0 && binaryPath[0] != "" {
		bin = binaryPath[0]
	}
	// If the default name isn't resolvable, check common locations.
	if bin == defaultClaudeBinary {
		if _, err := exec.LookPath(bin); err != nil {
			home, _ := os.UserHomeDir()
			for _, candidate := range []string{
				home + "/.local/bin/claude",
				"/usr/local/bin/claude",
			} {
				if _, err := os.Stat(candidate); err == nil {
					bin = candidate
					break
				}
			}
		}
	}
	return &ClaudeCodeLLM{binaryPath: bin}
}

// Name returns "claude-code".
func (c *ClaudeCodeLLM) Name() string {
	return "claude-code"
}

// GenerateContent runs `claude -p` with the given request and returns the response.
// Native tools (e.g., web_search → WebSearch) are mapped to Claude Code CLI tools.
// Custom FunctionDeclarations are supported via text-based tool calling: definitions
// are injected into the system prompt, and <tool_call> blocks in the response are
// parsed into FunctionCall Parts for the handler's tool execution loop.
func (c *ClaudeCodeLLM) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, stream bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		resp, err := c.generate(ctx, req)
		yield(resp, err)
	}
}

// generate performs a synchronous call to the Claude Code CLI.
func (c *ClaudeCodeLLM) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	// Extract system prompt.
	var systemPrompt string
	if req.Config != nil && req.Config.SystemInstruction != nil {
		for _, part := range req.Config.SystemInstruction.Parts {
			if part.Text != "" {
				systemPrompt = part.Text
				break
			}
		}
	}

	// Append custom tool definitions to system prompt for text-based tool calling.
	customDefs := extractCustomToolDefs(req)
	if len(customDefs) > 0 {
		systemPrompt += buildToolPrompt(customDefs)
	}

	// Build user message (handles Text, FunctionCall, and FunctionResponse parts).
	userMessage := buildUserMessage(req.Contents)

	// Build the command: claude -p --model <model> --tools "" --output-format text
	args := []string{"-p"}

	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	if systemPrompt != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}

	// Map request tools to Claude Code CLI tool names.
	// Native tools (GoogleSearch → WebSearch) are supported;
	// custom FunctionDeclarations are ignored (CLI doesn't support them).
	cliTools := mapToolsToCLI(req)
	if len(cliTools) > 0 {
		args = append(args, "--tools", strings.Join(cliTools, ","))
		args = append(args, "--allowedTools", strings.Join(cliTools, ","))
	} else {
		// Disable all built-in tools for pure text completion.
		args = append(args, "--tools", "")
	}
	args = append(args, "--output-format", "text")
	// Effort level: use explicit context value if set, otherwise default to
	// "low" for fast responses (unless thinking is enabled or tools are active).
	hasTools := len(cliTools) > 0 || len(customDefs) > 0
	if effort := effortFromContext(ctx); effort != "" {
		args = append(args, "--effort", effort)
	} else if hasTools {
		// Tools need at least medium effort to actually trigger usage.
	} else if !thinkingFromContext(ctx) {
		args = append(args, "--effort", "low")
	}

	// Set a timeout for the subprocess.
	// Tools and high-effort (extended thinking) calls need more time.
	timeout := 2 * time.Minute
	if hasTools || effortFromContext(ctx) == "high" {
		timeout = 5 * time.Minute
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	emitLog(ctx, fmt.Sprintf("exec: %s %s", c.binaryPath, strings.Join(args, " ")))

	cmd := exec.CommandContext(execCtx, c.binaryPath, args...)

	// Remove CLAUDECODE env var to avoid nested session detection.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	// Pass user message via stdin.
	cmd.Stdin = strings.NewReader(userMessage)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			emitLog(ctx, fmt.Sprintf("error: %s", errMsg))
			return nil, fmt.Errorf("claude-code: %s: %w", errMsg, err)
		}
		emitLog(ctx, fmt.Sprintf("error: %s", err))
		return nil, fmt.Errorf("claude-code: %w", err)
	}

	text := strings.TrimSpace(stdout.String())
	emitLog(ctx, fmt.Sprintf("response: %d chars", len(text)))
	if text == "" {
		return nil, fmt.Errorf("claude-code: empty response")
	}

	return buildToolResponse(text), nil
}

// mapToolsToCLI maps ADK genai.Tool entries to Claude Code CLI tool names.
// Returns nil if no mappable tools are found.
func mapToolsToCLI(req *adkmodel.LLMRequest) []string {
	if req.Config == nil {
		return nil
	}
	var names []string
	for _, tool := range req.Config.Tools {
		if tool.GoogleSearch != nil {
			names = append(names, "WebSearch")
		}
	}
	return names
}

// ---------------------------------------------------------------------------
// Text-based tool calling helpers
// ---------------------------------------------------------------------------

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

// schemaTypeString returns a human-readable type name for a genai.Schema.
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

// toolCallRe matches <tool_call>{...}</tool_call> blocks in LLM text output.
var toolCallRe = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)

// rawToolCall is used for JSON unmarshaling of tool call blocks.
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

// removeRangesFromText returns text with the specified byte ranges removed.
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
				argsJSON, err := json.Marshal(fc.Args)
				if err != nil {
					argsJSON = []byte("{}")
				}
				sb.WriteString("[Assistant called tool: ")
				sb.WriteString(fc.Name)
				sb.WriteString("]\nArguments: ")
				sb.Write(argsJSON)
				sb.WriteString("\n\n")
			case part.FunctionResponse != nil:
				fr := part.FunctionResponse
				resultJSON, err := json.Marshal(fr.Response)
				if err != nil {
					resultJSON = []byte("{}")
				}
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

// buildToolResponse parses CLI text output and creates an LLMResponse.
// If <tool_call> blocks are found, they become FunctionCall Parts.
// Any surrounding text becomes a Text Part.
func buildToolResponse(text string) *adkmodel.LLMResponse {
	calls, remaining := parseToolCalls(text)

	var parts []*genai.Part

	if len(calls) > 0 {
		// Add remaining text as a text part (if non-empty).
		if remaining != "" {
			parts = append(parts, genai.NewPartFromText(remaining))
		}
		// Add function call parts.
		for _, fc := range calls {
			parts = append(parts, &genai.Part{FunctionCall: fc})
		}
	} else {
		// No tool calls — plain text response.
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

func init() {
	RegisterProvider("claude-code", func(name string, cfg config.ProviderConfig) adkmodel.LLM {
		return NewClaudeCodeLLM()
	})
}

// filterEnv returns a copy of env with any variables matching the given key removed.
func filterEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
