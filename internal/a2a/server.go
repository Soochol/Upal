package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/engine"
)

// NodeHandler wraps a NodeExecutorInterface as a JSON-RPC HTTP handler
// that speaks the A2A protocol (a2a.sendMessage).
type NodeHandler struct {
	executor engine.NodeExecutorInterface
	nodeDef  *engine.NodeDefinition
}

// NewNodeHandler creates a NodeHandler for the given executor and node definition.
func NewNodeHandler(executor engine.NodeExecutorInterface, nodeDef *engine.NodeDefinition) *NodeHandler {
	return &NodeHandler{executor: executor, nodeDef: nodeDef}
}

// ServeHTTP dispatches incoming JSON-RPC requests by method name.
func (h *NodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req a2atypes.JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error")
		return
	}

	switch req.Method {
	case "a2a.sendMessage":
		h.handleSendMessage(w, r, &req)
	default:
		writeJSONRPCError(w, req.ID, -32601, "Method not found")
	}
}

func (h *NodeHandler) handleSendMessage(w http.ResponseWriter, r *http.Request, req *a2atypes.JSONRPCRequest) {
	paramsData, err := json.Marshal(req.Params)
	if err != nil {
		writeJSONRPCError(w, req.ID, -32602, "Invalid params")
		return
	}
	var params a2atypes.SendMessageParams
	if err := json.Unmarshal(paramsData, &params); err != nil {
		writeJSONRPCError(w, req.ID, -32602, "Invalid params")
		return
	}

	// Build execution state from message parts.
	state := make(map[string]any)
	for _, part := range params.Message.Parts {
		if part.Type == "text" {
			state["__user_input__"+h.nodeDef.ID] = part.Text
			state["__a2a_message__"] = part.Text
			break
		}
	}

	task := a2atypes.NewTask("")
	task.Status = a2atypes.TaskWorking
	task.Messages = append(task.Messages, params.Message)

	result, err := h.executor.Execute(r.Context(), h.nodeDef, state)
	if err != nil {
		task.Status = a2atypes.TaskFailed
		task.Messages = append(task.Messages, a2atypes.Message{
			Role:  "agent",
			Parts: []a2atypes.Part{a2atypes.TextPart("Error: " + err.Error())},
		})
		writeJSONRPCResponse(w, req.ID, task)
		return
	}

	artifact := resultToArtifact(result)
	task.Artifacts = []a2atypes.Artifact{artifact}
	task.Status = a2atypes.TaskCompleted
	task.Messages = append(task.Messages, a2atypes.Message{Role: "agent", Parts: artifact.Parts})
	writeJSONRPCResponse(w, req.ID, task)
}

// resultToArtifact converts an executor result into an A2A Artifact.
func resultToArtifact(result any) a2atypes.Artifact {
	switch v := result.(type) {
	case string:
		return a2atypes.Artifact{Parts: []a2atypes.Part{a2atypes.TextPart(v)}, Index: 0}
	case map[string]any:
		return a2atypes.Artifact{Parts: []a2atypes.Part{a2atypes.DataPart(v, "application/json")}, Index: 0}
	case []any:
		return a2atypes.Artifact{Parts: []a2atypes.Part{a2atypes.DataPart(v, "application/json")}, Index: 0}
	default:
		return a2atypes.Artifact{Parts: []a2atypes.Part{a2atypes.TextPart(fmt.Sprintf("%v", v))}, Index: 0}
	}
}

func writeJSONRPCResponse(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func writeJSONRPCError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &a2atypes.JSONRPCError{Code: code, Message: message}})
}

// ---------------------------------------------------------------------------
// AgentCard handler
// ---------------------------------------------------------------------------

// AgentCardFromNodeDef builds an AgentCard describing an A2A agent backed
// by the given node definition.
func AgentCardFromNodeDef(def *engine.NodeDefinition, baseURL string) a2atypes.AgentCard {
	description := fmt.Sprintf("Upal %s node: %s", def.Type, def.ID)
	if sp, ok := def.Config["system_prompt"].(string); ok && sp != "" {
		description = sp
	}
	return a2atypes.AgentCard{
		Name:        def.ID,
		Description: description,
		URL:         fmt.Sprintf("%s/a2a/nodes/%s", baseURL, def.ID),
		Capabilities: a2atypes.Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []a2atypes.Skill{{
			ID:          def.ID,
			Name:        string(def.Type) + ": " + def.ID,
			Description: description,
			InputModes:  []string{"text/plain"},
			OutputModes: []string{"text/plain", "application/json"},
		}},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}
}

// AgentCardHandler serves the AgentCard as JSON over HTTP GET.
type AgentCardHandler struct {
	card a2atypes.AgentCard
}

// NewAgentCardHandler creates an AgentCardHandler for the given node definition.
func NewAgentCardHandler(def *engine.NodeDefinition, baseURL string) *AgentCardHandler {
	return &AgentCardHandler{card: AgentCardFromNodeDef(def, baseURL)}
}

// ServeHTTP writes the AgentCard as JSON.
func (h *AgentCardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.card)
}
