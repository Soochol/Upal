package a2a

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/soochol/upal/internal/engine"
)

// NodeHandler wraps a NodeExecutorInterface as a JSON-RPC HTTP handler
// that speaks the A2A protocol (a2a.sendMessage).
type NodeHandler struct {
	executor engine.NodeExecutorInterface
	nodeDef  *engine.NodeDefinition
	state    map[string]any
}

// NewNodeHandler creates a NodeHandler for the given executor and node definition.
func NewNodeHandler(executor engine.NodeExecutorInterface, nodeDef *engine.NodeDefinition) *NodeHandler {
	return &NodeHandler{executor: executor, nodeDef: nodeDef}
}

// SetState allows external injection of state that will be merged into
// the execution state on each request.
func (h *NodeHandler) SetState(state map[string]any) {
	h.state = state
}

// ServeHTTP dispatches incoming JSON-RPC requests by method name.
func (h *NodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req JSONRPCRequest
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

func (h *NodeHandler) handleSendMessage(w http.ResponseWriter, r *http.Request, req *JSONRPCRequest) {
	paramsData, err := json.Marshal(req.Params)
	if err != nil {
		writeJSONRPCError(w, req.ID, -32602, "Invalid params")
		return
	}
	var params SendMessageParams
	if err := json.Unmarshal(paramsData, &params); err != nil {
		writeJSONRPCError(w, req.ID, -32602, "Invalid params")
		return
	}

	// Build execution state: start with injected state, then overlay message parts.
	state := make(map[string]any)
	if h.state != nil {
		for k, v := range h.state {
			state[k] = v
		}
	}
	for _, part := range params.Message.Parts {
		if part.Type == "text" {
			state["__user_input__"+h.nodeDef.ID] = part.Text
			state["__a2a_message__"] = part.Text
			break
		}
	}

	task := NewTask("")
	task.Status = TaskWorking
	task.Messages = append(task.Messages, params.Message)

	result, err := h.executor.Execute(r.Context(), h.nodeDef, state)
	if err != nil {
		task.Status = TaskFailed
		task.Messages = append(task.Messages, Message{
			Role:  "agent",
			Parts: []Part{TextPart("Error: " + err.Error())},
		})
		writeJSONRPCResponse(w, req.ID, task)
		return
	}

	artifact := resultToArtifact(result)
	task.Artifacts = []Artifact{artifact}
	task.Status = TaskCompleted
	task.Messages = append(task.Messages, Message{Role: "agent", Parts: artifact.Parts})
	writeJSONRPCResponse(w, req.ID, task)
}

// resultToArtifact converts an executor result into an A2A Artifact.
func resultToArtifact(result any) Artifact {
	switch v := result.(type) {
	case string:
		return Artifact{Parts: []Part{TextPart(v)}, Index: 0}
	case map[string]any:
		return Artifact{Parts: []Part{DataPart(v, "application/json")}, Index: 0}
	case []any:
		return Artifact{Parts: []Part{DataPart(v, "application/json")}, Index: 0}
	default:
		return Artifact{Parts: []Part{TextPart(fmt.Sprintf("%v", v))}, Index: 0}
	}
}

func writeJSONRPCResponse(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func writeJSONRPCError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &JSONRPCError{Code: code, Message: message}})
}

// ---------------------------------------------------------------------------
// AgentCard handler
// ---------------------------------------------------------------------------

// AgentCardFromNodeDef builds an AgentCard describing an A2A agent backed
// by the given node definition.
func AgentCardFromNodeDef(def *engine.NodeDefinition, baseURL string) AgentCard {
	description := fmt.Sprintf("Upal %s node: %s", def.Type, def.ID)
	if sp, ok := def.Config["system_prompt"].(string); ok && sp != "" {
		description = sp
	}
	return AgentCard{
		Name:        def.ID,
		Description: description,
		URL:         fmt.Sprintf("%s/a2a/nodes/%s", baseURL, def.ID),
		Capabilities: Capabilities{
			Streaming:         false,
			PushNotifications: false,
		},
		Skills: []Skill{{
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
	card AgentCard
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
