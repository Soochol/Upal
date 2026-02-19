package nodes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

func TestExternalNode_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req a2atypes.JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		task := a2atypes.Task{
			ID:     "task-1",
			Status: a2atypes.TaskCompleted,
			Artifacts: []a2atypes.Artifact{{
				Parts: []a2atypes.Part{a2atypes.TextPart("external response")},
				Index: 0,
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0", ID: req.ID, Result: task,
		})
	}))
	defer server.Close()

	client := a2aclient.NewClient(http.DefaultClient)
	node := NewExternalNode(client)
	def := &engine.NodeDefinition{
		ID:   "ext1",
		Type: engine.NodeTypeExternal,
		Config: map[string]any{
			"endpoint_url": server.URL,
		},
	}
	state := map[string]any{
		"__a2a_message__": "translate this",
	}
	result, err := node.Execute(context.Background(), def, state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	text, ok := result.(string)
	if !ok || text != "external response" {
		t.Errorf("result: got %v", result)
	}
}

func TestExternalNode_MissingURL(t *testing.T) {
	client := a2aclient.NewClient(http.DefaultClient)
	node := NewExternalNode(client)
	def := &engine.NodeDefinition{
		ID:     "ext1",
		Type:   engine.NodeTypeExternal,
		Config: map[string]any{},
	}
	_, err := node.Execute(context.Background(), def, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing endpoint_url")
	}
}
