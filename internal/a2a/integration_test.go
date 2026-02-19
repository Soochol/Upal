package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

// mockAgentExecutor returns a response based on state content.
type mockAgentExecutor struct{}

func (m *mockAgentExecutor) Execute(ctx context.Context, def *engine.NodeDefinition, state map[string]any) (any, error) {
	if def.Type == engine.NodeTypeInput {
		key := "__user_input__" + def.ID
		if v, ok := state[key]; ok {
			return v, nil
		}
		return "no input", nil
	}
	if msg, ok := state["__a2a_message__"]; ok {
		return "Processed: " + msg.(string), nil
	}
	return "agent output", nil
}

func TestIntegration_LinearWorkflow(t *testing.T) {
	wf := &engine.WorkflowDefinition{
		Name: "test-pipeline",
		Nodes: []engine.NodeDefinition{
			{ID: "input", Type: engine.NodeTypeInput},
			{ID: "agent", Type: engine.NodeTypeAgent, Config: map[string]any{"model": "test/model"}},
			{ID: "output", Type: engine.NodeTypeOutput},
		},
		Edges: []engine.EdgeDefinition{
			{From: "input", To: "agent"},
			{From: "agent", To: "output"},
		},
	}

	executor := &mockAgentExecutor{}
	r := chi.NewRouter()
	nodeDefs := make([]*engine.NodeDefinition, len(wf.Nodes))
	for i := range wf.Nodes {
		nodeDefs[i] = &wf.Nodes[i]
	}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:  executor,
		engine.NodeTypeAgent:  executor,
		engine.NodeTypeOutput: executor,
	}
	MountA2ARoutes(r, nodeDefs, executors, "http://localhost")
	server := httptest.NewServer(r)
	defer server.Close()

	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()
	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(eventBus, sessions, a2aClient)

	nodeURLs := map[string]string{
		"input":  server.URL + "/a2a/nodes/input",
		"agent":  server.URL + "/a2a/nodes/agent",
		"output": server.URL + "/a2a/nodes/output",
	}

	sess, err := a2aRunner.Run(context.Background(), wf, nodeURLs, map[string]any{"input": "hello world"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != engine.SessionCompleted {
		t.Errorf("status: got %q, want %q", sess.Status, engine.SessionCompleted)
	}
	if len(sess.Artifacts) == 0 {
		t.Error("expected artifacts in session")
	}
}

func TestIntegration_ParallelWorkflow(t *testing.T) {
	wf := &engine.WorkflowDefinition{
		Name: "parallel-test",
		Nodes: []engine.NodeDefinition{
			{ID: "input", Type: engine.NodeTypeInput},
			{ID: "agent_a", Type: engine.NodeTypeAgent},
			{ID: "agent_b", Type: engine.NodeTypeAgent},
			{ID: "output", Type: engine.NodeTypeOutput},
		},
		Edges: []engine.EdgeDefinition{
			{From: "input", To: "agent_a"},
			{From: "input", To: "agent_b"},
			{From: "agent_a", To: "output"},
			{From: "agent_b", To: "output"},
		},
	}

	executor := &mockAgentExecutor{}
	r := chi.NewRouter()
	nodeDefs := make([]*engine.NodeDefinition, len(wf.Nodes))
	for i := range wf.Nodes {
		nodeDefs[i] = &wf.Nodes[i]
	}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:  executor,
		engine.NodeTypeAgent:  executor,
		engine.NodeTypeOutput: executor,
	}
	MountA2ARoutes(r, nodeDefs, executors, "http://localhost")
	server := httptest.NewServer(r)
	defer server.Close()

	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()
	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(eventBus, sessions, a2aClient)

	nodeURLs := map[string]string{
		"input":   server.URL + "/a2a/nodes/input",
		"agent_a": server.URL + "/a2a/nodes/agent_a",
		"agent_b": server.URL + "/a2a/nodes/agent_b",
		"output":  server.URL + "/a2a/nodes/output",
	}

	sess, err := a2aRunner.Run(context.Background(), wf, nodeURLs, map[string]any{"input": "test input"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != engine.SessionCompleted {
		t.Errorf("status: got %q", sess.Status)
	}
	if len(sess.Artifacts) < 3 {
		t.Errorf("expected at least 3 nodes with artifacts, got %d", len(sess.Artifacts))
	}
}

func TestIntegration_AgentCard_Discovery(t *testing.T) {
	wf := &engine.WorkflowDefinition{
		Name: "card-test",
		Nodes: []engine.NodeDefinition{
			{ID: "agent1", Type: engine.NodeTypeAgent, Config: map[string]any{"model": "test/model", "system_prompt": "You are helpful"}},
		},
	}
	r := chi.NewRouter()
	nodeDefs := []*engine.NodeDefinition{&wf.Nodes[0]}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeAgent: &mockAgentExecutor{},
	}
	MountA2ARoutes(r, nodeDefs, executors, "http://localhost:8080")
	server := httptest.NewServer(r)
	defer server.Close()

	resp, err := http.Get(server.URL + "/a2a/nodes/agent1/agent-card")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status: %d", resp.StatusCode)
	}
	var card a2atypes.AgentCard
	json.NewDecoder(resp.Body).Decode(&card)
	if card.Name != "agent1" {
		t.Errorf("name: got %q", card.Name)
	}
	if card.Description != "You are helpful" {
		t.Errorf("description: got %q", card.Description)
	}
}

