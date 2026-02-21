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
	gen := New(llm, "gpt-4o", nil, nil, nil)
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
	gen := New(llm, "gpt-4o", nil, nil, nil)
	result, err := gen.Generate(context.Background(), "Simple workflow", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Name != "fenced-wf" {
		t.Errorf("name: got %q", result.Name)
	}
}

func TestGenerate_LeadingTemplateText(t *testing.T) {
	// Reproduces the bug where LLM returns explanatory text containing
	// {{node_id}} template references before the actual JSON.
	wf := upal.WorkflowDefinition{
		Name:    "blog-writer",
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "in", To: "out"}},
	}
	wfJSON, _ := json.Marshal(wf)
	// Simulate LLM returning Korean explanation with {{template}} refs before JSON
	withLeading := "{{node_id}} 템플릿 참조를 통해 데이터가 흐릅니다.\n\n" + string(wfJSON)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": withLeading}, "finish_reason": "stop"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := upalmodel.NewOpenAILLM("test-key", upalmodel.WithOpenAIBaseURL(server.URL))
	gen := New(llm, "gpt-4o", nil, nil, nil)
	result, err := gen.Generate(context.Background(), "Create a blog writer", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Name != "blog-writer" {
		t.Errorf("name: got %q, want blog-writer", result.Name)
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
	gen := New(llm, "gpt-4o", nil, nil, nil)
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
	gen := New(llm, "gpt-4o", nil, nil, nil)
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

func TestStripInvalidTools(t *testing.T) {
	gen := &Generator{toolNames: []string{"web_search", "real_tool"}}

	wf := &upal.WorkflowDefinition{
		Name: "strip-test",
		Nodes: []upal.NodeDefinition{
			{ID: "a1", Type: upal.NodeTypeAgent, Config: map[string]any{
				"tools": []any{"real_tool", "hallucinated_tool", "web_search"},
			}},
			{ID: "a2", Type: upal.NodeTypeAgent, Config: map[string]any{
				"tools": []any{"fake_only"},
			}},
			{ID: "a3", Type: upal.NodeTypeAgent, Config: map[string]any{
				"prompt": "no tools here",
			}},
		},
	}

	gen.stripInvalidTools(wf)

	// a1: hallucinated_tool removed, real_tool and web_search kept.
	tools1, _ := wf.Nodes[0].Config["tools"].([]any)
	if len(tools1) != 2 {
		t.Fatalf("a1 tools: got %d, want 2: %v", len(tools1), tools1)
	}

	// a2: all tools invalid → "tools" key removed entirely.
	if _, exists := wf.Nodes[1].Config["tools"]; exists {
		t.Fatal("a2: expected tools key to be removed")
	}

	// a3: no tools config → unchanged.
	if _, exists := wf.Nodes[2].Config["tools"]; exists {
		t.Fatal("a3: should not have tools key")
	}
}

func TestStripInvalidNodeTypes(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "strip-types-test",
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "bogus", Type: "unknown", Config: map[string]any{}},
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{"model": "openai/gpt-4o", "prompt": "hi"}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "in", To: "bogus"},
			{From: "bogus", To: "agent1"},
			{From: "agent1", To: "out"},
		},
	}

	stripInvalidNodeTypes(wf)

	// bogus removed; edges referencing bogus removed.
	if len(wf.Nodes) != 3 {
		t.Fatalf("nodes: got %d, want 3", len(wf.Nodes))
	}
	for _, n := range wf.Nodes {
		if n.Type == "unknown" {
			t.Fatalf("unknown node should have been stripped")
		}
	}
	// Only agent1→out edge survives.
	if len(wf.Edges) != 1 {
		t.Fatalf("edges: got %d, want 1", len(wf.Edges))
	}
	if wf.Edges[0].From != "agent1" || wf.Edges[0].To != "out" {
		t.Fatalf("unexpected surviving edge: %+v", wf.Edges[0])
	}
}

func TestValidate_RejectsUnknownType(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "unknown-reject",
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "bogus", Type: "unknown", Config: map[string]any{}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "in", To: "bogus"},
			{From: "bogus", To: "out"},
		},
	}
	err := validate(wf)
	if err == nil {
		t.Fatal("expected error for unknown node type")
	}
}

