package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), http.DefaultClient)
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

	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), http.DefaultClient)
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
	runner := NewA2ARunner(NewEventBus(), NewSessionManager(), http.DefaultClient)
	_, err := runner.Run(context.Background(), wf, map[string]string{}, nil)
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}