func TestIntegration_ExternalNode(t *testing.T) {
	// Mock external A2A agent
	externalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		task := a2atypes.Task{
			ID:     "ext-task-1",
			Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("translated: hello")},
				Index: 0,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0", ID: req.ID, Result: task,
		})
	}))
	defer externalServer.Close()

	wf := &engine.WorkflowDefinition{
		Name: "ext-test",
		Nodes: []engine.NodeDefinition{
			{ID: "input", Type: engine.NodeTypeInput},
			{ID: "translate", Type: engine.NodeTypeExternal, Config: map[string]any{
				"endpoint_url": externalServer.URL,
			}},
			{ID: "output", Type: engine.NodeTypeOutput},
		},
		Edges: []engine.EdgeDefinition{
			{From: "input", To: "translate"},
			{From: "translate", To: "output"},
		},
	}

	executor := &mockAgentExecutor{}
	localRouter := chi.NewRouter()
	nodeDefs := make([]*engine.NodeDefinition, 0)
	for i, n := range wf.Nodes {
		if n.Type != engine.NodeTypeExternal {
			nodeDefs = append(nodeDefs, &wf.Nodes[i])
		}
	}
	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:  executor,
		engine.NodeTypeOutput: executor,
	}
	MountA2ARoutes(localRouter, nodeDefs, executors, "http://localhost")
	localServer := httptest.NewServer(localRouter)
	defer localServer.Close()

	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(engine.NewEventBus(), engine.NewSessionManager(), a2aClient)

	nodeURLs := map[string]string{
		"input":     localServer.URL + "/a2a/nodes/input",
		"translate": externalServer.URL,
		"output":    localServer.URL + "/a2a/nodes/output",
	}

	sess, err := a2aRunner.Run(context.Background(), wf, nodeURLs, map[string]any{"input": "hello"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != engine.SessionCompleted {
		t.Errorf("status: got %q", sess.Status)
	}
	// Verify external node's artifacts are stored
	arts := sess.Artifacts["translate"]
	if len(arts) == 0 {
		t.Fatal("expected artifacts for translate node")
	}
	if arts[0].FirstText() != "translated: hello" {
		t.Errorf("translate output: got %q", arts[0].FirstText())
	}
}

func TestIntegration_TemplateResolution(t *testing.T) {
	var agentReceivedPrompt string

	executor := &mockAgentExecutor{}
	localRouter := chi.NewRouter()

	wf := &engine.WorkflowDefinition{
		Name: "template-test",
		Nodes: []engine.NodeDefinition{
			{ID: "input", Type: engine.NodeTypeInput},
			{ID: "agent", Type: engine.NodeTypeAgent, Config: map[string]any{
				"prompt": "Summarize this: {{input}}",
			}},
		},
		Edges: []engine.EdgeDefinition{{From: "input", To: "agent"}},
	}

	// Custom handler that captures the received message
	localRouter.Post("/a2a/nodes/{nodeID}", func(w http.ResponseWriter, r *http.Request) {
		nodeID := chi.URLParam(r, "nodeID")
		// Read full body so we can decode it AND replay it for the handler
		bodyBytes, _ := io.ReadAll(r.Body)
		var req a2atypes.JSONRPCRequest
		json.Unmarshal(bodyBytes, &req)
		paramsData, _ := json.Marshal(req.Params)
		var params a2atypes.SendMessageParams
		json.Unmarshal(paramsData, &params)
		if nodeID == "agent" {
			for _, p := range params.Message.Parts {
				if p.Type == "text" {
					agentReceivedPrompt = p.Text
					break
				}
			}
		}
		// Replay body for the handler
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		// Use mock executor
		nodeDef := &wf.Nodes[0]
		for i := range wf.Nodes {
			if wf.Nodes[i].ID == nodeID {
				nodeDef = &wf.Nodes[i]
				break
			}
		}
		handler := NewNodeHandler(executor, nodeDef)
		handler.ServeHTTP(w, r)
	})
	localServer := httptest.NewServer(localRouter)
	defer localServer.Close()

	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(engine.NewEventBus(), engine.NewSessionManager(), a2aClient)

	nodeURLs := map[string]string{
		"input": localServer.URL + "/a2a/nodes/input",
		"agent": localServer.URL + "/a2a/nodes/agent",
	}

	_, err := a2aRunner.Run(context.Background(), wf, nodeURLs, map[string]any{"input": "hello world"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	expected := "Summarize this: hello world"
	if agentReceivedPrompt != expected {
		t.Errorf("agent prompt: got %q, want %q", agentReceivedPrompt, expected)
	}
}