func TestValidate_AgentMissingModel(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "no-model",
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "a", Type: upal.NodeTypeAgent, Config: map[string]any{"prompt": "test"}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "in", To: "a"}, {From: "a", To: "out"}},
	}
	if err := validate(wf); err == nil {
		t.Fatal("expected error for agent missing model")
	}
}

func TestFixInvalidModels(t *testing.T) {
	gen := &Generator{
		model: "sonnet",
		models: []ModelOption{
			{ID: "claude/sonnet", Tier: "mid", Hint: "balanced"},
			{ID: "claude/haiku", Tier: "low", Hint: "fast"},
			{ID: "gemini/gemini-2.0-flash", Tier: "low", Hint: "fast"},
		},
	}

	wf := &upal.WorkflowDefinition{
		Name: "fix-models-test",
		Nodes: []upal.NodeDefinition{
			{ID: "a1", Type: upal.NodeTypeAgent, Config: map[string]any{
				"model": "claude/sonnet", "prompt": "valid model",
			}},
			{ID: "a2", Type: upal.NodeTypeAgent, Config: map[string]any{
				"model": "anthropic/claude-sonnet-4-6", "prompt": "hallucinated model",
			}},
			{ID: "a3", Type: upal.NodeTypeAgent, Config: map[string]any{
				"model": "gemini/gemini-2.0-flash", "prompt": "valid model",
			}},
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
		},
	}

	gen.fixInvalidModels(wf)

	// a1: valid → unchanged
	if m := wf.Nodes[0].Config["model"]; m != "claude/sonnet" {
		t.Errorf("a1 model: got %q, want claude/sonnet", m)
	}
	// a2: invalid → replaced with default (claude/sonnet, matched via g.model="sonnet")
	if m := wf.Nodes[1].Config["model"]; m != "claude/sonnet" {
		t.Errorf("a2 model: got %q, want claude/sonnet", m)
	}
	// a3: valid → unchanged
	if m := wf.Nodes[2].Config["model"]; m != "gemini/gemini-2.0-flash" {
		t.Errorf("a3 model: got %q, want gemini/gemini-2.0-flash", m)
	}
	// in: input node → untouched
	if _, exists := wf.Nodes[3].Config["model"]; exists {
		t.Fatal("input node should not have model field")
	}
}

func TestValidate_AgentMissingPrompt(t *testing.T) {
	wf := &upal.WorkflowDefinition{
		Name: "no-prompt",
		Nodes: []upal.NodeDefinition{
			{ID: "in", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "a", Type: upal.NodeTypeAgent, Config: map[string]any{"model": "openai/gpt-4o"}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{{From: "in", To: "a"}, {From: "a", To: "out"}},
	}
	if err := validate(wf); err == nil {
		t.Fatal("expected error for agent missing prompt")
	}
}

func makeStage(id, typ string) upal.Stage {
	return upal.Stage{ID: id, Name: id, Type: typ, Config: upal.StageConfig{}}
}

func TestStripInvalidStageTypes(t *testing.T) {
	stages := []upal.Stage{
		makeStage("s1", "workflow"),
		makeStage("s2", "notification"), // hallucinated
		makeStage("s3", "approval"),
		makeStage("s4", "custom-thing"), // hallucinated
		makeStage("s5", "trigger"),
	}
	result := stripInvalidStageTypes(stages)
	if len(result) != 3 {
		t.Fatalf("expected 3 valid stages, got %d", len(result))
	}
	for _, s := range result {
		if !validStageTypes[s.Type] {
			t.Errorf("invalid stage type %q should have been stripped", s.Type)
		}
	}
}

func TestApplyDelta_PreservesUnchangedStages(t *testing.T) {
	existing := &upal.Pipeline{
		Name: "my-pipeline",
		Stages: []upal.Stage{
			makeStage("stage-1", "workflow"),
			makeStage("stage-2", "approval"),
			makeStage("stage-3", "schedule"),
		},
	}
	delta := &PipelineEditDelta{
		StageChanges: []PipelineStageDelta{
			{Op: "update", Stage: &upal.Stage{ID: "stage-2", Name: "updated-approval", Type: "approval"}},
		},
	}
	result := applyDelta(existing, delta)
	if len(result.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(result.Stages))
	}
	if result.Stages[0].ID != "stage-1" || result.Stages[0].Name != "stage-1" {
		t.Errorf("stage-1 should be unchanged, got %+v", result.Stages[0])
	}
	if result.Stages[1].Name != "updated-approval" {
		t.Errorf("stage-2 should be updated, got %+v", result.Stages[1])
	}
	if result.Stages[2].ID != "stage-3" || result.Stages[2].Name != "stage-3" {
		t.Errorf("stage-3 should be unchanged, got %+v", result.Stages[2])
	}
}

func TestApplyDelta_AddStage(t *testing.T) {
	existing := &upal.Pipeline{
		Stages: []upal.Stage{makeStage("stage-1", "workflow")},
	}
	newStage := makeStage("stage-2", "approval")
	delta := &PipelineEditDelta{
		StageChanges: []PipelineStageDelta{
			{Op: "add", Stage: &newStage},
		},
	}
	result := applyDelta(existing, delta)
	if len(result.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(result.Stages))
	}
	if result.Stages[1].ID != "stage-2" {
		t.Errorf("expected added stage at end, got %+v", result.Stages[1])
	}
}

