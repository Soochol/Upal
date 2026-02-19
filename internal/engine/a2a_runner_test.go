package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
)

func TestA2ARunner_LinearChain(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{{From: "a", To: "b"}, {From: "b", To: "c"}},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		parts := strings.Split(r.URL.Path, "/")
		nodeID := parts[len(parts)-1]
		task := map[string]any{
			"id":     "task-1",
			"status": "completed",
			"artifacts": []map[string]any{{
				"parts": []map[string]any{{
					"type": "text", "text": "output of " + nodeID, "mimeType": "text/plain",
				}},
				"index": 0,
			}},
		}
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": req["id"], "result": task,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"a": server.URL + "/a2a/nodes/a",
		"b": server.URL + "/a2a/nodes/b",
		"c": server.URL + "/a2a/nodes/c",
	}
	sess, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"a": "input data"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != SessionCompleted {
		t.Errorf("status: got %q, want %q", sess.Status, SessionCompleted)
	}
	if len(sess.Artifacts) < 3 {
		t.Errorf("artifacts: got %d nodes, want >= 3", len(sess.Artifacts))
	}
}

func TestA2ARunner_FanOutFanIn(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "fan-test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeAgent},
			{ID: "d", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{
			{From: "a", To: "b"}, {From: "a", To: "c"},
			{From: "b", To: "d"}, {From: "c", To: "d"},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		parts := strings.Split(r.URL.Path, "/")
		nodeID := parts[len(parts)-1]
		task := map[string]any{
			"id": "task-1", "status": "completed",
			"artifacts": []map[string]any{{
				"parts": []map[string]any{{"type": "text", "text": "output of " + nodeID, "mimeType": "text/plain"}},
				"index": 0,
			}},
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": task})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"a": server.URL + "/a2a/nodes/a",
		"b": server.URL + "/a2a/nodes/b",
		"c": server.URL + "/a2a/nodes/c",
		"d": server.URL + "/a2a/nodes/d",
	}
	sess, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"a": "input"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != SessionCompleted {
		t.Errorf("status: got %q", sess.Status)
	}
}

func TestA2ARunner_MissingNodeURL(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "test",
		Nodes: []NodeDefinition{{ID: "a", Type: NodeTypeInput}},
	}
	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	_, err := runner.Run(context.Background(), wf, map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestA2ARunner_TypedArtifacts(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "typed-test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
		},
		Edges: []EdgeDefinition{{From: "a", To: "b"}},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		parts := strings.Split(r.URL.Path, "/")
		nodeID := parts[len(parts)-1]

		var text string
		if nodeID == "a" {
			text = "input value"
		} else {
			text = "typed output"
		}

		task := map[string]any{
			"id":     "task-1",
			"status": "completed",
			"artifacts": []map[string]any{{
				"parts": []map[string]any{{
					"type": "text", "text": text, "mimeType": "text/plain",
				}},
				"index": 0,
			}},
		}
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": req["id"], "result": task,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"a": server.URL + "/a2a/nodes/a",
		"b": server.URL + "/a2a/nodes/b",
	}

	sess, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"a": "hello"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sess.Status != SessionCompleted {
		t.Errorf("status: got %q, want %q", sess.Status, SessionCompleted)
	}

	// Verify typed artifact access on node "b"
	arts, ok := sess.Artifacts["b"]
	if !ok || len(arts) == 0 {
		t.Fatal("expected artifacts for node b")
	}
	got := arts[0].FirstText()
	if got != "typed output" {
		t.Errorf("artifact text: got %q, want %q", got, "typed output")
	}

	// Verify legacy state compat
	state, _ := sess.State["b"]
	if state != "typed output" {
		t.Errorf("state: got %v, want %q", state, "typed output")
	}
}

func TestA2ARunner_ErrorCancelsDownstream(t *testing.T) {
	// 3-node chain: a → b → c
	// b fails, c should NOT execute
	wf := &WorkflowDefinition{
		Name: "cancel-test",
		Nodes: []NodeDefinition{
			{ID: "a", Type: NodeTypeInput},
			{ID: "b", Type: NodeTypeAgent},
			{ID: "c", Type: NodeTypeOutput},
		},
		Edges: []EdgeDefinition{{From: "a", To: "b"}, {From: "b", To: "c"}},
	}
	var cExecuted bool
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		nodeID := parts[len(parts)-1]
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if nodeID == "b" {
			// Node b fails
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0", "id": req["id"],
				"error": map[string]any{"code": -32000, "message": "node b failed"},
			})
			return
		}
		if nodeID == "c" {
			cExecuted = true
		}
		task := map[string]any{
			"id": "task-1", "status": "completed",
			"artifacts": []map[string]any{{
				"parts": []map[string]any{{"type": "text", "text": "ok", "mimeType": "text/plain"}},
				"index": 0,
			}},
		}
		json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": task})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"a": server.URL + "/a2a/nodes/a",
		"b": server.URL + "/a2a/nodes/b",
		"c": server.URL + "/a2a/nodes/c",
	}
	sess, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"a": "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if sess.Status != SessionFailed {
		t.Errorf("status: got %q, want failed", sess.Status)
	}
	if cExecuted {
		t.Error("node c should NOT have executed after b failed")
	}
}

func TestA2ARunner_TemplateResolution(t *testing.T) {
	wf := &WorkflowDefinition{
		Name: "template-test",
		Nodes: []NodeDefinition{
			{ID: "input1", Type: NodeTypeInput},
			{ID: "agent1", Type: NodeTypeAgent, Config: map[string]any{
				"prompt": "Summarize: {{input1}}",
			}},
		},
		Edges: []EdgeDefinition{{From: "input1", To: "agent1"}},
	}

	var receivedMessage string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		// Extract message text from params
		paramsData, _ := json.Marshal(req.Params)
		var params a2atypes.SendMessageParams
		json.Unmarshal(paramsData, &params)
		for _, p := range params.Message.Parts {
			if p.Type == "text" {
				receivedMessage = p.Text
				break
			}
		}

		task := a2atypes.Task{
			ID: "task-1", Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("summary output")},
				Index: 0,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: task})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), client)
	nodeURLs := map[string]string{
		"input1": server.URL + "/a2a/nodes/input1",
		"agent1": server.URL + "/a2a/nodes/agent1",
	}
	_, err := runner.Run(context.Background(), wf, nodeURLs, map[string]any{"input1": "hello world"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The agent node's prompt should have been resolved with input1's artifact text
	expected := "Summarize: hello world"
	if receivedMessage != expected {
		t.Errorf("message: got %q, want %q", receivedMessage, expected)
	}
}
