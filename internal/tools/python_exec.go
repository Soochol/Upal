package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// maxOutputSize caps stdout/stderr returned to the LLM.
const maxOutputSize = 100 * 1024 // 100 KB

// PythonExecTool executes Python code and returns stdout/stderr.
type PythonExecTool struct{}

func (p *PythonExecTool) Name() string { return "python_exec" }

func (p *PythonExecTool) Description() string {
	return "Execute Python code and return stdout and stderr. Use this to run calculations, process data, or perform any task that benefits from code execution."
}

func (p *PythonExecTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "Python source code to execute",
			},
		},
		"required": []any{"code"},
	}
}

func (p *PythonExecTool) Execute(ctx context.Context, input any) (any, error) {
	args, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input: expected object")
	}

	code, _ := args["code"].(string)
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	// Write code to temp file
	tmpFile, err := os.CreateTemp("", "upal-python-*.py")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write code: %w", err)
	}
	tmpFile.Close()

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "python3", tmpFile.Name())

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute python: %w", err)
		}
	}

	// Cap output sizes
	stdoutStr := stdout.String()
	if len(stdoutStr) > maxOutputSize {
		stdoutStr = stdoutStr[:maxOutputSize] + "\n... [truncated at 100KB]"
	}
	stderrStr := stderr.String()
	if len(stderrStr) > maxOutputSize {
		stderrStr = stderrStr[:maxOutputSize] + "\n... [truncated at 100KB]"
	}

	return map[string]any{
		"stdout":    stdoutStr,
		"stderr":    stderrStr,
		"exit_code": exitCode,
	}, nil
}
