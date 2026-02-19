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
)

// Compile-time interface compliance check.
var _ adkmodel.LLM = (*ClaudeCodeLLM)(nil)

const defaultClaudeBinary = "claude"

// ClaudeCodeLLM implements the ADK model.LLM interface by shelling out to the
// Claude Code CLI (`claude -p`). This allows using a Claude Code subscription
// without an API key.
type ClaudeCodeLLM struct {
	binaryPath string
}

// NewClaudeCodeLLM creates a new ClaudeCodeLLM. It locates the claude binary
// on PATH or uses the provided path.
func NewClaudeCodeLLM(binaryPath ...string) *ClaudeCodeLLM {
	bin := defaultClaudeBinary
	if len(binaryPath) > 0 && binaryPath[0] != "" {
		bin = binaryPath[0]
	}
	return &ClaudeCodeLLM{binaryPath: bin}
}

// Name returns "claude-code".
func (c *ClaudeCodeLLM) Name() string {
	return "claude-code"
}

// GenerateContent runs `claude -p` with the given request and returns the text response.
// Tool calling is not supported — the response will always be plain text.
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

	// Disable all built-in tools for pure text completion.
	args = append(args, "--tools", "")
	args = append(args, "--output-format", "text")

	// Set a timeout for the subprocess.
	timeout := 2 * time.Minute
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
		return nil, fmt.Errorf("claude-code: %s", errMsg)
	}

	text := strings.TrimSpace(stdout.String())
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
