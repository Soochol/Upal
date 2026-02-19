package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/engine"
)

func TestClient_SendMessage(t *testing.T) {
	exec := &mockExecutor{result: "agent response"}
	def := &engine.NodeDefinition{ID: "responder", Type: engine.NodeTypeAgent, Config: map[string]any{}}
	handler := NewNodeHandler(exec, def)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewClient(srv.Client())
	msg := Message{
		Role:  "user",
		Parts: []Part{TextPart("hello agent")},
	}

	task, err := client.SendMessage(context.Background(), srv.URL, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != TaskCompleted {
		t.Errorf("expected status %q, got %q", TaskCompleted, task.Status)
	}
	if len(task.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(task.Artifacts))
	}
	if task.Artifacts[0].FirstText() != "agent response" {
		t.Errorf("expected artifact text %q, got %q", "agent response", task.Artifacts[0].FirstText())
	}
	if len(task.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(task.Messages))
	}
	if task.Messages[0].Role != "user" {
		t.Errorf("expected first message role %q, got %q", "user", task.Messages[0].Role)
	}
	if task.Messages[1].Role != "agent" {
		t.Errorf("expected second message role %q, got %q", "agent", task.Messages[1].Role)
	}
}

func TestClient_SendMessage_ServerError(t *testing.T) {
	// Mock server that returns a JSON-RPC error response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error:   &JSONRPCError{Code: -32600, Message: "Invalid Request"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	msg := Message{
		Role:  "user",
		Parts: []Part{TextPart("trigger error")},
	}

	_, err := client.SendMessage(context.Background(), srv.URL, msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "a2a error -32600: Invalid Request"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestClient_SendMessage_NetworkError(t *testing.T) {
	client := NewClient(&http.Client{})
	msg := Message{
		Role:  "user",
		Parts: []Part{TextPart("hello")},
	}

	// Connect to an unreachable address.
	_, err := client.SendMessage(context.Background(), "http://localhost:1/a2a", msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestClient_SendMessage_FailedTask(t *testing.T) {
	exec := &mockExecutor{err: errors.New("execution failed")}
	def := &engine.NodeDefinition{ID: "failing", Type: engine.NodeTypeAgent, Config: map[string]any{}}
	handler := NewNodeHandler(exec, def)

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client := NewClient(srv.Client())
	msg := Message{
		Role:  "user",
		Parts: []Part{TextPart("do something")},
	}

	task, err := client.SendMessage(context.Background(), srv.URL, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v (failed task should not be a client error)", err)
	}

	if task.Status != TaskFailed {
		t.Errorf("expected status %q, got %q", TaskFailed, task.Status)
	}
	if len(task.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(task.Messages))
	}
	if task.Messages[1].Parts[0].Text != "Error: execution failed" {
		t.Errorf("expected error text %q, got %q", "Error: execution failed", task.Messages[1].Parts[0].Text)
	}
	if len(task.Artifacts) != 0 {
		t.Errorf("expected 0 artifacts for failed task, got %d", len(task.Artifacts))
	}
}
