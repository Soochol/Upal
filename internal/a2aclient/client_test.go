package a2aclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/a2atypes"
)

func TestSendMessage_Success(t *testing.T) {
	// Mock JSON-RPC server that returns a completed task.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format.
		var req a2atypes.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Method != "a2a.sendMessage" {
			t.Errorf("expected method %q, got %q", "a2a.sendMessage", req.Method)
		}

		task := a2atypes.Task{
			ID:     "task-123",
			Status: a2atypes.TaskCompleted,
			Messages: []a2atypes.Message{
				{Role: "user", Parts: []a2atypes.Part{a2atypes.TextPart("hello")}},
				{Role: "agent", Parts: []a2atypes.Part{a2atypes.TextPart("world")}},
			},
			Artifacts: []a2atypes.Artifact{
				{Parts: []a2atypes.Part{a2atypes.TextPart("result text")}, Index: 0},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  task,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	msg := a2atypes.Message{
		Role:  "user",
		Parts: []a2atypes.Part{a2atypes.TextPart("hello")},
	}

	task, err := client.SendMessage(context.Background(), srv.URL, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.Status != a2atypes.TaskCompleted {
		t.Errorf("expected status %q, got %q", a2atypes.TaskCompleted, task.Status)
	}
	if len(task.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(task.Artifacts))
	}
	if task.Artifacts[0].FirstText() != "result text" {
		t.Errorf("expected artifact text %q, got %q", "result text", task.Artifacts[0].FirstText())
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

func TestSendMessage_ServerError(t *testing.T) {
	// Mock server that returns a JSON-RPC error response.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Error:   &a2atypes.JSONRPCError{Code: -32600, Message: "Invalid Request"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.Client())
	msg := a2atypes.Message{
		Role:  "user",
		Parts: []a2atypes.Part{a2atypes.TextPart("trigger error")},
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

func TestSendMessage_NetworkError(t *testing.T) {
	client := NewClient(&http.Client{})
	msg := a2atypes.Message{
		Role:  "user",
		Parts: []a2atypes.Part{a2atypes.TextPart("hello")},
	}

	// Connect to an unreachable address.
	_, err := client.SendMessage(context.Background(), "http://localhost:1/a2a", msg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
