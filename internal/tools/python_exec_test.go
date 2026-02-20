package tools

import (
	"context"
	"testing"
)

func TestPythonExecTool_SimpleOutput(t *testing.T) {
	tool := &PythonExecTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"code": "print('hello world')",
	})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["exit_code"] != 0 {
		t.Errorf("expected exit_code 0, got %v", m["exit_code"])
	}
	if m["stdout"] != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", m["stdout"])
	}
}

func TestPythonExecTool_Stderr(t *testing.T) {
	tool := &PythonExecTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"code": "import sys; sys.stderr.write('err msg')",
	})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["stderr"] != "err msg" {
		t.Errorf("expected 'err msg', got %q", m["stderr"])
	}
}

func TestPythonExecTool_NonZeroExit(t *testing.T) {
	tool := &PythonExecTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"code": "import sys; sys.exit(1)",
	})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["exit_code"] != 1 {
		t.Errorf("expected exit_code 1, got %v", m["exit_code"])
	}
}

func TestPythonExecTool_EmptyCode(t *testing.T) {
	tool := &PythonExecTool{}
	_, err := tool.Execute(context.Background(), map[string]any{
		"code": "",
	})
	if err == nil {
		t.Error("expected error for empty code")
	}
}

func TestPythonExecTool_Calculation(t *testing.T) {
	tool := &PythonExecTool{}
	result, err := tool.Execute(context.Background(), map[string]any{
		"code": `
def fib(n):
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a

print(fib(10))
`,
	})
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if m["exit_code"] != 0 {
		t.Errorf("expected exit_code 0, got %v", m["exit_code"])
	}
	if m["stdout"] != "55\n" {
		t.Errorf("expected '55\\n', got %q", m["stdout"])
	}
}
