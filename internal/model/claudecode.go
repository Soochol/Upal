package model

import (
	"bytes"
	"context"
	"fmt"
	"iter"
	"os"
	"os/exec"
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

// GenerateContent runs `claude -p` with the given request and returns the text response.
// Native tools (e.g., web_search → WebSearch) are mapped to Claude Code CLI tools.
// Custom function declarations are not supported — only native tool mappings.
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

	// Build user message from contents.
	var userMessage strings.Builder
	for _, content := range req.Contents {
		role := content.Role
		for _, part := range content.Parts {
			if part.Text != "" {
				switch role {
				case "user":
					userMessage.WriteString(part.Text)
					userMessage.WriteString("\n")
				case "model":
					userMessage.WriteString("[Assistant]: ")
					userMessage.WriteString(part.Text)
					userMessage.WriteString("\n\n[User]: ")
				}
			}
		}
	}

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
	if effort := effortFromContext(ctx); effort != "" {
		args = append(args, "--effort", effort)
	} else if len(cliTools) > 0 {
		// Tools need at least medium effort to actually trigger usage.
	} else if !thinkingFromContext(ctx) {
		args = append(args, "--effort", "low")
	}

	// Set a timeout for the subprocess.
	// Tools (e.g., WebSearch) and high-effort (extended thinking) calls need more time.
	timeout := 2 * time.Minute
	if len(cliTools) > 0 || effortFromContext(ctx) == "high" {
		timeout = 5 * time.Minute
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	emitLog(ctx, fmt.Sprintf("exec: %s %s", c.binaryPath, strings.Join(args, " ")))

	cmd := exec.CommandContext(execCtx, c.binaryPath, args...)

	// Remove CLAUDECODE env var to avoid nested session detection.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")

	// Pass user message via stdin.
	cmd.Stdin = strings.NewReader(userMessage.String())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		emitLog(ctx, fmt.Sprintf("error: %s", errMsg))
		return nil, fmt.Errorf("claude-code: %s", errMsg)
	}

	text := strings.TrimSpace(stdout.String())
	emitLog(ctx, fmt.Sprintf("response: %d chars", len(text)))
	if text == "" {
		return nil, fmt.Errorf("claude-code: empty response")
	}

	return &adkmodel.LLMResponse{
		Content: &genai.Content{
			Role:  "model",
			Parts: []*genai.Part{genai.NewPartFromText(text)},
		},
		TurnComplete: true,
		FinishReason: genai.FinishReasonStop,
	}, nil
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
