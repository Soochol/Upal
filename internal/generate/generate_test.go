package generate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
)

func TestGenerate_Success(t *testing.T) {
	// Mock LLM returns a valid workflow JSON.
	wf := upal.WorkflowDefinition{
		Name:    "test-workflow",
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "user_input", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "summarizer", Type: upal.NodeTypeAgent, Config: map[string]any{"model": "openai/gpt-4o", "prompt": "Summarize {{user_input}}"}},
			{ID: "final_output", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "user_input", To: "summarizer"},
			{From: "summarizer", To: "final_output"},
		},
	}
	wfJSON, _ := json.Marshal(wf)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": string(wfJSON)}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := upalmodel.NewOpenAILLM("test-key", upalmodel.WithOpenAIBaseURL(server.URL))
	gen := New(llm, "gpt-4o")
	result, err := gen.Generate(context.Background(), "Create a summarizer workflow", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Name != "test-workflow" {
		t.Errorf("name: got %q, want test-workflow", result.Name)
	}
	if len(result.Nodes) != 3 {
		t.Errorf("nodes: got %d, want 3", len(result.Nodes))
	}
	if len(result.Edges) != 2 {
		t.Errorf("edges: got %d, want 2", len(result.Edges))
	}
}

func TestGenerate_StripMarkdownFences(t *testing.T) {
	wf := upal.WorkflowDefinition{
		Name:    "fenced-wf",
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "in", To: "out"}},
	}
	wfJSON, _ := json.Marshal(wf)
	fenced := "```json\n" + string(wfJSON) + "\n```"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": fenced}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := upalmodel.NewOpenAILLM("test-key", upalmodel.WithOpenAIBaseURL(server.URL))
	gen := New(llm, "gpt-4o")
	result, err := gen.Generate(context.Background(), "Simple workflow", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Name != "fenced-wf" {
		t.Errorf("name: got %q", result.Name)
	}
}

func TestGenerate_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "not valid json at all"}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := upalmodel.NewOpenAILLM("test-key", upalmodel.WithOpenAIBaseURL(server.URL))
	gen := New(llm, "gpt-4o")
	_, err := gen.Generate(context.Background(), "Something", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON output")
	}
}

func TestGenerate_ValidationError_NoInput(t *testing.T) {
	wf := upal.WorkflowDefinition{
		Name:    "bad-wf",
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
	}
	wfJSON, _ := json.Marshal(wf)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": string(wfJSON)}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := upalmodel.NewOpenAILLM("test-key", upalmodel.WithOpenAIBaseURL(server.URL))
	gen := New(llm, "gpt-4o")
	_, err := gen.Generate(context.Background(), "Describe something", nil)
	if err == nil {
		t.Fatal("expected validation error for missing input node")
	}
}

func TestValidate_DuplicateNodeID(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "dup-test",
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput},
			{ID: "in", Type: upal.NodeTypeOutput},
		},
	}
	err := validate(wf)
	if err == nil {
		t.Fatal("expected error for duplicate node ID")
	}
}

func TestValidate_BadEdge(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "edge-test",
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput},
			{ID: "out", Type: upal.NodeTypeOutput},
		},
		Edges: []upal.EdgeDefinition{{From: "in", To: "nonexistent"}},
	}
	err := validate(wf)
	if err == nil {
		t.Fatal("expected error for bad edge target")
	}
}