func TestApplyDelta_RemoveStage(t *testing.T) {
	existing := &upal.Pipeline{
		Stages: []upal.Stage{
			makeStage("stage-1", "workflow"),
			makeStage("stage-2", "approval"),
			makeStage("stage-3", "schedule"),
		},
	}
	delta := &PipelineEditDelta{
		StageChanges: []PipelineStageDelta{
			{Op: "remove", StageID: "stage-2"},
		},
	}
	result := applyDelta(existing, delta)
	if len(result.Stages) != 2 {
		t.Fatalf("expected 2 stages after removal, got %d", len(result.Stages))
	}
	for _, s := range result.Stages {
		if s.ID == "stage-2" {
			t.Error("stage-2 should have been removed")
		}
	}
}

func TestApplyDelta_EmptyDeltaPreservesAll(t *testing.T) {
	existing := &upal.Pipeline{
		Name: "keep-me",
		Stages: []upal.Stage{
			makeStage("stage-1", "workflow"),
			makeStage("stage-2", "approval"),
		},
	}
	delta := &PipelineEditDelta{}
	result := applyDelta(existing, delta)
	if result.Name != "keep-me" {
		t.Errorf("name should be preserved, got %q", result.Name)
	}
	if len(result.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(result.Stages))
	}
}

func TestApplyDelta_Reorder(t *testing.T) {
	existing := &upal.Pipeline{
		Stages: []upal.Stage{
			makeStage("stage-1", "workflow"),
			makeStage("stage-2", "approval"),
			makeStage("stage-3", "schedule"),
		},
	}
	delta := &PipelineEditDelta{
		StageOrder: []string{"stage-3", "stage-1", "stage-2"},
	}
	result := applyDelta(existing, delta)
	if len(result.Stages) != 3 {
		t.Fatalf("expected 3 stages, got %d", len(result.Stages))
	}
	wantOrder := []string{"stage-3", "stage-1", "stage-2"}
	for i, want := range wantOrder {
		if result.Stages[i].ID != want {
			t.Errorf("position %d: want %q, got %q", i, want, result.Stages[i].ID)
		}
	}
}

func TestApplyDelta_ReorderAndUpdate(t *testing.T) {
	existing := &upal.Pipeline{
		Stages: []upal.Stage{
			makeStage("stage-1", "workflow"),
			makeStage("stage-2", "approval"),
		},
	}
	updated := upal.Stage{ID: "stage-2", Name: "updated", Type: "approval", Config: upal.StageConfig{Message: "hi"}}
	delta := &PipelineEditDelta{
		StageChanges: []PipelineStageDelta{
			{Op: "update", Stage: &updated},
		},
		StageOrder: []string{"stage-2", "stage-1"},
	}
	result := applyDelta(existing, delta)
	if result.Stages[0].ID != "stage-2" || result.Stages[0].Name != "updated" {
		t.Errorf("expected updated stage-2 first, got %+v", result.Stages[0])
	}
	if result.Stages[1].ID != "stage-1" {
		t.Errorf("expected stage-1 second, got %+v", result.Stages[1])
	}
}
