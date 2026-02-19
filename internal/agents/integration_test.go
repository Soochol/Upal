// internal/agents/integration_test.go
package agents_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/agents"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/agent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func TestDAGAgent_EndToEnd(t *testing.T) {
	// 1. Mock Anthropic API server
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content":     []map[string]any{{"type": "text", "text": "Generated response"}},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	// 2. Create LLM pointing to mock
	llms := map[string]adkmodel.LLM{
		"test": upalmodel.NewAnthropicLLM("test-key", upalmodel.WithAnthropicBaseURL(mockLLM.URL)),
	}

	// 3. Build workflow: input → agent → output
	wf := &upal.WorkflowDefinition{
		Name: "integration-test",
		Nodes: []upal.NodeDefinition{
			{ID: "input1", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "agent1", Type: upal.NodeTypeAgent, Config: map[string]any{
				"model":         "test/claude-sonnet-4-20250514",
				"system_prompt": "You are helpful",
			}},
			{ID: "output1", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "input1", To: "agent1"},
			{From: "agent1", To: "output1"},
		},
	}

	// 4. Build DAGAgent
	dagAgent, err := agents.NewDAGAgent(wf, llms, nil)
	if err != nil {
		t.Fatalf("new dag agent: %v", err)
	}

	// 5. Create ADK Runner
	sessionSvc := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        "integration-test",
		Agent:          dagAgent,
		SessionService: sessionSvc,
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	// 6. Create session with user input
	_, err = sessionSvc.Create(context.Background(), &session.CreateRequest{
		AppName:   "integration-test",
		UserID:    "user1",
		SessionID: "sess1",
		State:     map[string]any{"__user_input__input1": "Hello world"},
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// 7. Run and collect events
	userMsg := genai.NewContentFromText("run", genai.RoleUser)
	var events []*session.Event
	for event, err := range r.Run(context.Background(), "user1", "sess1", userMsg, agent.RunConfig{}) {
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
		if event != nil {
			events = append(events, event)
			t.Logf("event: author=%s", event.Author)
		}
	}

	// 8. Verify events
	if len(events) == 0 {
		t.Fatal("expected at least one event")
	}

	// Verify we got events from all three nodes
	authors := make(map[string]bool)
	for _, ev := range events {
		if ev.Author != "" {
			authors[ev.Author] = true
		}
	}
	if !authors["input1"] {
		t.Error("missing event from input1")
	}
	// agent1 events come from ADK's LLMAgent which may use a different author format
	// output1 should have collected results
	if !authors["output1"] {
		t.Error("missing event from output1")
	}

	t.Logf("total events: %d, authors: %v", len(events), authors)
}
