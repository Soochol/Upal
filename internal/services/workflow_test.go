package services

import (
	"context"
	"testing"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
)

func TestLookup_Found(t *testing.T) {
	repo := repository.NewMemory()
	repo.Create(context.Background(), &upal.WorkflowDefinition{Name: "test-wf"})

	svc := NewWorkflowService(repo, nil, session.InMemoryService(), nil, agents.DefaultRegistry())
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
	svc := NewWorkflowService(repo, nil, session.InMemoryService(), nil, agents.DefaultRegistry())

	_, err := svc.Lookup(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
}

func TestValidate_NoModel(t *testing.T) {
	svc := NewWorkflowService(repository.NewMemory(), nil, session.InMemoryService(), nil, agents.DefaultRegistry())

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
	svc := NewWorkflowService(repository.NewMemory(), nil, session.InMemoryService(), nil, agents.DefaultRegistry())

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
	svc := NewWorkflowService(repository.NewMemory(), llms, session.InMemoryService(), nil, agents.DefaultRegistry())

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
	svc := NewWorkflowService(repository.NewMemory(), llms, session.InMemoryService(), nil, agents.DefaultRegistry())

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

func TestRun_InputOutput(t *testing.T) {
	repo := repository.NewMemory()
	svc := NewWorkflowService(repo, nil, session.InMemoryService(), nil, agents.DefaultRegistry())

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
