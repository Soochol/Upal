package services

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func TestLookup_Found(t *testing.T) {
	repo := repository.NewMemory()
	repo.Create(context.Background(), &upal.WorkflowDefinition{Name: "test-wf"})

	svc := NewWorkflowService(repo, nil, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", nil)
	wf, err := svc.Lookup(context.Background(), "test-wf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.Name != "test-wf" {
		t.Errorf("expected name 'test-wf', got %q", wf.Name)
	}
}

func TestLookup_NotFound(t *testing.T) {
	repo := repository.NewMemory()
	svc := NewWorkflowService(repo, nil, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", nil)

	_, err := svc.Lookup(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
}

func TestValidate_NoModel(t *testing.T) {
	svc := NewWorkflowService(repository.NewMemory(), nil, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", nil)

	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{}},
		},
	}
	err := svc.Validate(wf)
	if err == nil {
		t.Fatal("expected validation error for agent without model")
	}
}

func TestValidate_InvalidModelFormat(t *testing.T) {
	llms := map[string]adkmodel.LLM{}
	resolver := llmutil.NewMapResolver(llms, nil, "")
	svc := NewWorkflowService(repository.NewMemory(), llms, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", resolver)

	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{"model": "no-slash"}},
		},
	}
	err := svc.Validate(wf)
	if err == nil {
		t.Fatal("expected validation error for bad model format")
	}
}

func TestValidate_UnknownProvider(t *testing.T) {
	llms := map[string]adkmodel.LLM{"anthropic": nil}
	resolver := llmutil.NewMapResolver(llms, nil, "")
	svc := NewWorkflowService(repository.NewMemory(), llms, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", resolver)

	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{"model": "unknown/model"}},
		},
	}
	err := svc.Validate(wf)
	if err == nil {
		t.Fatal("expected validation error for unknown provider")
	}
}

func TestValidate_Valid(t *testing.T) {
	llms := map[string]adkmodel.LLM{"anthropic": nil}
	resolver := llmutil.NewMapResolver(llms, nil, "")
	svc := NewWorkflowService(repository.NewMemory(), llms, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", resolver)

	wf := &upal.WorkflowDefinition{
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{"model": "anthropic/claude-sonnet"}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
	}
	err := svc.Validate(wf)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestClassifyEvent_PrefersStateDelta(t *testing.T) {
	ev := session.NewEvent("inv-1")
	ev.Author = "node-1"
	ev.LLMResponse = adkmodel.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{{Text: "full LLM response"}},
		},
	}
	ev.Actions.StateDelta["node-1"] = "extracted artifact"

	we := classifyEvent(ev)
	if we.Type != "node_completed" {
		t.Fatalf("expected node_completed, got %s", we.Type)
	}
	output, _ := we.Payload["output"].(string)
	if output != "extracted artifact" {
		t.Fatalf("expected stateDelta artifact %q, got %q", "extracted artifact", output)
	}
}

func TestClassifyEvent_FallsBackToExtractContent(t *testing.T) {
	ev := session.NewEvent("inv-1")
	ev.Author = "node-1"
	ev.LLMResponse = adkmodel.LLMResponse{
		Content: &genai.Content{
			Parts: []*genai.Part{{Text: "full response"}},
		},
	}
	// No stateDelta for "node-1"

	we := classifyEvent(ev)
	if we.Type != "node_completed" {
		t.Fatalf("expected node_completed, got %s", we.Type)
	}
	output, _ := we.Payload["output"].(string)
	if output != "full response" {
		t.Fatalf("expected LLMResponse text %q, got %q", "full response", output)
	}
}

func TestClassifyEvent_FlushEvent_UsesStateDelta(t *testing.T) {
	ev := session.NewEvent("inv-1")
	ev.Author = "node-1"
	ev.LLMResponse = adkmodel.LLMResponse{
		Content:      nil,
		FinishReason: genai.FinishReasonStop,
	}
	ev.Actions.StateDelta["node-1"] = "artifact"

	we := classifyEvent(ev)
	if we.Type != "node_completed" {
		t.Fatalf("flush event should be node_completed, got %s", we.Type)
	}
	output, _ := we.Payload["output"].(string)
	if output != "artifact" {
		t.Fatalf("expected artifact in flush payload, got %q", output)
	}
}

func TestRun_InputOutput(t *testing.T) {
	repo := repository.NewMemory()
	svc := NewWorkflowService(repo, nil, session.InMemoryService(), nil, agents.DefaultRegistry(), "", "", nil)

	wf := &upal.WorkflowDefinition{
		Name: "run-test",
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "input1", To: "output1"},
		},
	}

	events, result, err := svc.Run(context.Background(), wf, map[string]any{"input1": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drain events.
	var eventTypes []string
	for ev := range events {
		eventTypes = append(eventTypes, ev.Type)
	}

	// Check we got events.
	if len(eventTypes) == 0 {
		t.Fatal("expected at least one event")
	}

	// Check result.
	res := <-result
	if res.SessionID == "" {
		t.Error("expected non-empty session ID")
	}
}
