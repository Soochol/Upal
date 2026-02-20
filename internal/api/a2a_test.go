package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/soochol/upal/internal/upal"
)

func TestAgentCardEndpoint(t *testing.T) {
	srv := newTestServer()
	srv.SetA2ABaseURL("http://localhost:8080")
	srv.repo.Create(context.Background(), &upal.WorkflowDefinition{
		Name: "test-workflow",
		Nodes: []upal.NodeDefinition{
			{ID: "user-input", Type: upal.NodeTypeInput, Config: map[string]any{"placeholder": "Enter text"}},
			{ID: "agent-1", Type: upal.NodeTypeAgent, Config: map[string]any{}},
			{ID: "output", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "user-input", To: "agent-1"},
			{From: "agent-1", To: "output"},
		},
	})

	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/.well-known/agent-card.json", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var card a2a.AgentCard
	if err := json.Unmarshal(w.Body.Bytes(), &card); err != nil {
		t.Fatalf("failed to decode agent card: %v", err)
	}

	if card.Name != "Upal" {
		t.Errorf("expected card name 'Upal', got %q", card.Name)
	}
	if card.URL != "http://localhost:8080/a2a" {
		t.Errorf("expected URL 'http://localhost:8080/a2a', got %q", card.URL)
	}
	if len(card.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(card.Skills))
	}
	skill := card.Skills[0]
	if skill.ID != "test-workflow" {
		t.Errorf("expected skill ID 'test-workflow', got %q", skill.ID)
	}
	if skill.Name != "test-workflow" {
		t.Errorf("expected skill name 'test-workflow', got %q", skill.Name)
	}
}

func TestAgentCardDynamic(t *testing.T) {
	srv := newTestServer()
	srv.SetA2ABaseURL("http://localhost:8080")
	ctx := context.Background()

	// Initially empty.
	card := srv.buildAgentCard(ctx)
	if len(card.Skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(card.Skills))
	}

	// Add a workflow.
	srv.repo.Create(ctx, &upal.WorkflowDefinition{
		Name:  "wf-1",
		Nodes: []upal.NodeDefinition{{ID: "in", Type: upal.NodeTypeInput}},
	})
	card = srv.buildAgentCard(ctx)
	if len(card.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(card.Skills))
	}

	// Add another.
	srv.repo.Create(ctx, &upal.WorkflowDefinition{
		Name:  "wf-2",
		Nodes: []upal.NodeDefinition{{ID: "in", Type: upal.NodeTypeInput}},
	})
	card = srv.buildAgentCard(ctx)
	if len(card.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(card.Skills))
	}

	// Delete one.
	srv.repo.Delete(ctx, "wf-1")
	card = srv.buildAgentCard(ctx)
	if len(card.Skills) != 1 {
		t.Fatalf("expected 1 skill after delete, got %d", len(card.Skills))
	}
}

func TestParseA2AMessageJSON(t *testing.T) {
	msg := a2a.NewMessage(a2a.MessageRoleUser,
		a2a.TextPart{Text: `{"workflow": "my-wf", "inputs": {"input-1": "hello"}}`},
	)

	name, inputs, err := parseA2AMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "my-wf" {
		t.Errorf("expected workflow 'my-wf', got %q", name)
	}
	if inputs["input-1"] != "hello" {
		t.Errorf("expected input-1='hello', got %v", inputs["input-1"])
	}
}

func TestParseA2AMessageMetadata(t *testing.T) {
	msg := a2a.NewMessage(a2a.MessageRoleUser,
		a2a.TextPart{Text: "some plain text"},
	)
	msg.Metadata = map[string]any{"workflow": "meta-wf"}

	name, _, err := parseA2AMessage(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "meta-wf" {
		t.Errorf("expected workflow 'meta-wf', got %q", name)
	}
}

func TestParseA2AMessageEmpty(t *testing.T) {
	_, _, err := parseA2AMessage(nil)
	if err == nil {
		t.Fatal("expected error for nil message")
	}

	msg := a2a.NewMessage(a2a.MessageRoleUser)
	_, _, err = parseA2AMessage(msg)
	if err == nil {
		t.Fatal("expected error for empty parts")
	}
}

func TestParseA2AMessageNoWorkflow(t *testing.T) {
	msg := a2a.NewMessage(a2a.MessageRoleUser,
		a2a.TextPart{Text: "just some text without workflow info"},
	)

	_, _, err := parseA2AMessage(msg)
	if err == nil {
		t.Fatal("expected error when no workflow specified")
	}
}

func TestBuildExampleInputs(t *testing.T) {
	result := buildExampleInputs([]string{"name", "age"})
	expected := `"name": "...", "age": "..."`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	result = buildExampleInputs(nil)
	if result != "" {
		t.Errorf("expected empty string for nil, got %q", result)
	}
}
